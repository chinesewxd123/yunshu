package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/apperror"
	"yunshu/internal/repository"
)

type MenuService struct {
	menuRepo          *repository.MenuRepository
	mu                sync.RWMutex
	treeCache         []model.Menu
	treeCacheExpireAt time.Time
	maintenanceDone   bool
}

// NewMenuService 创建相关逻辑。
func NewMenuService(menuRepo *repository.MenuRepository) *MenuService {
	return &MenuService{menuRepo: menuRepo}
}

func sameParent(a, b *uint) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// ensureUniqueSiblingSort keeps sort unique within same parent scope.
// If requested sort is occupied, it shifts to next available value.
func (s *MenuService) ensureUniqueSiblingSort(ctx context.Context, parentID *uint, sort int, excludeID uint) (int, error) {
	all, err := s.menuRepo.ListAll(ctx)
	if err != nil {
		return sort, err
	}
	used := make(map[int]struct{}, 64)
	for _, it := range all {
		if excludeID > 0 && it.ID == excludeID {
			continue
		}
		if !sameParent(parentID, it.ParentID) {
			continue
		}
		used[it.Sort] = struct{}{}
	}
	for {
		if _, ok := used[sort]; !ok {
			return sort, nil
		}
		sort++
	}
}

type MenuCreatePayload struct {
	ParentID  *uint  `json:"parent_id"`
	Path      string `json:"path"`
	Name      string `json:"name" binding:"required,max=64"`
	Icon      string `json:"icon"`
	AdminOnly bool   `json:"admin_only"`
	Sort      int    `json:"sort"`
	Hidden    bool   `json:"hidden"`
	Component string `json:"component"`
	Redirect  string `json:"redirect"`
	Status    int    `json:"status" binding:"required,oneof=0 1"`
}

type MenuUpdatePayload struct {
	ParentID  *uint  `json:"parent_id"`
	Path      string `json:"path"`
	Name      string `json:"name" binding:"omitempty,max=64"`
	Icon      string `json:"icon"`
	AdminOnly *bool  `json:"admin_only,omitempty"`
	Sort      int    `json:"sort"`
	Hidden    bool   `json:"hidden"`
	Component string `json:"component"`
	Redirect  string `json:"redirect"`
	// 使用指针避免 JSON 省略时误把 status 写成 0（原逻辑 `id == menu.ID` 恒为真会错误停用菜单）
	Status *int `json:"status,omitempty" binding:"omitempty,oneof=0 1"`
}

type MenuBatchStatusPayload struct {
	IDs    []uint `json:"ids" binding:"required,min=1,dive,gt=0"`
	Status int    `json:"status" binding:"oneof=0 1"`
}

// Tree 获取树形数据相关的业务逻辑。
func (s *MenuService) Tree(ctx context.Context) ([]model.Menu, error) {
	s.mu.RLock()
	if time.Now().Before(s.treeCacheExpireAt) && s.treeCache != nil {
		cached := s.treeCache
		s.mu.RUnlock()
		return cached, nil
	}
	done := s.maintenanceDone
	s.mu.RUnlock()

	list, err := s.menuRepo.Tree(ctx)
	if err != nil {
		return nil, err
	}

	needRefresh := false
	if !done {
		changed, err := s.ensureBuiltInMenus(ctx, list)
		if err != nil {
			return nil, err
		}
		repaired, err := s.repairEventMenus(ctx)
		if err != nil {
			return nil, err
		}
		normalized, err := s.normalizeMenuHierarchyAndSort(ctx)
		if err != nil {
			return nil, err
		}
		hiddenAdjusted, err := s.ensureHiddenMenusDisabled(ctx)
		if err != nil {
			return nil, err
		}
		needRefresh = changed || repaired || normalized
		needRefresh = needRefresh || hiddenAdjusted
		s.mu.Lock()
		s.maintenanceDone = true
		s.mu.Unlock()
	}
	if needRefresh {
		list, err = s.menuRepo.Tree(ctx)
		if err != nil {
			return nil, err
		}
	}

	s.mu.Lock()
	s.treeCache = list
	s.treeCacheExpireAt = time.Now().Add(3 * time.Second)
	s.mu.Unlock()
	return list, nil
}

