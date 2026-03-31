package cmd

import (
	"go-permission-system/internal/bootstrap"
	"go-permission-system/internal/model"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(migrateCmd)
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := bootstrap.NewBuilder().
			WithConfig(configPath).
			WithLogger().
			WithMySQL().
			WithCasbin().
			Build()
		if err != nil {
			return err
		}
		defer app.Close()

		return app.DB.AutoMigrate(
			&model.User{},
			&model.Role{},
			&model.Permission{},
			&model.UserRole{},
		)
	},
}
