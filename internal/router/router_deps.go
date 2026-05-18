package router

import (
	"log/slog"
	"strings"

	"yunshu/internal/bootstrap"
	grpcclient "yunshu/internal/grpc/client"
	"yunshu/internal/handler"
	"yunshu/internal/middleware"
	"yunshu/internal/repository"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

// routeDeps 聚合路由注册所需的 handler、中间件与共享仓储。
type routeDeps struct {
	app *bootstrap.App

	authMiddleware    gin.HandlerFunc
	wsAuthMiddleware  gin.HandlerFunc
	authorize         gin.HandlerFunc
	k8sScopeAuthorize gin.HandlerFunc
	opAudit           gin.HandlerFunc

	projectMemberRepo *repository.ProjectMemberRepository

	systemHandler *handler.SystemHandler

	authHandler              *handler.AuthHandler
	loginLogHandler          *handler.LoginLogHandler
	opLogHandler             *handler.OperationLogHandler
	userHandler              *handler.UserHandler
	departmentHandler        *handler.DepartmentHandler
	roleHandler              *handler.RoleHandler
	permissionHandler        *handler.PermissionHandler
	policyHandler            *handler.PolicyHandler
	k8sScopedPolicyHandler   *handler.K8sScopedPolicyHandler
	k8sNamespaceDenyHandler  *handler.K8sNamespaceDenyHandler
	k8sNamespaceAllowHandler *handler.K8sNamespaceAllowHandler
	userGroupHandler         *handler.UserGroupHandler
	regHandler               *handler.RegistrationHandler
	menuHandler              *handler.MenuHandler
	dictEntryHandler         *handler.DictEntryHandler
	adminHandler             *handler.AdminHandler

	alertHandler             *handler.AlertHandler
	alertPlatformHandler     *handler.AlertPlatformHandler
	alertSubscriptionHandler *handler.AlertSubscriptionHandler
	alertInhibitionHandler   *handler.AlertInhibitionHandler
	alertReceiverGroupHandler *handler.AlertReceiverGroupHandler
	cloudExpiryRuleHandler   *handler.CloudExpiryRuleHandler

	clusterHandler           *handler.ClusterHandler
	podHandler               *handler.PodHandler
	namespaceHandler         *handler.NamespaceHandler
	nodeHandler              *handler.NodeHandler
	workloadHandler          *handler.WorkloadHandler
	configHandler            *handler.ConfigHandler
	storageHandler           *handler.StorageHandler
	serviceResourceHandler   *handler.ServiceResourceHandler
	ingressHandler           *handler.IngressHandler
	networkPolicyHandler     *handler.NetworkPolicyHandler
	k8sDiscoveryHandler      *handler.K8sDiscoveryHandler
	k8sHPAHandler            *handler.K8sHPAHandler
	k8sResourceWatchHandler  *handler.K8sResourceWatchHandler
	k8sEventForwardHandler   *handler.K8sEventForwardHandler
	eventHandler             *handler.EventHandler
	crdHandler               *handler.CRDHandler
	crHandler                *handler.CRHandler
	rbacHandler              *handler.RBACHandler
	serviceAccountHandler    *handler.ServiceAccountHandler
	overviewHandler          *handler.OverviewHandler

	projectHandler         *handler.ProjectHandler
	logAgentHandler        *handler.LogAgentHandler
	agentDiscoveryHandler  *handler.AgentDiscoveryHandler
}

func wireRouteDeps(app *bootstrap.App, runtimeClient *grpcclient.RuntimeClient) *routeDeps {
	systemHandler := handler.NewSystemHandler(app.Config.App.Name, app.Config.App.Env)
	userRepo := repository.NewUserRepository(app.DB)
	departmentRepo := repository.NewDepartmentRepository(app.DB)
	roleRepo := repository.NewRoleRepository(app.DB)
	permissionRepo := repository.NewPermissionRepository(app.DB)

	loginLogRepo := repository.NewLoginLogRepository(app.DB)
	opLogRepo := repository.NewOperationLogRepository(app.DB)
	loginLogSvc := service.NewLoginLogService(loginLogRepo)
	opLogSvc := service.NewOperationLogService(opLogRepo)

	authService := service.NewAuthService(userRepo, app.Redis, app.Config.Auth, app.Mailer, app.Config.App.Name)
	projectMemberRepo := repository.NewProjectMemberRepository(app.DB)
	alertAssigneeSvc := service.NewAlertRuleAssigneeService(app.DB, userRepo, projectMemberRepo, departmentRepo)
	userService := service.NewUserService(userRepo, roleRepo, departmentRepo, app.Enforcer, projectMemberRepo, alertAssigneeSvc)
	departmentService := service.NewDepartmentService(departmentRepo, userRepo, alertAssigneeSvc)
	roleService := service.NewRoleService(roleRepo, app.Enforcer)
	permissionService := service.NewPermissionService(permissionRepo, app.Enforcer)
	policyService := service.NewPolicyService(roleRepo, permissionRepo, app.Enforcer)
	k8sNsDenyRepo := repository.NewK8sNamespaceDenyRepository(app.DB)
	k8sNsAllowRepo := repository.NewK8sNamespaceAllowRepository(app.DB)
	userGroupRepo := repository.NewUserGroupRepository(app.DB)
	k8sClusterAccessRepo := repository.NewK8sClusterAccessRepository(app.DB)
	clusterRepo := repository.NewK8sClusterRepository(app.DB)
	projectRepo := repository.NewProjectRepository(app.DB)
	k8sScopedPolicyService := service.NewK8sScopedPolicyService(roleRepo, permissionRepo, k8sClusterAccessRepo, k8sNsDenyRepo, k8sNsAllowRepo, userGroupRepo, userRepo, clusterRepo)
	k8sNamespaceDenySvc := service.NewK8sNamespaceDenyService(k8sNsDenyRepo)
	k8sNamespaceDenyHandler := handler.NewK8sNamespaceDenyHandler(k8sNamespaceDenySvc)
	k8sNamespaceAllowSvc := service.NewK8sNamespaceAllowService(k8sNsAllowRepo)
	k8sNamespaceAllowHandler := handler.NewK8sNamespaceAllowHandler(k8sNamespaceAllowSvc)
	userGroupSvc := service.NewUserGroupService(userGroupRepo, userRepo, projectMemberRepo, projectRepo)
	userGroupHandler := handler.NewUserGroupHandler(userGroupSvc)

	regReqRepo := repository.NewRegistrationRequestRepository(app.DB)
	menuRepo := repository.NewMenuRepository(app.DB)
	dictEntryRepo := repository.NewDictEntryRepository(app.DB)
	registrationService := service.NewRegistrationService(regReqRepo, userRepo, app.Redis, app.Config.Auth, app.Mailer, app.Config.App.Name)
	menuService := service.NewMenuService(menuRepo)
	dictEntryService := service.NewDictEntryService(dictEntryRepo)
	alertSilenceSvc := service.NewAlertSilenceService(app.DB)
	alertDutySvc := service.NewAlertDutyService(app.DB, userRepo)
	alertReceiverGroupCache := service.NewReceiverGroupCache(app.DB)
	alertService := service.NewAlertService(app.DB, app.Redis, app.Mailer, app.Config.Alert, &service.AlertServiceOptions{
		SilenceSvc:         alertSilenceSvc,
		AssigneeSvc:        alertAssigneeSvc,
		DutySvc:            alertDutySvc,
		ReceiverGroupCache: alertReceiverGroupCache,
		EncryptionKey:      app.Config.Security.EncryptionKey,
		InfoLog:            app.Logger.Info,
	})
	if strings.TrimSpace(app.Config.Alert.WebhookToken) == "" {
		slog.Warn("alert.webhook_token is empty; Alertmanager webhooks will be rejected until configured")
	}
	cloudExpiryRuleSvc := service.NewCloudExpiryRuleService(app.DB)
	alertDatasourceSvc := service.NewAlertDatasourceService(app.DB)
	alertMonitorRuleSvc := service.NewAlertMonitorRuleService(app.DB, app.Redis)

	k8sRuntimeService := service.NewK8sRuntimeService(clusterRepo)
	clusterService := service.NewK8sClusterService(clusterRepo, dictEntryRepo, k8sRuntimeService, k8sNsDenyRepo, k8sNsAllowRepo, projectMemberRepo)
	podService := service.NewK8sPodService(k8sRuntimeService)
	namespaceService := service.NewK8sNamespaceService(k8sRuntimeService, k8sNsDenyRepo, k8sNsAllowRepo)
	nodeService := service.NewK8sNodeService(k8sRuntimeService)
	workloadService := service.NewK8sWorkloadService(k8sRuntimeService)
	configService := service.NewK8sConfigService(k8sRuntimeService)
	storageService := service.NewK8sStorageService(k8sRuntimeService)
	serviceResourceService := service.NewK8sServiceResourceService(k8sRuntimeService)
	ingressService := service.NewK8sIngressService(k8sRuntimeService, k8sClusterAccessRepo)
	networkPolicyService := service.NewK8sNetworkPolicyService(k8sRuntimeService)
	k8sDiscoveryService := service.NewK8sDiscoveryService(k8sRuntimeService)
	k8sHPAService := service.NewK8sHPAService(k8sRuntimeService)
	eventService := service.NewK8sEventService(k8sRuntimeService)
	crdService := service.NewK8sCRDService(k8sRuntimeService)
	crService := service.NewK8sCRService(k8sRuntimeService)
	rbacService := service.NewK8sRBACService(k8sRuntimeService)
	serviceAccountService := service.NewK8sServiceAccountService(k8sRuntimeService)
	overviewService := service.NewOverviewService(app.DB, k8sRuntimeService, app.Redis, projectMemberRepo, k8sClusterAccessRepo)

	serverRepo := repository.NewServerRepository(app.DB)
	serverGroupRepo := repository.NewServerGroupRepository(app.DB)
	cloudAccountRepo := repository.NewCloudAccountRepository(app.DB)
	serviceRepo := repository.NewServiceRepository(app.DB)
	logRepo := repository.NewLogSourceRepository(app.DB)
	logAgentRepo := repository.NewLogAgentRepository(app.DB)
	agentDiscoveryRepo := repository.NewAgentDiscoveryRepository(app.DB)
	projectMgmtService, err := service.NewProjectMgmtService(projectRepo, serverRepo, serverGroupRepo, cloudAccountRepo, serviceRepo, logRepo, projectMemberRepo, userRepo, departmentRepo, app.Config.Security.EncryptionKey)
	if err != nil {
		panic(err)
	}
	logAgentService := service.NewLogAgentService(logAgentRepo, serverRepo, logRepo, app.Config.Agent.RegisterSecret, app.Config.Agent.DiscoveryRoots)
	agentDiscoveryService := service.NewAgentDiscoveryService(agentDiscoveryRepo, logAgentRepo, serverRepo, logRepo)

	authHandler := handler.NewAuthHandler(authService, loginLogSvc)
	loginLogHandler := handler.NewLoginLogHandler(loginLogSvc)
	opLogHandler := handler.NewOperationLogHandler(opLogSvc)
	userHandler := handler.NewUserHandler(userService)
	departmentHandler := handler.NewDepartmentHandler(departmentService)
	roleHandler := handler.NewRoleHandler(roleService)
	permissionHandler := handler.NewPermissionHandler(permissionService)
	policyHandler := handler.NewPolicyHandler(policyService)
	k8sScopedPolicyHandler := handler.NewK8sScopedPolicyHandler(k8sScopedPolicyService)
	regHandler := handler.NewRegistrationHandler(registrationService)
	menuHandler := handler.NewMenuHandler(menuService)
	dictEntryHandler := handler.NewDictEntryHandler(dictEntryService)
	alertHandler := handler.NewAlertHandler(alertService)
	cloudExpiryRuleHandler := handler.NewCloudExpiryRuleHandler(cloudExpiryRuleSvc, alertService)
	alertPlatformHandler := handler.NewAlertPlatformHandler(alertDatasourceSvc, alertSilenceSvc, alertMonitorRuleSvc, alertAssigneeSvc, alertDutySvc)
	alertSubscriptionSvc := alertService.GetSubscriptionService()
	alertSubscriptionHandler := handler.NewAlertSubscriptionHandler(alertSubscriptionSvc)
	var alertInhibitionHandler *handler.AlertInhibitionHandler
	if inh := alertService.GetInhibitionService(); inh != nil {
		alertInhibitionHandler = handler.NewAlertInhibitionHandler(inh)
	}
	alertReceiverGroupSvc := service.NewAlertReceiverGroupService(app.DB, alertReceiverGroupCache)
	alertReceiverGroupHandler := handler.NewAlertReceiverGroupHandler(alertReceiverGroupSvc)
	adminHandler := handler.NewAdminHandler(app.Redis)
	clusterHandler := handler.NewClusterHandler(clusterService)
	podHandler := handler.NewPodHandler(podService)
	namespaceHandler := handler.NewNamespaceHandler(namespaceService)
	nodeHandler := handler.NewNodeHandler(nodeService)
	workloadHandler := handler.NewWorkloadHandler(workloadService)
	configHandler := handler.NewConfigHandler(configService)
	storageHandler := handler.NewStorageHandler(storageService)
	serviceResourceHandler := handler.NewServiceResourceHandler(serviceResourceService)
	ingressHandler := handler.NewIngressHandler(ingressService)
	networkPolicyHandler := handler.NewNetworkPolicyHandler(networkPolicyService)
	k8sDiscoveryHandler := handler.NewK8sDiscoveryHandler(k8sDiscoveryService)
	k8sHPAHandler := handler.NewK8sHPAHandler(k8sHPAService)
	k8sResourceWatchHandler := handler.NewK8sResourceWatchHandler(k8sRuntimeService)
	k8sEventForwardAdminSvc := service.NewK8sEventForwardAdminService(app.DB)
	k8sEventForwardHandler := handler.NewK8sEventForwardHandler(k8sEventForwardAdminSvc)
	eventHandler := handler.NewEventHandler(eventService)
	crdHandler := handler.NewCRDHandler(crdService)
	crHandler := handler.NewCRHandler(crService)
	rbacHandler := handler.NewRBACHandler(rbacService)
	serviceAccountHandler := handler.NewServiceAccountHandler(serviceAccountService)
	overviewHandler := handler.NewOverviewHandler(overviewService)
	projectHandler := handler.NewProjectHandler(projectMgmtService, runtimeClient.ProjectSrv, runtimeClient.LogSourceSrv)
	logAgentHandler := handler.NewLogAgentHandler(logAgentService, runtimeClient.AgentSrv)
	agentDiscoveryHandler := handler.NewAgentDiscoveryHandler(agentDiscoveryService, runtimeClient.AgentSrv)

	authMiddleware := middleware.Auth(app.Config.Auth.JWTSecret, app.Redis, userRepo, app.Logger)
	wsAuthMiddleware := middleware.WSAuth(app.Config.Auth.JWTSecret, app.Redis, userRepo, app.Logger)
	authorize := middleware.Authorize(app.Enforcer, app.Logger, k8sClusterAccessRepo)
	k8sScopeAuthorize := middleware.K8sScopeAuthorize(app.Logger, permissionRepo, k8sClusterAccessRepo, k8sNsDenyRepo, k8sNsAllowRepo)
	opAudit := middleware.OperationAudit(opLogSvc, app.Logger)


	return &routeDeps{
		app: app,

		authMiddleware:    authMiddleware,
		wsAuthMiddleware:  wsAuthMiddleware,
		authorize:         authorize,
		k8sScopeAuthorize: k8sScopeAuthorize,
		opAudit:           opAudit,

		projectMemberRepo: projectMemberRepo,

		systemHandler: systemHandler,

		authHandler:              authHandler,
		loginLogHandler:          loginLogHandler,
		opLogHandler:             opLogHandler,
		userHandler:              userHandler,
		departmentHandler:        departmentHandler,
		roleHandler:              roleHandler,
		permissionHandler:        permissionHandler,
		policyHandler:            policyHandler,
		k8sScopedPolicyHandler:   k8sScopedPolicyHandler,
		k8sNamespaceDenyHandler:  k8sNamespaceDenyHandler,
		k8sNamespaceAllowHandler: k8sNamespaceAllowHandler,
		userGroupHandler:         userGroupHandler,
		regHandler:               regHandler,
		menuHandler:              menuHandler,
		dictEntryHandler:         dictEntryHandler,
		adminHandler:             adminHandler,

		alertHandler:              alertHandler,
		alertPlatformHandler:      alertPlatformHandler,
		alertSubscriptionHandler:  alertSubscriptionHandler,
		alertInhibitionHandler:    alertInhibitionHandler,
		alertReceiverGroupHandler: alertReceiverGroupHandler,
		cloudExpiryRuleHandler:    cloudExpiryRuleHandler,

		clusterHandler:          clusterHandler,
		podHandler:              podHandler,
		namespaceHandler:        namespaceHandler,
		nodeHandler:             nodeHandler,
		workloadHandler:         workloadHandler,
		configHandler:           configHandler,
		storageHandler:          storageHandler,
		serviceResourceHandler:  serviceResourceHandler,
		ingressHandler:          ingressHandler,
		networkPolicyHandler:    networkPolicyHandler,
		k8sDiscoveryHandler:     k8sDiscoveryHandler,
		k8sHPAHandler:           k8sHPAHandler,
		k8sResourceWatchHandler: k8sResourceWatchHandler,
		k8sEventForwardHandler:  k8sEventForwardHandler,
		eventHandler:            eventHandler,
		crdHandler:              crdHandler,
		crHandler:               crHandler,
		rbacHandler:             rbacHandler,
		serviceAccountHandler:   serviceAccountHandler,
		overviewHandler:         overviewHandler,

		projectHandler:        projectHandler,
		logAgentHandler:       logAgentHandler,
		agentDiscoveryHandler: agentDiscoveryHandler,
	}
}
