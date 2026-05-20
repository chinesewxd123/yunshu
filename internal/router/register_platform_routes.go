package router

import (
	"yunshu/internal/middleware"

	"github.com/gin-gonic/gin"
)

func registerPlatformRoutes(api *gin.RouterGroup, d *routeDeps) {
	// 系统健康检查接口
	api.GET("/health", d.systemHandler.Health)
	// 认证组
	authGroup := api.Group("/auth")
	// 发送邮箱验证码接口
	authGroup.POST("/verification-code", d.authHandler.SendEmailCode)
	// 发送登录验证码接口
	authGroup.POST("/login-code", d.authHandler.SendLoginCodeByUsername)
	// 发送密码登录验证码接口
	authGroup.POST("/password-login-code", d.authHandler.SendPasswordLoginCode)
	// 登录接口
	authGroup.POST("/login", d.authHandler.Login)
	// 邮箱登录接口
	authGroup.POST("/email-login", d.authHandler.EmailLogin)
	// 注册接口（改为申请模式）
	authGroup.POST("/register", middleware.RegistrationRateLimit(d.app.Redis), d.regHandler.Apply)

	authAuthed := authGroup.Group("")
	authAuthed.Use(d.authMiddleware, d.opAudit)
	authAuthed.POST("/logout", d.authHandler.Logout)
	authAuthed.GET("/me", d.authHandler.Me)
	authAuthed.PUT("/me", d.authHandler.UpdateProfile)
	authAuthed.PUT("/password", d.authHandler.ChangePassword)

	users := api.Group("/users")
	users.Use(d.authMiddleware, d.authorize, d.opAudit)
	users.GET("", d.userHandler.List)
	users.GET("/export", d.userHandler.Export)
	users.GET("/import-template", d.userHandler.ImportTemplate)
	users.POST("/import", d.userHandler.Import)
	users.POST("", d.userHandler.Create)
	users.GET("/:id", d.userHandler.Detail)
	users.PUT("/:id", d.userHandler.Update)
	users.DELETE("/:id", d.userHandler.Delete)
	users.PUT("/:id/roles", d.userHandler.AssignRoles)

	departments := api.Group("/departments")
	departments.Use(d.authMiddleware, d.authorize, d.opAudit)
	departments.GET("/tree", d.departmentHandler.Tree)
	departments.GET("/:id", d.departmentHandler.Detail)
	departments.POST("", d.departmentHandler.Create)
	departments.PUT("/:id", d.departmentHandler.Update)
	departments.DELETE("/:id", d.departmentHandler.Delete)

	roles := api.Group("/roles")
	roles.Use(d.authMiddleware, d.authorize, d.opAudit)
	roles.GET("", d.roleHandler.List)
	roles.POST("", d.roleHandler.Create)
	roles.GET("/:id", d.roleHandler.Detail)
	roles.PUT("/:id", d.roleHandler.Update)
	roles.DELETE("/:id", d.roleHandler.Delete)

	userGroups := api.Group("/user-groups")
	userGroups.Use(d.authMiddleware, d.authorize, d.opAudit)
	userGroups.GET("", d.userGroupHandler.List)
	userGroups.POST("", d.userGroupHandler.Create)
	userGroups.GET("/:id", d.userGroupHandler.Detail)
	userGroups.PUT("/:id", d.userGroupHandler.Update)
	userGroups.DELETE("/:id", d.userGroupHandler.Delete)
	userGroups.PUT("/:id/users", d.userGroupHandler.AssignUsers)

	permissions := api.Group("/permissions")
	permissions.Use(d.authMiddleware, d.authorize, d.opAudit)
	permissions.GET("", d.permissionHandler.List)
	permissions.POST("", d.permissionHandler.Create)
	permissions.GET("/:id", d.permissionHandler.Detail)
	permissions.PUT("/:id", d.permissionHandler.Update)
	permissions.DELETE("/:id", d.permissionHandler.Delete)

	policies := api.Group("/policies")
	policies.Use(d.authMiddleware, d.authorize, d.opAudit)
	policies.GET("", d.policyHandler.List)
	policies.POST("", d.policyHandler.Grant)
	policies.DELETE("", d.policyHandler.Revoke)

	k8sPolicies := api.Group("/k8s-policies")
	k8sPolicies.Use(d.authMiddleware, d.authorize, d.opAudit)
	k8sPolicies.GET("/actions", d.k8sScopedPolicyHandler.Actions)
	k8sPolicies.GET("/paths", d.k8sScopedPolicyHandler.Paths)
	k8sPolicies.GET("", d.k8sScopedPolicyHandler.ListByRole)
	k8sPolicies.GET("/cluster-auth-matrix", d.k8sScopedPolicyHandler.ClusterAuthMatrix)
	k8sPolicies.GET("/user-cluster-auth", d.k8sScopedPolicyHandler.UserClusterAuth)
	k8sPolicies.POST("/grant-preset", d.k8sScopedPolicyHandler.GrantPreset)
	k8sPolicies.DELETE("/cluster-grants/:id", d.k8sScopedPolicyHandler.DeleteClusterGrant)
	k8sPolicies.POST("/cluster-grants/batch-delete", d.k8sScopedPolicyHandler.DeleteClusterGrantsBatch)

	k8sNsDeny := api.Group("/k8s-namespace-deny-rules")
	k8sNsDeny.Use(d.authMiddleware, d.authorize, d.opAudit)
	k8sNsDeny.GET("", d.k8sNamespaceDenyHandler.List)
	k8sNsDeny.POST("", d.k8sNamespaceDenyHandler.Create)
	k8sNsDeny.DELETE("/:id", d.k8sNamespaceDenyHandler.Delete)

	k8sNsAllow := api.Group("/k8s-namespace-allow-rules")
	k8sNsAllow.Use(d.authMiddleware, d.authorize, d.opAudit)
	k8sNsAllow.GET("", d.k8sNamespaceAllowHandler.List)
	k8sNsAllow.POST("", d.k8sNamespaceAllowHandler.Create)
	k8sNsAllow.DELETE("/:id", d.k8sNamespaceAllowHandler.Delete)

	// 注册审核接口
	registrations := api.Group("/registrations")
	registrations.Use(d.authMiddleware, d.authorize, d.opAudit)
	registrations.GET("", d.regHandler.List)
	registrations.POST("/:id/review", d.regHandler.Review)

	// Admin management (list/unban banned IPs) - super-admin only
	admin := api.Group("/security")
	admin.Use(d.authMiddleware)
	admin.GET("/banned-ips", d.adminHandler.ListBannedIPs)
	admin.POST("/banned-ips/unban", d.adminHandler.UnbanIP)

	// 菜单管理接口
	menus := api.Group("/menus")
	menus.Use(d.authMiddleware, d.authorize, d.opAudit)
	menus.GET("/tree", d.menuHandler.Tree)
	menus.POST("", d.menuHandler.Create)
	menus.PUT("/status", d.menuHandler.BatchStatus)
	menus.PUT("/:id", d.menuHandler.Update)
	menus.DELETE("/:id", d.menuHandler.Delete)

	dictEntries := api.Group("/dict/entries")
	dictEntries.Use(d.authMiddleware, d.authorize, d.opAudit)
	dictEntries.GET("", d.dictEntryHandler.List)
	dictEntries.POST("", d.dictEntryHandler.Create)
	dictEntries.POST("/:id/reveal-value", d.dictEntryHandler.RevealValue)
	dictEntries.PUT("/:id", d.dictEntryHandler.Update)
	dictEntries.DELETE("/:id", d.dictEntryHandler.Delete)

	dictOptions := api.Group("/dict/options")
	dictOptions.Use(d.authMiddleware, d.authorize, d.opAudit)
	dictOptions.GET("/:dictType", d.dictEntryHandler.Options)

	alertWebhook := api.Group("/alerts")
	alertWebhook.POST("/webhook/alertmanager", d.alertHandler.ReceiveAlertmanager)

	alerts := api.Group("/alerts")
	alerts.Use(d.authMiddleware, d.authorize, d.opAudit)
	alerts.GET("/channels", d.alertHandler.ListChannels)
	alerts.POST("/channels", d.alertHandler.CreateChannel)
	alerts.PUT("/channels/:id", d.alertHandler.UpdateChannel)
	alerts.DELETE("/channels/:id", d.alertHandler.DeleteChannel)
	alerts.POST("/channels/:id/test", d.alertHandler.TestChannel)
	alerts.POST("/channels/preview-template", d.alertHandler.PreviewChannelTemplate)
	alerts.GET("/events", d.alertHandler.ListEvents)
	alerts.GET("/history/stats", d.alertHandler.HistoryStats)

	alerts.GET("/datasources", d.alertPlatformHandler.ListDatasources)
	alerts.POST("/datasources", d.alertPlatformHandler.CreateDatasource)
	alerts.GET("/datasources/:id/ping", d.alertPlatformHandler.PingDatasource)
	alerts.GET("/datasources/:id/prometheus-alerts", d.alertPlatformHandler.PromActiveAlerts)
	alerts.POST("/datasources/:id/query", d.alertPlatformHandler.PromQuery)
	alerts.POST("/datasources/:id/query_range", d.alertPlatformHandler.PromQueryRange)
	alerts.PUT("/datasources/:id", d.alertPlatformHandler.UpdateDatasource)
	alerts.DELETE("/datasources/:id", d.alertPlatformHandler.DeleteDatasource)

	alerts.GET("/silences", d.alertPlatformHandler.ListSilences)
	alerts.POST("/silences", d.alertPlatformHandler.CreateSilence)
	alerts.POST("/silences/batch", d.alertPlatformHandler.CreateSilenceBatch)
	alerts.PUT("/silences/:id", d.alertPlatformHandler.UpdateSilence)
	alerts.DELETE("/silences/:id", d.alertPlatformHandler.DeleteSilence)

	alerts.GET("/monitor-rules", d.alertPlatformHandler.ListMonitorRules)
	alerts.POST("/monitor-rules", d.alertPlatformHandler.CreateMonitorRule)
	alerts.PUT("/monitor-rules/:id", d.alertPlatformHandler.UpdateMonitorRule)
	alerts.DELETE("/monitor-rules/:id", d.alertPlatformHandler.DeleteMonitorRule)
	alerts.GET("/monitor-rules/:id/assignees", d.alertPlatformHandler.GetMonitorRuleAssignees)
	alerts.PUT("/monitor-rules/:id/assignees", d.alertPlatformHandler.UpsertMonitorRuleAssignees)
	alerts.GET("/duty-blocks", d.alertPlatformHandler.ListDutyBlocks)
	alerts.POST("/duty-blocks", d.alertPlatformHandler.CreateDutyBlock)
	alerts.PUT("/duty-blocks/:id", d.alertPlatformHandler.UpdateDutyBlock)
	alerts.DELETE("/duty-blocks/:id", d.alertPlatformHandler.DeleteDutyBlock)

	// 订阅树（路由树）：用于更接近夜莺的“订阅/路由”配置方式
	alerts.GET("/subscriptions", d.alertSubscriptionHandler.ListNodes)
	alerts.GET("/subscriptions/tree", d.alertSubscriptionHandler.GetNodeTree)
	alerts.POST("/subscriptions", d.alertSubscriptionHandler.CreateNode)
	alerts.PUT("/subscriptions/:id", d.alertSubscriptionHandler.UpdateNode)
	alerts.DELETE("/subscriptions/:id", d.alertSubscriptionHandler.DeleteNode)
	alerts.POST("/subscriptions/:id/move", d.alertSubscriptionHandler.MoveNode)
	alerts.POST("/subscriptions/migrate-from-policies", d.alertSubscriptionHandler.MigrateFromPolicies)
	alerts.POST("/subscriptions/clone-from-project", d.alertSubscriptionHandler.CloneProjectRouting)

	if d.alertInhibitionHandler != nil {
		alerts.GET("/inhibition-rules", d.alertInhibitionHandler.List)
		alerts.POST("/inhibition-rules", d.alertInhibitionHandler.Create)
		alerts.PUT("/inhibition-rules/:id", d.alertInhibitionHandler.Update)
		alerts.DELETE("/inhibition-rules/:id", d.alertInhibitionHandler.Delete)
		alerts.POST("/inhibition-rules/refresh-cache", d.alertInhibitionHandler.RefreshCache)
	}

	// 接收组（订阅节点引用）
	alerts.GET("/receiver-groups", d.alertReceiverGroupHandler.List)
	alerts.POST("/receiver-groups", d.alertReceiverGroupHandler.Create)
	alerts.PUT("/receiver-groups/:id", d.alertReceiverGroupHandler.Update)
	alerts.DELETE("/receiver-groups/:id", d.alertReceiverGroupHandler.Delete)

	// 云服务器到期规则（Cloud Expiry Rules）
	alerts.GET("/cloud-expiry-rules", d.cloudExpiryRuleHandler.List)
	alerts.POST("/cloud-expiry-rules", d.cloudExpiryRuleHandler.Create)
	alerts.PUT("/cloud-expiry-rules/:id", d.cloudExpiryRuleHandler.Update)
	alerts.DELETE("/cloud-expiry-rules/:id", d.cloudExpiryRuleHandler.Delete)
	alerts.POST("/cloud-expiry-rules/evaluate-now", d.cloudExpiryRuleHandler.EvaluateNow)

	loginLogs := api.Group("/login-logs")
	loginLogs.Use(d.authMiddleware, d.authorize, d.opAudit)
	loginLogs.GET("/export", d.loginLogHandler.Export)
	loginLogs.GET("", d.loginLogHandler.List)
	loginLogs.POST("/delete", d.loginLogHandler.BatchDelete)
	loginLogs.DELETE("/:id", d.loginLogHandler.Delete)

	operationLogs := api.Group("/operation-logs")
	operationLogs.Use(d.authMiddleware, d.authorize, d.opAudit)
	operationLogs.GET("/export", d.opLogHandler.Export)
	operationLogs.GET("", d.opLogHandler.List)
	operationLogs.POST("/delete", d.opLogHandler.BatchDelete)
	operationLogs.DELETE("/:id", d.opLogHandler.Delete)

}
