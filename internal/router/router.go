package router

import (
	"go-permission-system/internal/bootstrap"
	"go-permission-system/internal/handler"
	"go-permission-system/internal/middleware"
	"go-permission-system/internal/repository"
	"go-permission-system/internal/service"
)

func Register(app *bootstrap.App) {
	registerSwagger(app)

	systemHandler := handler.NewSystemHandler(app.Config.App.Name, app.Config.App.Env)
	userRepo := repository.NewUserRepository(app.DB)
	roleRepo := repository.NewRoleRepository(app.DB)
	permissionRepo := repository.NewPermissionRepository(app.DB)

	authService := service.NewAuthService(userRepo, app.Redis, app.Config.Auth)
	userService := service.NewUserService(userRepo, roleRepo, app.Enforcer)
	roleService := service.NewRoleService(roleRepo, app.Enforcer)
	permissionService := service.NewPermissionService(permissionRepo, app.Enforcer)
	policyService := service.NewPolicyService(roleRepo, permissionRepo, app.Enforcer)

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	roleHandler := handler.NewRoleHandler(roleService)
	permissionHandler := handler.NewPermissionHandler(permissionService)
	policyHandler := handler.NewPolicyHandler(policyService)

	authMiddleware := middleware.Auth(app.Config.Auth.JWTSecret, app.Redis, userRepo, app.Logger)
	authorize := middleware.Authorize(app.Enforcer, app.Logger)

	api := app.Engine.Group("/api/v1")
	api.GET("/health", systemHandler.Health)

	authGroup := api.Group("/auth")
	authGroup.POST("/login", authHandler.Login)
	authGroup.Use(authMiddleware)
	authGroup.POST("/logout", authHandler.Logout)
	authGroup.GET("/me", authHandler.Me)

	users := api.Group("/users")
	users.Use(authMiddleware, authorize)
	users.GET("", userHandler.List)
	users.POST("", userHandler.Create)
	users.GET("/:id", userHandler.Detail)
	users.PUT("/:id", userHandler.Update)
	users.DELETE("/:id", userHandler.Delete)
	users.PUT("/:id/roles", userHandler.AssignRoles)

	roles := api.Group("/roles")
	roles.Use(authMiddleware, authorize)
	roles.GET("", roleHandler.List)
	roles.POST("", roleHandler.Create)
	roles.GET("/:id", roleHandler.Detail)
	roles.PUT("/:id", roleHandler.Update)
	roles.DELETE("/:id", roleHandler.Delete)

	permissions := api.Group("/permissions")
	permissions.Use(authMiddleware, authorize)
	permissions.GET("", permissionHandler.List)
	permissions.POST("", permissionHandler.Create)
	permissions.GET("/:id", permissionHandler.Detail)
	permissions.PUT("/:id", permissionHandler.Update)
	permissions.DELETE("/:id", permissionHandler.Delete)

	policies := api.Group("/policies")
	policies.Use(authMiddleware, authorize)
	policies.GET("", policyHandler.List)
	policies.POST("", policyHandler.Grant)
	policies.DELETE("", policyHandler.Revoke)
}
