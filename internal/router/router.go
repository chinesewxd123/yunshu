package router

import (
	"yunshu/internal/bootstrap"
	grpcclient "yunshu/internal/grpc/client"
	"yunshu/internal/handler"
)

// Register 装配依赖并注册全部 HTTP 路由。
func Register(app *bootstrap.App, runtimeClient *grpcclient.RuntimeClient) {
	handler.SetLogger(app.Logger)
	registerSwagger(app)

	d := wireRouteDeps(app, runtimeClient)
	api := app.Engine.Group("/api/v1")
	registerPlatformRoutes(api, d)
	registerK8sRoutes(api, d)
	registerProjectRoutes(api, d)
}
