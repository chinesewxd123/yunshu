package cmd

import (
	"context"
	"errors"
	"fmt"

	"go-permission-system/internal/bootstrap"
	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/password"
	"go-permission-system/internal/service"

	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

func init() {
	rootCmd.AddCommand(seedCmd)
}

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed default admin user, roles and permissions",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := bootstrap.NewBuilder().
			WithConfig(configPath).
			WithLogger().
			WithMySQL().
			WithCasbin().
			Build()
		if err != nil {
			return err
		}
		defer app.Close()

		ctx := context.Background()

		// 历史错误：撤销策略实际路由为 DELETE /api/v1/policies（JSON body），与 /api/v1/policies/:id 不匹配，会导致无法撤销授权
		if err := service.RemovePermissionPolicies(app.Enforcer, "/api/v1/policies/:id", "DELETE"); err != nil {
			return err
		}
		if err := app.DB.WithContext(ctx).Where("resource = ? AND action = ?", "/api/v1/policies/:id", "DELETE").Delete(&model.Permission{}).Error; err != nil {
			return err
		}

		permissions := defaultPermissions()
		for _, item := range permissions {
			var permission model.Permission
			err := app.DB.WithContext(ctx).
				Where("resource = ? AND action = ?", item.Resource, item.Action).
				First(&permission).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err = app.DB.WithContext(ctx).Create(&item).Error; err != nil {
					return err
				}
				permission = item
			} else if err != nil {
				return err
			} else {
				permission.Name = item.Name
				permission.Description = item.Description
				if err = app.DB.WithContext(ctx).Save(&permission).Error; err != nil {
					return err
				}
			}

			if _, err = app.Enforcer.AddPolicy("super-admin", permission.Resource, permission.Action); err != nil {
				return err
			}
		}

		adminRole := model.Role{
			Name:        "Super Admin",
			Code:        "super-admin",
			Description: "Built-in administrator role with full access.",
			Status:      model.StatusEnabled,
		}
		if err := upsertRole(ctx, app.DB, &adminRole); err != nil {
			return err
		}

		hashedPassword, err := password.Hash("Admin@123")
		if err != nil {
			return err
		}

		adminEmail := "rootwxd@163.com"
		adminUser := model.User{
			Username: "admin",
			Email:    &adminEmail,
			Password: hashedPassword,
			Nickname: "System Admin",
			Status:   model.StatusEnabled,
		}
		if err := upsertUser(ctx, app.DB, &adminUser); err != nil {
			return err
		}

		if err := app.DB.WithContext(ctx).Model(&adminUser).Association("Roles").Replace([]model.Role{adminRole}); err != nil {
			return err
		}
		if err := service.SyncUserRoles(app.Enforcer, adminUser.ID, []model.Role{adminRole}); err != nil {
			return err
		}

		if err := seedMenus(ctx, app.DB); err != nil {
			return err
		}

		app.Logger.Info.Info("seed completed", "username", adminUser.Username, "email", adminUser.Email, "password", "Admin@123")
		fmt.Println("seed completed: admin / Admin@123 / admin@example.com")
		return nil
	},
}

func upsertRole(ctx context.Context, db *gorm.DB, role *model.Role) error {
	var existing model.Role
	err := db.WithContext(ctx).Where("code = ?", role.Code).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.WithContext(ctx).Create(role).Error
	}
	if err != nil {
		return err
	}

	existing.Name = role.Name
	existing.Description = role.Description
	existing.Status = role.Status
	if err := db.WithContext(ctx).Save(&existing).Error; err != nil {
		return err
	}
	*role = existing
	return nil
}

func upsertUser(ctx context.Context, db *gorm.DB, user *model.User) error {
	var existing model.User
	err := db.WithContext(ctx).Where("username = ?", user.Username).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.WithContext(ctx).Create(user).Error
	}
	if err != nil {
		return err
	}

	existing.Email = user.Email
	existing.Password = user.Password
	existing.Nickname = user.Nickname
	existing.Status = user.Status
	if err := db.WithContext(ctx).Save(&existing).Error; err != nil {
		return err
	}
	*user = existing
	return nil
}