// normalizeMenuHierarchyAndSort repairs two common data issues:
// 1) menu path implies parent path but parent_id is missing/wrong
// 2) sibling sort collisions / disorder
func (s *MenuService) normalizeMenuHierarchyAndSort(ctx context.Context) (bool, error) {
	all, err := s.menuRepo.ListAll(ctx)
	if err != nil {
		return false, err
	}
	changed := false

	pathMap := make(map[string]*model.Menu, len(all))
	for i := range all {
		p := strings.TrimSpace(all[i].Path)
		if p != "" {
			pathMap[p] = &all[i]
		}
	}

	// Repair parent_id by route path hierarchy: /a/b/c -> parent /a/b
	for i := range all {
		m := &all[i]
		p := strings.TrimSpace(m.Path)
		if p == "" || p == "/" {
			continue
		}
		segs := strings.Split(strings.Trim(p, "/"), "/")
		if len(segs) <= 1 {
			continue
		}
		parentPath := "/" + strings.Join(segs[:len(segs)-1], "/")
		parent, ok := pathMap[parentPath]
		if !ok || parent.ID == m.ID {
			continue
		}
		if m.ParentID == nil || *m.ParentID != parent.ID {
			pid := parent.ID
			m.ParentID = &pid
			if err := s.menuRepo.Update(ctx, m); err != nil {
				return false, err
			}
			changed = true
		}
	}

	// Reload after possible parent updates.
	all, err = s.menuRepo.ListAll(ctx)
	if err != nil {
		return false, err
	}

	siblingMap := make(map[uint][]*model.Menu)
	var roots []*model.Menu
	for i := range all {
		m := &all[i]
		if m.ParentID == nil {
			roots = append(roots, m)
			continue
		}
		siblingMap[*m.ParentID] = append(siblingMap[*m.ParentID], m)
	}

	sortSibling := func(items []*model.Menu) {
		sort.Slice(items, func(i, j int) bool {
			if items[i].Sort != items[j].Sort {
				return items[i].Sort < items[j].Sort
			}
			return items[i].ID < items[j].ID
		})
	}

	sortSibling(roots)
	for idx, m := range roots {
		target := idx + 1
		if m.Sort != target {
			m.Sort = target
			if err := s.menuRepo.Update(ctx, m); err != nil {
				return false, err
			}
			changed = true
		}
	}

	for _, items := range siblingMap {
		sortSibling(items)
		for idx, m := range items {
			target := idx + 1
			if m.Sort != target {
				m.Sort = target
				if err := s.menuRepo.Update(ctx, m); err != nil {
					return false, err
				}
				changed = true
			}
		}
	}
	return changed, nil
}

// ensureHiddenMenusDisabled keeps data consistent: hidden menu must be status=0.
func (s *MenuService) ensureHiddenMenusDisabled(ctx context.Context) (bool, error) {
	all, err := s.menuRepo.ListAll(ctx)
	if err != nil {
		return false, err
	}
	changed := false
	for i := range all {
		m := &all[i]
		if !m.Hidden || m.Status == 0 {
			continue
		}
		m.Status = 0
		if err := s.menuRepo.Update(ctx, m); err != nil {
			return false, err
		}
		changed = true
	}
	return changed, nil
}

// repairEventMenus 修正误配的 K8s Event 菜单（path/component 与 Webhook 告警事件混淆时）。
func (s *MenuService) repairEventMenus(ctx context.Context) (bool, error) {
	all, err := s.menuRepo.ListAll(ctx)
	if err != nil {
		return false, err
	}
	changed := false
	for i := range all {
		m := &all[i]
		p := strings.TrimSpace(m.Path)
		name := strings.TrimSpace(m.Name)
		comp := strings.TrimSpace(m.Component)

		needSave := false
		// 名称是 K8s Event，但 path 误指到告警事件页
		if name == "Event 事件" && p == "/alert-events" {
			m.Path = "/events"
			needSave = true
		}
		p = strings.TrimSpace(m.Path)
		if p == "/events" {
			if comp != "events-page" {
				m.Component = "events-page"
				needSave = true
			}
			// 与「Webhook 告警通道」区分，避免同图标误点
			if strings.TrimSpace(m.Icon) == "NotificationOutlined" {
				m.Icon = "FileSearchOutlined"
				needSave = true
			}
		}
		if needSave {
			if err := s.menuRepo.Update(ctx, m); err != nil {
				return false, err
			}
			changed = true
		}
	}
	return changed, nil
}

