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
		{Name: "List Users", Resource: "/api/v1/users", Action: "GET", Description: "View user list"},
		{Name: "Create User", Resource: "/api/v1/users", Action: "POST", Description: "Create user"},
		{Name: "User Detail", Resource: "/api/v1/users/:id", Action: "GET", Description: "View user detail"},
		{Name: "Update User", Resource: "/api/v1/users/:id", Action: "PUT", Description: "Update user"},
		{Name: "Delete User", Resource: "/api/v1/users/:id", Action: "DELETE", Description: "Delete user"},
		{Name: "Assign User Roles", Resource: "/api/v1/users/:id/roles", Action: "PUT", Description: "Assign roles to user"},
		{Name: "List Roles", Resource: "/api/v1/roles", Action: "GET", Description: "View role list"},
		{Name: "Create Role", Resource: "/api/v1/roles", Action: "POST", Description: "Create role"},
		{Name: "Role Detail", Resource: "/api/v1/roles/:id", Action: "GET", Description: "View role detail"},
		{Name: "Update Role", Resource: "/api/v1/roles/:id", Action: "PUT", Description: "Update role"},
		{Name: "Delete Role", Resource: "/api/v1/roles/:id", Action: "DELETE", Description: "Delete role"},
		{Name: "List Permissions", Resource: "/api/v1/permissions", Action: "GET", Description: "View permission list"},
		{Name: "Create Permission", Resource: "/api/v1/permissions", Action: "POST", Description: "Create permission"},
		{Name: "Permission Detail", Resource: "/api/v1/permissions/:id", Action: "GET", Description: "View permission detail"},
		{Name: "Update Permission", Resource: "/api/v1/permissions/:id", Action: "PUT", Description: "Update permission"},
		{Name: "Delete Permission", Resource: "/api/v1/permissions/:id", Action: "DELETE", Description: "Delete permission"},
		{Name: "List Policies", Resource: "/api/v1/policies", Action: "GET", Description: "View policy list"},
		{Name: "Create Policy", Resource: "/api/v1/policies", Action: "POST", Description: "Grant permission to role"},
		{Name: "Delete Policy", Resource: "/api/v1/policies", Action: "DELETE", Description: "Revoke permission from role"},
	}
}
