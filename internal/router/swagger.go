package router

import (
	"strings"

	swaggerDocs "go-permission-system/docs/swagger"
	"go-permission-system/internal/bootstrap"

	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func registerSwagger(app *bootstrap.App) {
	cfg := app.Config.Swagger
	if !cfg.Enabled {
		return
	}

	basePath := normalizeSwaggerPath(cfg.Path)
	swaggerDocs.SwaggerInfo.Title = "Permission System API"
	swaggerDocs.SwaggerInfo.Description = "Permission management system built with Gin, MySQL, Redis, Casbin, Cobra and slog."
	swaggerDocs.SwaggerInfo.Version = "1.0"
	swaggerDocs.SwaggerInfo.BasePath = "/"

	app.Engine.GET(basePath+"/*any", ginSwagger.WrapHandler(
		swaggerfiles.Handler,
		ginSwagger.DocExpansion("list"),
		ginSwagger.DefaultModelsExpandDepth(-1),
		ginSwagger.PersistAuthorization(true),
	))
}

func normalizeSwaggerPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/swagger"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if path != "/" {
		path = strings.TrimRight(path, "/")
	}
	return path
}