func (s *MenuService) ensureBuiltInMenus(ctx context.Context, tree []model.Menu) (bool, error) {
	changed := false

	k8sChanged, err := s.ensureK8sMenus(ctx, tree)
	if err != nil {
		return false, err
	}
	if k8sChanged {
		changed = true
	}

	alertChanged, err := s.ensureAlertNotifyMenus(ctx)
	if err != nil {
		return false, err
	}
	if alertChanged {
		changed = true
	}

	projectChanged, err := s.ensureProjectMgmtMenus(ctx)
	if err != nil {
		return false, err
	}
	if projectChanged {
		changed = true
	}

	orgChanged, err := s.ensureOrgMenus(ctx)
	if err != nil {
		return false, err
	}
	if orgChanged {
		changed = true
	}

	systemChanged, err := s.ensureSystemMenus(ctx)
	if err != nil {
		return false, err
	}
	if systemChanged {
		changed = true
	}

	return changed, nil
}

func (s *MenuService) ensureOrgMenus(ctx context.Context) (bool, error) {
	all, err := s.menuRepo.ListAll(ctx)
	if err != nil {
		return false, err
	}
	required := []model.Menu{
		{Path: "/users", Name: "账号管理", Icon: "TeamOutlined", Sort: 4, Component: "users-page", Status: 1},
	}
	changed := false
	for _, spec := range required {
		var found *model.Menu
		for i := range all {
			if strings.TrimSpace(all[i].Path) == spec.Path {
				found = &all[i]
				break
			}
		}
		if found == nil {
			m := spec
			if err := s.menuRepo.Create(ctx, &m); err != nil {
				return false, err
			}
			changed = true
			continue
		}
		needSave := false
		if strings.TrimSpace(found.Name) != spec.Name {
			found.Name = spec.Name
			needSave = true
		}
		if strings.TrimSpace(found.Icon) != spec.Icon {
			found.Icon = spec.Icon
			needSave = true
		}
		if found.Sort != spec.Sort {
			found.Sort = spec.Sort
			needSave = true
		}
		if strings.TrimSpace(found.Component) != spec.Component {
			found.Component = spec.Component
			needSave = true
		}
		if found.Status != spec.Status {
			found.Status = spec.Status
			needSave = true
		}
		if needSave {
			if err := s.menuRepo.Update(ctx, found); err != nil {
				return false, err
			}
			changed = true
		}
	}
	return changed, nil
}

func (s *MenuService) ensureSystemMenus(ctx context.Context) (bool, error) {
	all, err := s.menuRepo.ListAll(ctx)
	if err != nil {
		return false, err
	}
	changed := false

	const rootPath = "/system"
	var root *model.Menu
	for i := range all {
		if strings.TrimSpace(all[i].Path) == rootPath {
			root = &all[i]
			break
		}
	}
	if root == nil {
		m := model.Menu{Name: "系统管理", Path: rootPath, Icon: "MenuOutlined", Sort: 6, Status: 1}
		if err := s.menuRepo.Create(ctx, &m); err != nil {
			return false, err
		}
		root = &m
		changed = true
	}

	rootID := root.ID
	requiredChildren := []model.Menu{
		{Path: "/departments", Name: "组织架构", Icon: "ApartmentOutlined", Sort: 1, Component: "departments-page", Status: 1},
		{Path: "/dict-entries", Name: "数据字典", Icon: "DatabaseOutlined", Sort: 2, Component: "dict-entries-page", Status: 1},
		{Path: "/login-logs", Name: "登录日志", Icon: "LoginOutlined", Sort: 3, Component: "login-logs-page", Status: 1},
		{Path: "/operation-logs", Name: "操作历史", Icon: "HistoryOutlined", Sort: 4, Component: "operation-logs-page", Status: 1},
		{Path: "/banned-ips", Name: "封禁 IP 管理", Icon: "ApiOutlined", Sort: 5, Component: "banned-ips-page", Status: 1},
	}
	for _, spec := range requiredChildren {
		var found *model.Menu
		for i := range all {
			if strings.TrimSpace(all[i].Path) == spec.Path {
				found = &all[i]
				break
			}
		}
		if found == nil {
			m := spec
			m.ParentID = &rootID
			if err := s.menuRepo.Create(ctx, &m); err != nil {
				return false, err
			}
			changed = true
			continue
		}
		needSave := false
		if found.ParentID == nil || *found.ParentID != rootID {
			pid := rootID
			found.ParentID = &pid
			needSave = true
		}
		if strings.TrimSpace(found.Name) != spec.Name {
			found.Name = spec.Name
			needSave = true
		}
		if strings.TrimSpace(found.Icon) != spec.Icon {
			found.Icon = spec.Icon
			needSave = true
		}
		if found.Sort != spec.Sort {
			found.Sort = spec.Sort
			needSave = true
		}
		if strings.TrimSpace(found.Component) != spec.Component {
			found.Component = spec.Component
			needSave = true
		}
		if found.Status != spec.Status {
			found.Status = spec.Status
			needSave = true
		}
		if needSave {
			if err := s.menuRepo.Update(ctx, found); err != nil {
				return false, err
			}
			changed = true
		}
	}
	return changed, nil
}

