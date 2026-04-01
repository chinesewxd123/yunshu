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

	authService := service.NewAuthService(userRepo, app.Redis, app.Config.Auth, app.Mailer, app.Config.App.Name)
	userService := service.NewUserService(userRepo, roleRepo, app.Enforcer)
	roleService := service.NewRoleService(roleRepo, app.Enforcer)
	permissionService := service.NewPermissionService(permissionRepo, app.Enforcer)
	policyService := service.NewPolicyService(roleRepo, permissionRepo, app.Enforcer)

	regReqRepo := repository.NewRegistrationRequestRepository(app.DB)
	menuRepo := repository.NewMenuRepository(app.DB)
	registrationService := service.NewRegistrationService(regReqRepo, userRepo, app.Redis, app.Config.Auth, app.Mailer, app.Config.App.Name)
	menuService := service.NewMenuService(menuRepo)

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	roleHandler := handler.NewRoleHandler(roleService)
	permissionHandler := handler.NewPermissionHandler(permissionService)
	policyHandler := handler.NewPolicyHandler(policyService)
	regHandler := handler.NewRegistrationHandler(registrationService)
	menuHandler := handler.NewMenuHandler(menuService)

	authMiddleware := middleware.Auth(app.Config.Auth.JWTSecret, app.Redis, userRepo, app.Logger)
	authorize := middleware.Authorize(app.Enforcer, app.Logger)

	api := app.Engine.Group("/api/v1")
	// 系统健康检查接口
	api.GET("/health", systemHandler.Health)
	// 认证组
	authGroup := api.Group("/auth")
	// 发送邮箱验证码接口
	authGroup.POST("/verification-code", authHandler.SendEmailCode)
	// 发送登录验证码接口
	authGroup.POST("/login-code", authHandler.SendLoginCodeByUsername)
	// 发送密码登录验证码接口
	authGroup.POST("/password-login-code", authHandler.SendPasswordLoginCode)
	// 登录接口
	authGroup.POST("/login", authHandler.Login)
	// 邮箱登录接口
	authGroup.POST("/email-login", authHandler.EmailLogin)
	// 注册接口（改为申请模式）
	authGroup.POST("/register", regHandler.Apply)

	authGroup.Use(authMiddleware)
	// 退出登录接口
	authGroup.POST("/logout", authHandler.Logout)
	// 获取当前用户信息接口
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

	// 注册审核接口
	registrations := api.Group("/registrations")
	registrations.Use(authMiddleware, authorize)
	registrations.GET("", regHandler.List)
	registrations.POST("/:id/review", regHandler.Review)

	// 菜单管理接口
	menus := api.Group("/menus")
	menus.Use(authMiddleware, authorize)
	menus.GET("/tree", menuHandler.Tree)
	menus.POST("", menuHandler.Create)
	menus.PUT("/:id", menuHandler.Update)
	menus.DELETE("/:id", menuHandler.Delete)
}
