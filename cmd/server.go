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

	"go-permission-system/internal/bootstrap"
	grpcclient "go-permission-system/internal/grpc/client"
	grpcserver "go-permission-system/internal/grpc/server"
	"go-permission-system/internal/repository"
	"go-permission-system/internal/router"
	"go-permission-system/internal/service"

	"github.com/spf13/cobra"
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

		if err := bootstrap.AutoMigrateModels(app.DB); err != nil {
			return fmt.Errorf("auto migrate: %w", err)
		}
		app.Logger.Info.Info("database schema migrated")

		projectRepo := repository.NewProjectRepository(app.DB)
		serverRepo := repository.NewServerRepository(app.DB)
		serverGroupRepo := repository.NewServerGroupRepository(app.DB)
		cloudAccountRepo := repository.NewCloudAccountRepository(app.DB)
		serviceRepo := repository.NewServiceRepository(app.DB)
		logRepo := repository.NewLogSourceRepository(app.DB)
		logAgentRepo := repository.NewLogAgentRepository(app.DB)
		agentDiscoveryRepo := repository.NewAgentDiscoveryRepository(app.DB)

		userRepo := repository.NewUserRepository(app.DB)
		projectMemberRepo := repository.NewProjectMemberRepository(app.DB)
		projectSvc, err := service.NewProjectMgmtService(projectRepo, serverRepo, serverGroupRepo, cloudAccountRepo, serviceRepo, logRepo, projectMemberRepo, userRepo, app.Config.Security.EncryptionKey)
		if err != nil {
			return err
		}
		agentSvc := service.NewLogAgentService(logAgentRepo, serverRepo, logRepo, app.Config.Agent.RegisterSecret)
		discoverySvc := service.NewAgentDiscoveryService(agentDiscoveryRepo, logAgentRepo, serverRepo)

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

		runtimeClient, err := grpcclient.Dial(
			app.Config.GRPC.TargetAddr,
			5*time.Second,
			app.Config.GRPC.MaxRecvMsgBytes,
			app.Config.GRPC.MaxSendMsgBytes,
		)
		if err != nil {
			return fmt.Errorf("dial grpc runtime: %w", err)
		}
		defer runtimeClient.Close()

		router.Register(app, runtimeClient)

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
			app.Logger.Info.Info("permission system server started", "addr", server.Addr)
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
		}()

		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

		select {
		case sig := <-stop:
			app.Logger.Info.Info("received shutdown signal", "signal", sig.String())
		case err := <-errCh:
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		grpcRuntime.Stop(ctx)
		return server.Shutdown(ctx)
	},
}