func (s *MenuService) ensureProjectMgmtMenus(ctx context.Context) (bool, error) {
	all, err := s.menuRepo.ListAll(ctx)
	if err != nil {
		return false, err
	}

	const rootPath = "/project-management"
	requiredChildren := []model.Menu{
		{Path: "/projects", Name: "项目列表", Icon: "AppstoreOutlined", Sort: 1, Component: "projects-page", Status: 1},
		{Path: "/project-servers", Name: "服务器管理", Icon: "HddOutlined", Sort: 2, Component: "project-servers-page", Status: 1},
		{Path: "/project-services", Name: "服务配置", Icon: "SettingOutlined", Sort: 3, Component: "project-services-page", Status: 1},
		{Path: "/project-log-sources", Name: "日志源配置", Icon: "FileSearchOutlined", Sort: 4, Component: "project-log-sources-page", Status: 1},
		{Path: "/project-logs", Name: "日志平台", Icon: "FileTextOutlined", Sort: 5, Component: "project-logs-page", Status: 1},
		{Path: "/agent-list", Name: "Agent 列表", Icon: "RobotOutlined", Sort: 6, Component: "agent-list-page", Status: 1},
	}

	var root *model.Menu
	for i := range all {
		if strings.TrimSpace(all[i].Path) == rootPath {
			root = &all[i]
			break
		}
	}

	changed := false
	if root == nil {
		m := model.Menu{Name: "项目管理", Path: rootPath, Icon: "ProjectOutlined", Sort: 4, Status: 1}
		if err := s.menuRepo.Create(ctx, &m); err != nil {
			return false, err
		}
		root = &m
		changed = true
	} else {
		if root.Sort != 4 {
			root.Sort = 4
			if err := s.menuRepo.Update(ctx, root); err != nil {
				return false, err
			}
			changed = true
		}
	}

	rootID := root.ID
	for _, spec := range requiredChildren {
		var found *model.Menu
		for i := range all {
			if strings.TrimSpace(all[i].Path) == spec.Path {
				found = &all[i]
				break
			}
		}

		if found == nil {
			m := spec
			m.ParentID = &rootID
			if err := s.menuRepo.Create(ctx, &m); err != nil {
				return false, err
			}
			changed = true
			continue
		}

		needSave := false
		if found.ParentID == nil || *found.ParentID != rootID {
			p := rootID
			found.ParentID = &p
			needSave = true
		}
		if strings.TrimSpace(found.Name) != spec.Name {
			found.Name = spec.Name
			needSave = true
		}
		if strings.TrimSpace(found.Icon) != spec.Icon {
			found.Icon = spec.Icon
			needSave = true
		}
		if found.Sort != spec.Sort {
			found.Sort = spec.Sort
			needSave = true
		}
		if strings.TrimSpace(found.Component) != spec.Component {
			found.Component = spec.Component
			needSave = true
		}
		if found.Status != spec.Status {
			found.Status = spec.Status
			needSave = true
		}
		if needSave {
			if err := s.menuRepo.Update(ctx, found); err != nil {
				return false, err
			}
			changed = true
		}
	}
	return changed, nil
}