func defaultPermissions() []model.Permission {
	return []model.Permission{
		{Name: "用户列表", Resource: "/api/v1/users", Action: "GET", Description: "View user list"},
		{Name: "创建用户", Resource: "/api/v1/users", Action: "POST", Description: "Create user"},
		{Name: "用户详情", Resource: "/api/v1/users/:id", Action: "GET", Description: "View user detail"},
		{Name: "更新用户", Resource: "/api/v1/users/:id", Action: "PUT", Description: "Update user"},
		{Name: "删除用户", Resource: "/api/v1/users/:id", Action: "DELETE", Description: "Delete user"},
		{Name: "分配用户角色", Resource: "/api/v1/users/:id/roles", Action: "PUT", Description: "Assign roles to user"},
		{Name: "导出用户", Resource: "/api/v1/users/export", Action: "GET", Description: "Export users to Excel"},
		{Name: "导入用户", Resource: "/api/v1/users/import", Action: "POST", Description: "Import users from Excel"},
		{Name: "角色列表", Resource: "/api/v1/roles", Action: "GET", Description: "View role list"},
		{Name: "创建角色", Resource: "/api/v1/roles", Action: "POST", Description: "Create role"},
		{Name: "角色详情", Resource: "/api/v1/roles/:id", Action: "GET", Description: "View role detail"},
		{Name: "更新角色", Resource: "/api/v1/roles/:id", Action: "PUT", Description: "Update role"},
		{Name: "删除角色", Resource: "/api/v1/roles/:id", Action: "DELETE", Description: "Delete role"},
		{Name: "API列表", Resource: "/api/v1/permissions", Action: "GET", Description: "View permission list"},
		{Name: "创建API", Resource: "/api/v1/permissions", Action: "POST", Description: "Create permission"},
		{Name: "API详情", Resource: "/api/v1/permissions/:id", Action: "GET", Description: "View permission detail"},
		{Name: "更新API", Resource: "/api/v1/permissions/:id", Action: "PUT", Description: "Update permission"},
		{Name: "删除API", Resource: "/api/v1/permissions/:id", Action: "DELETE", Description: "Delete permission"},
		{Name: "授权列表", Resource: "/api/v1/policies", Action: "GET", Description: "View policy list"},
		{Name: "创建授权策略", Resource: "/api/v1/policies", Action: "POST", Description: "Grant permission to role"},
		{Name: "删除授权策略", Resource: "/api/v1/policies", Action: "DELETE", Description: "Revoke permission from role (JSON body)"},
		{Name: "注册审核列表", Resource: "/api/v1/registrations", Action: "GET", Description: "View registration requests"},
		{Name: "审核注册申请", Resource: "/api/v1/registrations/:id/review", Action: "POST", Description: "Review registration request"},
		{Name: "菜单树", Resource: "/api/v1/menus/tree", Action: "GET", Description: "View menu tree"},
		{Name: "创建菜单", Resource: "/api/v1/menus", Action: "POST", Description: "Create menu"},
		{Name: "更新菜单", Resource: "/api/v1/menus/:id", Action: "PUT", Description: "Update menu"},
		{Name: "删除菜单", Resource: "/api/v1/menus/:id", Action: "DELETE", Description: "Delete menu"},
		{Name: "登录日志列表", Resource: "/api/v1/login-logs", Action: "GET", Description: "View login logs"},
		{Name: "导出登录日志", Resource: "/api/v1/login-logs/export", Action: "GET", Description: "Export login logs to Excel"},
		{Name: "删除登录日志", Resource: "/api/v1/login-logs/:id", Action: "DELETE", Description: "Delete login log"},
		{Name: "批量删除登录日志", Resource: "/api/v1/login-logs/delete", Action: "POST", Description: "Batch delete login logs"},
		{Name: "操作历史列表", Resource: "/api/v1/operation-logs", Action: "GET", Description: "View operation logs"},
		{Name: "导出操作历史", Resource: "/api/v1/operation-logs/export", Action: "GET", Description: "Export operation logs to Excel"},
		{Name: "删除操作历史", Resource: "/api/v1/operation-logs/:id", Action: "DELETE", Description: "Delete operation log"},
		{Name: "批量删除操作历史", Resource: "/api/v1/operation-logs/delete", Action: "POST", Description: "Batch delete operation logs"},
		{Name: "查看封禁 IP 列表", Resource: "/api/v1/security/banned-ips", Action: "GET", Description: "View banned IPs list"},
		{Name: "解除封禁 IP", Resource: "/api/v1/security/banned-ips/unban", Action: "POST", Description: "Unban IP"},
	}
}

