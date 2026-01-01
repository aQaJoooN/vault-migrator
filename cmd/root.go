package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vault-migrator",
	Short: "Migrate secrets, policies, and auth methods between Vault servers",
	Long:  `A tool to backup and restore HashiCorp Vault configurations including secrets with all versions, policies, auth methods, and access configurations.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)
}