func (s *MenuService) ensureK8sMenus(ctx context.Context, tree []model.Menu) (bool, error) {
	var k8sRoot *model.Menu
	for i := range tree {
		p := strings.TrimSpace(tree[i].Path)
		if p == "/kubernetes" || strings.Contains(strings.ToLower(strings.TrimSpace(tree[i].Name)), "kubernetes") {
			k8sRoot = &tree[i]
			break
		}
	}
	if k8sRoot == nil {
		return false, nil
	}

	existing := make(map[string]bool, len(k8sRoot.Children))
	for _, c := range k8sRoot.Children {
		existing[strings.TrimSpace(c.Path)] = true
	}

	required := []model.Menu{
		{Path: "/nodes", Name: "Node 管理", Icon: "HddOutlined", Sort: 3, Component: "nodes-page", Status: 1},
		{Path: "/component-status", Name: "组件状态", Icon: "HeartOutlined", Sort: 4, Component: "component-status-page", Status: 1},
		{Path: "/k8s-services", Name: "Service 管理", Icon: "ApartmentOutlined", Sort: 13, Component: "k8s-services-page", Status: 1},
		{Path: "/persistentvolumes", Name: "PersistentVolume", Icon: "DatabaseOutlined", Sort: 14, Component: "persistentvolumes-page", Status: 1},
		{Path: "/persistentvolumeclaims", Name: "PersistentVolumeClaim", Icon: "HddOutlined", Sort: 15, Component: "persistentvolumeclaims-page", Status: 1},
		{Path: "/storageclasses", Name: "StorageClass", Icon: "FolderOpenOutlined", Sort: 16, Component: "storageclasses-page", Status: 1},
		{Path: "/ingresses", Name: "Ingress 管理", Icon: "GatewayOutlined", Sort: 17, Component: "ingresses-page", Status: 1},
		{Path: "/ingress-classes", Name: "IngressClass 入口类", Icon: "GatewayOutlined", Sort: 18, Component: "ingress-classes-page", Status: 1},
		{Path: "/network-policies", Name: "网络策略管理", Icon: "DeploymentUnitOutlined", Sort: 19, Component: "network-policies-page", Status: 1},
		{Path: "/rbac/roles", Name: "RBAC - Role", Icon: "SafetyCertificateOutlined", Sort: 20, Component: "rbac-roles-page", Status: 1},
		{Path: "/rbac/rolebindings", Name: "RBAC - RoleBinding", Icon: "SafetyCertificateOutlined", Sort: 21, Component: "rbac-rolebindings-page", Status: 1},
		{Path: "/rbac/clusterroles", Name: "RBAC - ClusterRole", Icon: "SafetyCertificateOutlined", Sort: 22, Component: "rbac-clusterroles-page", Status: 1},
		{Path: "/rbac/clusterrolebindings", Name: "RBAC - ClusterRoleBinding", Icon: "SafetyCertificateOutlined", Sort: 23, Component: "rbac-clusterrolebindings-page", Status: 1},
		{Path: "/k8s-scoped-policies", Name: "K8s 三元策略", Icon: "AuditOutlined", Sort: 24, Component: "k8s-scoped-policies-page", Status: 1},
	}

	changed := false
	parentID := k8sRoot.ID
	for _, r := range required {
		if existing[r.Path] {
			continue
		}
		m := r
		m.ParentID = &parentID
		if err := s.menuRepo.Create(ctx, &m); err != nil {
			return false, err
		}
		changed = true
	}
	return changed, nil
}

