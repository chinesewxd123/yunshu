package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var configPath string

var rootCmd = &cobra.Command{
	Use:   "permission-system",
	Short: "A permission management system built with Gin, MySQL, Redis and Casbin",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "configs/config.yaml", "config file path")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
