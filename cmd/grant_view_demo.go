package cmd

import (
	"context"
	"fmt"
	"strings"

	"yunshu/internal/bootstrap"
	"yunshu/internal/model"
	"yunshu/internal/repository"
	"yunshu/internal/service"

	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

func init() {
	rootCmd.AddCommand(grantViewDemoCmd)
	grantViewDemoCmd.Flags().String("username", "dev", "target username")
	grantViewDemoCmd.Flags().String("role", "view-user", "role code to bind")
}

var grantViewDemoCmd = &cobra.Command{
	Use:   "grant-view-demo",
	Short: "Bind user to view role and grant readonly + k8s scoped GET policies",
	RunE: func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString("username")
		roleCode, _ := cmd.Flags().GetString("role")
		username = strings.TrimSpace(username)
		roleCode = strings.TrimSpace(roleCode)
		if username == "" || roleCode == "" {
			return fmt.Errorf("username and role are required")
		}

		app, err := bootstrap.NewBuilder().
			WithConfig(configPath).
			WithLogger().
			WithMySQL().
			WithDictOverrides().
			WithCasbin().
			Build()
		if err != nil {
			return err
		}
		defer app.Close()

		if err := bootstrap.AutoMigrateModels(app.DB); err != nil {
			return err
		}

		ctx := context.Background()
		userRepo := repository.NewUserRepository(app.DB)
		roleRepo := repository.NewRoleRepository(app.DB)
		permRepo := repository.NewPermissionRepository(app.DB)

		user, err := userRepo.GetByUsername(ctx, username)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("user %q not found", username)
			}
			return err
		}

		role, err := findRoleByCode(ctx, roleRepo, roleCode)
		if err != nil {
			return err
		}

		if err := userRepo.ReplaceRoles(ctx, user, []model.Role{*role}); err != nil {
			return err
		}
		if err := service.SyncUserRoles(app.Enforcer, user.ID, []model.Role{*role}); err != nil {
			return err
		}

		// Reset p policies for roleCode
		if _, err := app.Enforcer.RemoveFilteredPolicy(0, roleCode); err != nil {
			return err
		}

		perms, err := permRepo.ListAll(ctx)
		if err != nil {
			return err
		}

		// 1) 全站只读：允许所有 GET 权限
		added := 0
		for _, p := range perms {
			if strings.ToUpper(strings.TrimSpace(p.Action)) != "GET" {
				continue
			}
			obj := strings.TrimSpace(p.Resource)
			if obj == "" {
				continue
			}
			ok, err := app.Enforcer.AddPolicy(roleCode, obj, "GET")
			if err != nil {
				return err
			}
			if ok {
				added++
			}
		}

		accessRepo := repository.NewK8sClusterAccessRepository(app.DB)
		if err := accessRepo.Upsert(ctx, &model.K8sClusterAccessGrant{
			PrincipalKind: model.K8sPrincipalRole,
			PrincipalRef:  roleCode,
			ClusterID:     0,
			Preset:        "readonly",
		}); err != nil {
			return err
		}

		fmt.Printf("done: user=%s role=%s readonly_get_policies_added=%d k8s_cluster_access=readonly(all_clusters)\n", username, roleCode, added)
		return nil
	},
}

func findRoleByCode(ctx context.Context, repo *repository.RoleRepository, code string) (*model.Role, error) {
	all, err := repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	for i := range all {
		if strings.TrimSpace(all[i].Code) == code {
			r := all[i]
			return &r, nil
		}
	}
	return nil, fmt.Errorf("role %q not found", code)
}
