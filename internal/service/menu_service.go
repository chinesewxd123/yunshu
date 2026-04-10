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
	changed, err := s.ensureK8sMenus(ctx, list)
	if err != nil {
		return nil, err
	}
	if !changed {
		return list, nil
	}
	return s.menuRepo.Tree(ctx)
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

func (s *MenuService) Create(ctx context.Context, payload MenuCreatePayload) (*model.Menu, error) {
	menu := model.Menu{
		ParentID:  payload.ParentID,
		Path:      payload.Path,
		Name:      payload.Name,
		Icon:      payload.Icon,
		AdminOnly: payload.AdminOnly,
		Sort:      payload.Sort,
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
	menu.Sort = payload.Sort
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
