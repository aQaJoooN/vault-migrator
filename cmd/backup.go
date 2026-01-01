package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"vault-migrator/pkg/vault"

	"github.com/spf13/cobra"
)

var (
	backupFile    string
	backupAddr    string
	backupToken   string
	backupEngines []string
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup Vault data to a file",
	Long:  `Backup all secrets (with versions), policies, auth methods, and configurations from a Vault server to a JSON file.`,
	RunE:  runBackup,
}

func init() {
	backupCmd.Flags().StringVarP(&backupFile, "file", "f", "vault-backup.json", "Output backup file")
	backupCmd.Flags().StringVarP(&backupAddr, "address", "a", "", "Vault server address (or set VAULT_ADDR)")
	backupCmd.Flags().StringVarP(&backupToken, "token", "t", "", "Vault token (or set VAULT_TOKEN)")
	backupCmd.Flags().StringSliceVarP(&backupEngines, "engines", "e", []string{}, "Specific secret engines to backup (empty = all)")
}

func runBackup(cmd *cobra.Command, args []string) error {
	addr := getEnvOrFlag(backupAddr, "VAULT_ADDR")
	token := getEnvOrFlag(backupToken, "VAULT_TOKEN")

	if addr == "" || token == "" {
		return fmt.Errorf("vault address and token are required")
	}

	fmt.Printf("Connecting to Vault at %s...\n", addr)
	client, err := vault.NewClient(addr, token)
	if err != nil {
		return fmt.Errorf("failed to create vault client: %w", err)
	}

	fmt.Println("Starting backup process...")
	backup, err := client.Backup(backupEngines)
	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal backup data: %w", err)
	}

	if err := os.WriteFile(backupFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	fmt.Printf("\n✓ Backup completed successfully!\n")
	fmt.Printf("  File: %s\n", backupFile)
	fmt.Printf("  Secret Engines: %d\n", len(backup.SecretEngines))
	fmt.Printf("  Total Secrets: %d\n", countSecrets(backup))
	fmt.Printf("  Policies: %d\n", len(backup.Policies))
	fmt.Printf("  Auth Methods: %d\n", len(backup.AuthMethods))

	return nil
}

func countSecrets(backup *vault.BackupData) int {
	count := 0
	for _, engine := range backup.SecretEngines {
		count += len(engine.Secrets)
	}
	return count
}

func getEnvOrFlag(flag, envVar string) string {
	if flag != "" {
		return flag
	}
	return os.Getenv(envVar)
}
