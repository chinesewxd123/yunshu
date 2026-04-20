package cmd

import (
	"context"
	"fmt"
	"strings"

	"go-permission-system/internal/bootstrap"
	"go-permission-system/internal/model"
	"go-permission-system/internal/repository"
	"go-permission-system/internal/service"

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

		// 2) 三元只读：对 K8s 读接口下发 k8s:cluster:*:ns:*:<path> GET
		k8sPaths := collectK8sReadPaths(perms)
		for _, path := range k8sPaths {
			res := "k8s:cluster:*:ns:*:" + path
			_, _ = app.Enforcer.AddPolicy(roleCode, res, "GET")
		}

		fmt.Printf("done: user=%s role=%s readonly_get_policies_added=%d k8s_scoped_paths=%d\n", username, roleCode, added, len(k8sPaths))
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

func collectK8sReadPaths(perms []model.Permission) []string {
	seen := map[string]bool{}
	out := make([]string, 0)
	for _, p := range perms {
		if strings.ToUpper(strings.TrimSpace(p.Action)) != "GET" {
			continue
		}
		path := strings.TrimSpace(p.Resource)
		if path == "" || !strings.HasPrefix(path, "/api/v1/") {
			continue
		}
		if !isK8sReadPath(path) {
			continue
		}
		if seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, path)
	}
	return out
}

func isK8sReadPath(path string) bool {
	p := strings.TrimSpace(path)
	k8sPrefixes := []string{
		"/api/v1/clusters",
		"/api/v1/pods",
		"/api/v1/namespaces",
		"/api/v1/nodes",
		"/api/v1/deployments",
		"/api/v1/statefulsets",
		"/api/v1/daemonsets",
		"/api/v1/cronjobs",
		"/api/v1/jobs",
		"/api/v1/configmaps",
		"/api/v1/secrets",
		"/api/v1/k8s-services",
		"/api/v1/persistentvolumes",
		"/api/v1/persistentvolumeclaims",
		"/api/v1/storageclasses",
		"/api/v1/ingresses",
		"/api/v1/events",
		"/api/v1/crds",
		"/api/v1/crs",
		"/api/v1/rbac",
	}
	for _, prefix := range k8sPrefixes {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}
