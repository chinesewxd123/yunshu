package cmd

import (
	"go-permission-system/internal/bootstrap"

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
			WithDictOverrides().
			WithCasbin().
			Build()
		if err != nil {
			return err
		}
		defer app.Close()

		return bootstrap.AutoMigrateModels(app.DB)
	},
}
