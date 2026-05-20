package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"yunshu/internal/bootstrap"
	grpcclient "yunshu/internal/grpc/client"
	grpcserver "yunshu/internal/grpc/server"
	"yunshu/internal/handler"
	"yunshu/internal/model"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/password"
	"yunshu/internal/repository"
	"yunshu/internal/router"
	"yunshu/internal/service"

	"github.com/casbin/casbin/v2"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start permission system server",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := bootstrap.NewBuilder().
			WithConfig(configPath).
			WithLogger().
			WithMySQL().
			WithDictOverrides().
			WithRedis().
			WithMailer().
			WithCasbin().
			WithGin().
			Build()
		if err != nil {
			return err
		}
		defer app.Close()

		logx.Init(app.Logger)
		handler.SetLogger(app.Logger)

		if err := bootstrap.AutoMigrateModels(app.DB); err != nil {
			return fmt.Errorf("auto migrate: %w", err)
		}
		bootLog := app.Logger.Biz("bootstrap")
		bootLog.Infow("Database schema migrated")
		if err := app.Enforcer.LoadPolicy(); err != nil {
			return fmt.Errorf("reload casbin policy: %w", err)
		}

		// 初始化只读演示用户
		ctx := context.Background()
		if err := initReadonlyDemoUser(ctx, app.DB, app.Enforcer, bootLog); err != nil {
			bootLog.Errorw(err, "Failed to init readonly demo user")
			// 非致命错误，继续启动
		}

		projectRepo := repository.NewProjectRepository(app.DB)
		serverRepo := repository.NewServerRepository(app.DB)
		serverGroupRepo := repository.NewServerGroupRepository(app.DB)
		cloudAccountRepo := repository.NewCloudAccountRepository(app.DB)
		serviceRepo := repository.NewServiceRepository(app.DB)
		logRepo := repository.NewLogSourceRepository(app.DB)
		logAgentRepo := repository.NewLogAgentRepository(app.DB)
		agentDiscoveryRepo := repository.NewAgentDiscoveryRepository(app.DB)

		userRepo := repository.NewUserRepository(app.DB)
		departmentRepo := repository.NewDepartmentRepository(app.DB)
		projectMemberRepo := repository.NewProjectMemberRepository(app.DB)
		projectSvc, err := service.NewProjectMgmtService(projectRepo, serverRepo, serverGroupRepo, cloudAccountRepo, serviceRepo, logRepo, projectMemberRepo, userRepo, departmentRepo, app.Config.Security.EncryptionKey)
		if err != nil {
			return err
		}
		agentSvc := service.NewLogAgentService(logAgentRepo, serverRepo, logRepo, app.Config.Agent.RegisterSecret, app.Config.Agent.DiscoveryRoots)
		discoverySvc := service.NewAgentDiscoveryService(agentDiscoveryRepo, logAgentRepo, serverRepo, logRepo)

		grpcImpl := grpcserver.NewLogPlatformServer(projectSvc, agentSvc, discoverySvc)
		grpcRuntime, err := grpcserver.Start(
			app.Config.GRPC.ListenAddr,
			grpcImpl,
			app.Config.GRPC.MaxRecvMsgBytes,
			app.Config.GRPC.MaxSendMsgBytes,
		)
		if err != nil {
			return fmt.Errorf("start grpc server: %w", err)
		}

		grpcCallTimeout := time.Duration(app.Config.GRPC.CallTimeoutSeconds) * time.Second
		if grpcCallTimeout <= 0 {
			grpcCallTimeout = 30 * time.Second
		}
		runtimeClient, err := grpcclient.Dial(
			app.Config.GRPC.TargetAddr,
			5*time.Second,
			app.Config.GRPC.MaxRecvMsgBytes,
			app.Config.GRPC.MaxSendMsgBytes,
			grpcCallTimeout,
		)
		if err != nil {
			return fmt.Errorf("dial grpc runtime: %w", err)
		}
		defer runtimeClient.Close()

		bgWorkersCtx, bgWorkersCancel := context.WithCancel(context.Background())
		defer bgWorkersCancel()

		k8sEventForwardMgr := router.Register(app, runtimeClient, bgWorkersCtx)
		if k8sEventForwardMgr != nil {
			defer k8sEventForwardMgr.Stop()
		}

		sweepCtx, sweepCancel := context.WithCancel(context.Background())
		defer sweepCancel()
		go func() {
			ticker := time.NewTicker(45 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-sweepCtx.Done():
					return
				case <-ticker.C:
					ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
					err := agentSvc.RecordOfflineEpisodes(ctx)
					cancel()
					if err != nil {
						app.Logger.Biz("agent").Warnw("Failed to record agent offline episodes", "error", err)
					}
				}
			}
		}()

		server := &http.Server{
			Addr:              fmt.Sprintf(":%d", app.Config.App.Port),
			Handler:           app.Engine,
			ReadHeaderTimeout: time.Duration(app.Config.HTTP.ReadHeaderTimeoutSeconds) * time.Second,
			ReadTimeout:       time.Duration(app.Config.HTTP.ReadTimeoutSeconds) * time.Second,
			WriteTimeout:      time.Duration(app.Config.HTTP.WriteTimeoutSeconds) * time.Second,
			IdleTimeout:       time.Duration(app.Config.HTTP.IdleTimeoutSeconds) * time.Second,
		}

		errCh := make(chan error, 1)
		go func() {
			app.Logger.Biz("server").Infow("HTTP server started", "addr", server.Addr)
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
		}()

		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

		select {
		case sig := <-stop:
			app.Logger.Biz("server").Infow("Received shutdown signal", "signal", sig.String())
		case err := <-errCh:
			return err
		}

		grpcShutdown := time.Duration(app.Config.GRPC.ShutdownTimeoutSeconds) * time.Second
		if grpcShutdown <= 0 {
			grpcShutdown = 5 * time.Second
		}
		httpShutdown := time.Duration(app.Config.HTTP.ShutdownTimeoutSeconds) * time.Second
		if httpShutdown <= 0 {
			httpShutdown = 10 * time.Second
		}

		ctxGRPC, cancelGRPC := context.WithTimeout(context.Background(), grpcShutdown)
		defer cancelGRPC()
		grpcRuntime.Stop(ctxGRPC)

		ctxHTTP, cancelHTTP := context.WithTimeout(context.Background(), httpShutdown)
		defer cancelHTTP()
		defer logx.Sync()
		return server.Shutdown(ctxHTTP)
	},
}

