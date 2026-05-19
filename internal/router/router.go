package router

import (
	"context"
	"log/slog"

	"yunshu/internal/bootstrap"
	"yunshu/internal/service/svclog"
	grpcclient "yunshu/internal/grpc/client"
	"yunshu/internal/handler"
	"yunshu/internal/repository"
	"yunshu/internal/service"
	"yunshu/internal/service/k8seventforward"
)

// Register 装配依赖并注册全部 HTTP 路由；返回 K8s Event 转发管理器（可能为 nil）。
// bgCtx 用于 MySQL 定时备份等后台 Worker，进程退出时应 cancel。
func Register(app *bootstrap.App, runtimeClient *grpcclient.RuntimeClient, bgCtx context.Context) *k8seventforward.Manager {
	handler.SetLogger(app.Logger)
	registerSwagger(app)

	d := wireRouteDeps(app, runtimeClient)
	api := app.Engine.Group("/api/v1")
	registerPlatformRoutes(api, d)
	registerK8sRoutes(api, d)
	registerProjectRoutes(api, d)

	if d.mysqlBackupSvc != nil && bgCtx != nil {
		mysqlLog := svclog.Worker("mysql.backup")
		d.mysqlBackupSvc.SetBizLog(mysqlLog)
		go d.mysqlBackupSvc.RunMysqlBackupScheduler(bgCtx, mysqlLog)
	}

	clusterRepo := repository.NewK8sClusterRepository(app.DB)
	runtimeSvc := service.NewK8sRuntimeService(clusterRepo)
	k8sFwdLog := svclog.Worker("k8s.event_forward")
	mgr, err := k8seventforward.NewManager(
		app.DB,
		runtimeSvc,
		app.YamlK8sEventForwardBase,
		app.Config.Alert,
		app.Config.App.Port,
		k8sFwdLog,
	)
	if err != nil {
		k8sFwdLog.Error("manager init failed", slog.Any("error", err))
		return nil
	}
	mgr.Start()
	return mgr
}
