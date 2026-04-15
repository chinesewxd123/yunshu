package service

import (
	"context"
	"errors"
	"strings"

	"go-permission-system/internal/model"
	"go-permission-system/internal/repository"
)

type MenuService struct {
	menuRepo *repository.MenuRepository
}

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

func (s *MenuService) Tree(ctx context.Context) ([]model.Menu, error) {
	list, err := s.menuRepo.Tree(ctx)
	if err != nil {
		return nil, err
	}
	changed, err := s.ensureBuiltInMenus(ctx, list)
	if err != nil {
		return nil, err
	}
	repaired, err := s.repairEventMenus(ctx)
	if err != nil {
		return nil, err
	}
	if !changed && !repaired {
		return list, nil
	}
	return s.menuRepo.Tree(ctx)
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

// ensureAlertNotifyMenus 保证「告警通知」独立目录及 Webhook 子菜单存在，并从「系统管理」下移出（按 path 归位）。
func (s *MenuService) ensureAlertNotifyMenus(ctx context.Context) (bool, error) {
	all, err := s.menuRepo.ListAll(ctx)
	if err != nil {
		return false, err
	}

	const alertRootPath = "/alert-notify"
	requiredChildren := []model.Menu{
		{Path: "/alert-channels", Name: "Webhook 告警通道", Icon: "NotificationOutlined", Sort: 1, Component: "alert-channels-page", Status: 1},
		{Path: "/alert-events", Name: "Webhook 告警事件", Icon: "AlertOutlined", Sort: 2, Component: "alert-events-page", Status: 1},
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

	return changed, nil
}

func (s *MenuService) Create(ctx context.Context, payload MenuCreatePayload) (*model.Menu, error) {
	sortVal, err := s.ensureUniqueSiblingSort(ctx, payload.ParentID, payload.Sort, 0)
	if err != nil {
		return nil, err
	}
	menu := model.Menu{
		ParentID:  payload.ParentID,
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
	return &menu, nil
}

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
	return menu, nil
}

func (s *MenuService) Delete(ctx context.Context, id uint) error {
	count, err := s.menuRepo.CountChildren(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("please delete child menus first")
	}
	return s.menuRepo.Delete(ctx, id)
}
