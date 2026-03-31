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
			Name:        "超级管理员",
			Code:        "super-admin",
			Description: "系统内置超级管理员角色",
			Status:      model.StatusEnabled,
		}
		if err := upsertRole(ctx, app.DB, &adminRole); err != nil {
			return err
		}

		hashedPassword, err := password.Hash("Admin@123")
		if err != nil {
			return err
		}

		adminUser := model.User{
			Username: "admin",
			Password: hashedPassword,
			Nickname: "系统管理员",
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

		app.Logger.Info("seed completed", "username", adminUser.Username, "password", "Admin@123")
		fmt.Println("seed completed: admin / Admin@123")
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
		{Name: "用户列表", Resource: "/api/v1/users", Action: "GET", Description: "查看用户列表"},
		{Name: "创建用户", Resource: "/api/v1/users", Action: "POST", Description: "创建新用户"},
		{Name: "用户详情", Resource: "/api/v1/users/:id", Action: "GET", Description: "查看单个用户"},
		{Name: "更新用户", Resource: "/api/v1/users/:id", Action: "PUT", Description: "更新用户信息"},
		{Name: "删除用户", Resource: "/api/v1/users/:id", Action: "DELETE", Description: "删除用户"},
		{Name: "分配用户角色", Resource: "/api/v1/users/:id/roles", Action: "PUT", Description: "为用户分配角色"},
		{Name: "角色列表", Resource: "/api/v1/roles", Action: "GET", Description: "查看角色列表"},
		{Name: "创建角色", Resource: "/api/v1/roles", Action: "POST", Description: "创建角色"},
		{Name: "角色详情", Resource: "/api/v1/roles/:id", Action: "GET", Description: "查看单个角色"},
		{Name: "更新角色", Resource: "/api/v1/roles/:id", Action: "PUT", Description: "更新角色"},
		{Name: "删除角色", Resource: "/api/v1/roles/:id", Action: "DELETE", Description: "删除角色"},
		{Name: "权限列表", Resource: "/api/v1/permissions", Action: "GET", Description: "查看权限列表"},
		{Name: "创建权限", Resource: "/api/v1/permissions", Action: "POST", Description: "创建权限"},
		{Name: "权限详情", Resource: "/api/v1/permissions/:id", Action: "GET", Description: "查看单个权限"},
		{Name: "更新权限", Resource: "/api/v1/permissions/:id", Action: "PUT", Description: "更新权限"},
		{Name: "删除权限", Resource: "/api/v1/permissions/:id", Action: "DELETE", Description: "删除权限"},
		{Name: "策略列表", Resource: "/api/v1/policies", Action: "GET", Description: "查看策略列表"},
		{Name: "新增策略", Resource: "/api/v1/policies", Action: "POST", Description: "角色授予权限"},
		{Name: "删除策略", Resource: "/api/v1/policies", Action: "DELETE", Description: "角色移除权限"},
	}
}