func seedMenus(ctx context.Context, db *gorm.DB) error {
	menus := defaultMenus()
	for i := range menus {
		if err := upsertMenu(ctx, db, &menus[i], 0); err != nil {
			return err
		}
		for j := range menus[i].Children {
			if err := upsertMenu(ctx, db, &menus[i].Children[j], menus[i].ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func upsertMenu(ctx context.Context, db *gorm.DB, menu *model.Menu, parentID uint) error {
	var existing model.Menu
	query := db.WithContext(ctx).Where("name = ?", menu.Name)
	if parentID == 0 {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", parentID)
	}
	err := query.First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		menu.ParentID = nil
		if parentID > 0 {
			p := parentID
			menu.ParentID = &p
		}
		return db.WithContext(ctx).Create(menu).Error
	}
	if err != nil {
		return err
	}

	existing.Path = menu.Path
	existing.Icon = menu.Icon
	existing.Sort = menu.Sort
	existing.Hidden = menu.Hidden
	existing.AdminOnly = menu.AdminOnly
	existing.Component = menu.Component
	existing.Redirect = menu.Redirect
	existing.Status = menu.Status
	// 以前如果已有同名菜单但 parent_id 写错，重新 seed 时旧逻辑不会修正 parent_id。
	// 这里显式把 parent_id 修正到目标结构，保证菜单树能正确挂到“系统管理”下面。
	existing.ParentID = nil
	if parentID > 0 {
		p := parentID
		existing.ParentID = &p
	}
	if err := db.WithContext(ctx).Save(&existing).Error; err != nil {
		return err
	}
	*menu = existing
	return nil
}

func defaultMenus() []model.Menu {
	return []model.Menu{
		{
			Name:      "资产总览",
			Path:      "/",
			Icon:      "DatabaseOutlined",
			Sort:      1,
			Component: "",
			Status:    1,
		},
		{
			Name:   "系统管理",
			Path:   "/system",
			Icon:   "SettingOutlined",
			Sort:   2,
			Status: 1,
			Children: []model.Menu{
				{Name: "账号管理", Path: "/users", Icon: "TeamOutlined", Sort: 1, Component: "", Status: 1},
				{Name: "角色管理", Path: "/roles", Icon: "ApartmentOutlined", Sort: 2, Component: "", Status: 1},
				{Name: "API管理", Path: "/permissions", Icon: "ApiOutlined", Sort: 3, Component: "", Status: 1},
				{Name: "授权管理", Path: "/policies", Icon: "AuditOutlined", Sort: 4, Component: "", Status: 1},
				{Name: "注册审核", Path: "/registrations", Icon: "CheckCircleOutlined", Sort: 5, Component: "", Status: 1},
				{Name: "菜单管理", Path: "/menus", Icon: "MenuOutlined", Sort: 6, Component: "", Status: 1},
				{Name: "登录日志", Path: "/login-logs", Icon: "LoginOutlined", Sort: 7, Component: "", Status: 1},
				{Name: "操作历史", Path: "/operation-logs", Icon: "HistoryOutlined", Sort: 8, Component: "", Status: 1},
				{Name: "封禁 IP 管理", Path: "/banned-ips", Icon: "ApiOutlined", Sort: 9, Component: "", Status: 1, AdminOnly: false},
			},
		},
	}
}
