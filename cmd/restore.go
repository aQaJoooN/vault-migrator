package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"vault-migrator/pkg/vault"

	"github.com/spf13/cobra"
)

var (
	restoreFile       string
	restoreAddr       string
	restoreToken      string
	restoreEngines    []string
	skipPolicies      bool
	skipAuth          bool
	defaultPassword   string
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore Vault data from a backup file",
	Long:  `Restore all secrets (with versions), policies, auth methods, and configurations from a backup file to a Vault server.`,
	RunE:  runRestore,
}

func init() {
	restoreCmd.Flags().StringVarP(&restoreFile, "file", "f", "vault-backup.json", "Input backup file")
	restoreCmd.Flags().StringVarP(&restoreAddr, "address", "a", "", "Vault server address (or set VAULT_ADDR)")
	restoreCmd.Flags().StringVarP(&restoreToken, "token", "t", "", "Vault token (or set VAULT_TOKEN)")
	restoreCmd.Flags().StringSliceVarP(&restoreEngines, "engines", "e", []string{}, "Specific secret engines to restore (empty = all)")
	restoreCmd.Flags().BoolVar(&skipPolicies, "skip-policies", false, "Skip restoring policies")
	restoreCmd.Flags().BoolVar(&skipAuth, "skip-auth", false, "Skip restoring auth methods")
	restoreCmd.Flags().StringVarP(&defaultPassword, "default-password", "p", "ChangeMe123!", "Default password for restored users")
}

func runRestore(cmd *cobra.Command, args []string) error {
	addr := getEnvOrFlag(restoreAddr, "VAULT_ADDR")
	token := getEnvOrFlag(restoreToken, "VAULT_TOKEN")

	if addr == "" || token == "" {
		return fmt.Errorf("vault address and token are required")
	}

	data, err := os.ReadFile(restoreFile)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	var backup vault.BackupData
	if err := json.Unmarshal(data, &backup); err != nil {
		return fmt.Errorf("failed to parse backup file: %w", err)
	}

	fmt.Printf("Connecting to Vault at %s...\n", addr)
	client, err := vault.NewClient(addr, token)
	if err != nil {
		return fmt.Errorf("failed to create vault client: %w", err)
	}

	fmt.Println("Starting restore process...")
	
	opts := vault.RestoreOptions{
		Engines:         restoreEngines,
		SkipPolicies:    skipPolicies,
		SkipAuth:        skipAuth,
		DefaultPassword: defaultPassword,
	}

	if err := client.Restore(&backup, opts); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	fmt.Printf("\n✓ Restore completed successfully!\n")
	fmt.Printf("  Secret Engines: %d\n", len(backup.SecretEngines))
	fmt.Printf("  Total Secrets: %d\n", countSecrets(&backup))
	if !skipPolicies {
		fmt.Printf("  Policies: %d\n", len(backup.Policies))
	}
	if !skipAuth {
		fmt.Printf("  Auth Methods: %d\n", len(backup.AuthMethods))
	}

	return nil
}