// ensureAlertNotifyMenus 保证「告警通知」独立目录及告警子菜单存在，并从「系统管理」下移出（按 path 归位）。
func (s *MenuService) ensureAlertNotifyMenus(ctx context.Context) (bool, error) {
	all, err := s.menuRepo.ListAll(ctx)
	if err != nil {
		return false, err
	}

	const alertRootPath = "/alert-notify"
	requiredChildren := []model.Menu{
		{Path: "/alert-channels", Name: "Webhook 告警通道", Icon: "NotificationOutlined", Sort: 1, Component: "alert-channels-page", Status: 1},
		{Path: "/alert-monitor-platform", Name: "告警监控平台", Icon: "MonitorOutlined", Sort: 2, Component: "alert-monitor-platform-page", Status: 1},
		{Path: "/alert-duty", Name: "值班总览", Icon: "ScheduleOutlined", Sort: 3, Component: "alert-duty-page", Status: 1},
	}

	var root *model.Menu
	for i := range all {
		if strings.TrimSpace(all[i].Path) == alertRootPath {
			root = &all[i]
			break
		}
	}

	changed := false
	if root == nil {
		m := model.Menu{
			Name:   "告警通知",
			Path:   alertRootPath,
			Icon:   "BellOutlined",
			Sort:   2,
			Status: 1,
		}
		if err := s.menuRepo.Create(ctx, &m); err != nil {
			return false, err
		}
		root = &m
		changed = true
	}

	rootID := root.ID

	for _, spec := range requiredChildren {
		var found *model.Menu
		for i := range all {
			if strings.TrimSpace(all[i].Path) == spec.Path {
				found = &all[i]
				break
			}
		}

		if found == nil {
			m := spec
			m.ParentID = &rootID
			if err := s.menuRepo.Create(ctx, &m); err != nil {
				return false, err
			}
			changed = true
			continue
		}

		needSave := false
		if found.ParentID == nil || *found.ParentID != rootID {
			p := rootID
			found.ParentID = &p
			needSave = true
		}
		if strings.TrimSpace(found.Name) != spec.Name {
			found.Name = spec.Name
			needSave = true
		}
		if strings.TrimSpace(found.Icon) != spec.Icon {
			found.Icon = spec.Icon
			needSave = true
		}
		if found.Sort != spec.Sort {
			found.Sort = spec.Sort
			needSave = true
		}
		if strings.TrimSpace(found.Component) != spec.Component {
			found.Component = spec.Component
			needSave = true
		}
		if found.Status != spec.Status {
			found.Status = spec.Status
			needSave = true
		}
		if needSave {
			if err := s.menuRepo.Update(ctx, found); err != nil {
				return false, err
			}
			changed = true
		}
	}

	// 「Webhook 告警事件」已合并到告警监控平台「策略与联调 → 历史」；隐藏旧菜单以免重复入口。
	for i := range all {
		m := &all[i]
		if strings.TrimSpace(m.Path) != "/alert-events" {
			continue
		}
		needSave := false
		if !m.Hidden {
			m.Hidden = true
			needSave = true
		}
		wantEv := "/alert-monitor-platform?tab=config&cfg=history"
		if strings.TrimSpace(m.Redirect) != wantEv {
			m.Redirect = wantEv
			needSave = true
		}
		if needSave {
			if err := s.menuRepo.Update(ctx, m); err != nil {
				return false, err
			}
			changed = true
		}
	}

	// 「告警配置中心」已并入告警监控平台「策略与联调」Tab；隐藏旧入口并跳转。
	for i := range all {
		m := &all[i]
		if strings.TrimSpace(m.Path) != "/alert-config-center" {
			continue
		}
		needSave := false
		if !m.Hidden {
			m.Hidden = true
			needSave = true
		}
		wantCfg := "/alert-monitor-platform?tab=config&cfg=policies"
		if strings.TrimSpace(m.Redirect) != wantCfg {
			m.Redirect = wantCfg
			needSave = true
		}
		if needSave {
			if err := s.menuRepo.Update(ctx, m); err != nil {
				return false, err
			}
			changed = true
		}
	}

	return changed, nil
}

// Create 创建相关的业务逻辑。
func (s *MenuService) Create(ctx context.Context, payload MenuCreatePayload) (*model.Menu, error) {
	parentID := payload.ParentID
	if parentID == nil {
		// For nested paths like /system/security, auto-create missing parent menus.
		autoParentID, err := s.ensureMenuParentChainByPath(ctx, payload.Path)
		if err != nil {
			return nil, err
		}
		parentID = autoParentID
	}
	sortVal, err := s.ensureUniqueSiblingSort(ctx, parentID, payload.Sort, 0)
	if err != nil {
		return nil, err
	}
	menu := model.Menu{
		ParentID:  parentID,
		Path:      payload.Path,
		Name:      payload.Name,
		Icon:      payload.Icon,
		AdminOnly: payload.AdminOnly,
		Sort:      sortVal,
		Hidden:    payload.Hidden,
		Component: payload.Component,
		Redirect:  payload.Redirect,
		Status:    payload.Status,
	}
	if err := s.menuRepo.Create(ctx, &menu); err != nil {
		return nil, err
	}
	s.invalidateCache()
	return &menu, nil
}

