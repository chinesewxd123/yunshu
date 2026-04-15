package bootstrap

import (
	"go-permission-system/internal/model"

	"gorm.io/gorm"
)

// AutoMigrateModels 与 `go run . migrate` 使用同一套表结构；server 启动时执行可避免漏跑迁移导致 500。
func AutoMigrateModels(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.AutoMigrate(
		&model.User{},
		&model.Role{},
		&model.Permission{},
		&model.UserRole{},
		&model.RegistrationRequest{},
		&model.Menu{},
		&model.LoginLog{},
		&model.OperationLog{},
		&model.K8sCluster{},
		&model.AlertChannel{},
		&model.AlertEvent{},
		&model.Project{},
		&model.ProjectMember{},
		&model.Server{},
		&model.ServerCredential{},
		&model.Service{},
		&model.ServiceLogSource{},
		&model.LogAgent{},
		&model.AgentDiscovery{},
	)
}