// initReadonlyDemoUser 初始化只读演示用户
// 用户名: viewer, 密码: viewer123, 角色: viewer (仅查看权限)
func initReadonlyDemoUser(ctx context.Context, db *gorm.DB, enforcer *casbin.SyncedEnforcer, logger *logx.Component) error {
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	permRepo := repository.NewPermissionRepository(db)

	// 1. 检查或创建 viewer 角色
	roleCode := "viewer"
	roleName := "演示查看员"
	var role *model.Role
	allRoles, err := roleRepo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("list roles: %w", err)
	}
	for i := range allRoles {
		if allRoles[i].Code == roleCode {
			role = &allRoles[i]
			break
		}
	}

	// 角色不存在则创建
	if role == nil {
		role = &model.Role{
			Code:        roleCode,
			Name:        roleName,
			Description: "仅拥有查看权限的演示角色",
		}
		if err := db.Create(role).Error; err != nil {
			return fmt.Errorf("create role: %w", err)
		}
		logger.Infow("Created readonly role", "code", roleCode)
	}

	// 2. 配置角色权限：只读 GET 权限 + K8s 资源查看
	// 先清除旧权限
	if _, err := enforcer.RemoveFilteredPolicy(0, roleCode); err != nil {
		logger.Warnw("Failed to remove old Casbin policies", "error", err)
	}

	perms, err := permRepo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("list permissions: %w", err)
	}

	added := 0
	for _, p := range perms {
		if p.Action != "GET" {
			continue
		}
		obj := p.Resource
		if obj == "" {
			continue
		}
		if _, err := enforcer.AddPolicy(roleCode, obj, "GET"); err != nil {
			logger.Warnw("Failed to add Casbin policy", "resource", obj, "error", err)
			continue
		}
		added++
	}

	accessRepo := repository.NewK8sClusterAccessRepository(db)
	if err := accessRepo.Upsert(ctx, &model.K8sClusterAccessGrant{
		PrincipalKind: model.K8sPrincipalRole,
		PrincipalRef:  roleCode,
		ClusterID:     0,
		Preset:        "readonly",
	}); err != nil {
		return fmt.Errorf("upsert k8s cluster access grant: %w", err)
	}

	logger.Infow("Configured readonly role permissions", "role", roleCode, "policies_added", added)

	// 3. 检查或创建演示用户
	username := "viewer"
	email := "viewer@yunshu.demo"
	plainPassword := "viewer123"

	user, err := userRepo.GetByUsername(ctx, username)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("get user: %w", err)
		}
		// 用户不存在，创建新用户
		hashedPassword, err := password.Hash(plainPassword)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}

		user = &model.User{
			Username: username,
			Email:    &email,
			Password: hashedPassword,
			Status:   1,
		}
		if err := db.Create(user).Error; err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		logger.Infow("Created demo user", "username", username)
	} else {
		logger.Infow("Demo user already exists", "username", username)
	}

	// 4. 绑定用户到 viewer 角色
	if err := userRepo.ReplaceRoles(ctx, user, []model.Role{*role}); err != nil {
		return fmt.Errorf("bind role to user: %w", err)
	}

	// 同步 Casbin 权限
	if err := service.SyncUserRoles(enforcer, user.ID, []model.Role{*role}); err != nil {
		return fmt.Errorf("sync user roles: %w", err)
	}

	logger.Infow("Initialized demo user", "username", username, "password", plainPassword, "role", roleCode)
	logger.Infow("Demo user login info", "username", username, "password", plainPassword)
	return nil
}