func (s *MenuService) ensureMenuParentChainByPath(ctx context.Context, fullPath string) (*uint, error) {
	path := strings.TrimSpace(fullPath)
	if path == "" || path == "/" {
		return nil, nil
	}
	segs := strings.Split(strings.Trim(path, "/"), "/")
	if len(segs) <= 1 {
		return nil, nil
	}
	all, err := s.menuRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	pathMap := make(map[string]*model.Menu, len(all))
	for i := range all {
		p := strings.TrimSpace(all[i].Path)
		if p == "" {
			continue
		}
		pathMap[p] = &all[i]
	}

	var parentID *uint
	currentPath := ""
	for i := 0; i < len(segs)-1; i++ {
		currentPath += "/" + segs[i]
		if found, ok := pathMap[currentPath]; ok {
			id := found.ID
			parentID = &id
			continue
		}
		name := segs[i]
		sortVal, err := s.ensureUniqueSiblingSort(ctx, parentID, 0, 0)
		if err != nil {
			return nil, err
		}
		m := model.Menu{
			ParentID:  parentID,
			Path:      currentPath,
			Name:      name,
			Icon:      "",
			AdminOnly: false,
			Sort:      sortVal,
			Hidden:    false,
			Component: "",
			Redirect:  "",
			Status:    1,
		}
		if err := s.menuRepo.Create(ctx, &m); err != nil {
			return nil, fmt.Errorf("create parent menu %s failed: %w", currentPath, err)
		}
		id := m.ID
		parentID = &id
	}
	return parentID, nil
}

// Update 更新相关的业务逻辑。
func (s *MenuService) Update(ctx context.Context, id uint, payload MenuUpdatePayload) (*model.Menu, error) {
	menu, err := s.menuRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if payload.Name != "" {
		menu.Name = payload.Name
	}
	menu.Path = payload.Path
	menu.Icon = payload.Icon
	if payload.AdminOnly != nil {
		menu.AdminOnly = *payload.AdminOnly
	}
	targetParentID := menu.ParentID
	if payload.ParentID != nil {
		targetParentID = payload.ParentID
	}
	sortVal, err := s.ensureUniqueSiblingSort(ctx, targetParentID, payload.Sort, menu.ID)
	if err != nil {
		return nil, err
	}
	menu.Sort = sortVal
	menu.Hidden = payload.Hidden
	menu.Component = payload.Component
	menu.Redirect = payload.Redirect
	if payload.Status != nil {
		menu.Status = *payload.Status
	}
	if payload.ParentID != nil {
		menu.ParentID = payload.ParentID
	}
	if err := s.menuRepo.Update(ctx, menu); err != nil {
		return nil, err
	}
	s.invalidateCache()
	return menu, nil
}

// Delete 删除相关的业务逻辑。
func (s *MenuService) Delete(ctx context.Context, id uint) error {
	count, err := s.menuRepo.CountChildren(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return apperror.BadRequest("请先删除子菜单后再删除当前菜单")
	}
	if err := s.menuRepo.Delete(ctx, id); err != nil {
		return err
	}
	s.invalidateCache()
	return nil
}

// BatchSetStatus 批量启用/停用菜单。
func (s *MenuService) BatchSetStatus(ctx context.Context, payload MenuBatchStatusPayload) error {
	if len(payload.IDs) == 0 {
		return apperror.BadRequest("请选择需要批量操作的菜单")
	}
	if err := s.menuRepo.BatchUpdateStatus(ctx, payload.IDs, payload.Status); err != nil {
		return err
	}
	s.invalidateCache()
	return nil
}

func (s *MenuService) invalidateCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.treeCache = nil
	s.treeCacheExpireAt = time.Time{}
}
