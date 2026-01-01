package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hashicorp/vault/api"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: update-passwords <vault-address> <vault-token> <users-file>")
		fmt.Println("Example: update-passwords https://betavault.asax.local:8200 hvs.xxx user.json")
		os.Exit(1)
	}

	vaultAddr := os.Args[1]
	vaultToken := os.Args[2]
	usersFile := os.Args[3]

	// Read users file
	data, err := os.ReadFile(usersFile)
	if err != nil {
		fmt.Printf("Error reading users file: %v\n", err)
		os.Exit(1)
	}

	var users map[string]string
	if err := json.Unmarshal(data, &users); err != nil {
		fmt.Printf("Error parsing users file: %v\n", err)
		os.Exit(1)
	}

	// Connect to Vault
	config := api.DefaultConfig()
	config.Address = vaultAddr
	client, err := api.NewClient(config)
	if err != nil {
		fmt.Printf("Error creating Vault client: %v\n", err)
		os.Exit(1)
	}
	client.SetToken(vaultToken)

	// Auth methods to check
	authMethods := []string{
		"userpass",
		"ats-auth",
		"cd-auth",
		"ecs-auth",
		"hrm-auth",
		"ime-auth",
		"mic-auth",
		"ams-auth",
	}

	fmt.Printf("Updating passwords for %d users...\n\n", len(users))

	successCount := 0
	failCount := 0

	for username, password := range users {
		updated := false
		
		for _, authMethod := range authMethods {
			userPath := fmt.Sprintf("auth/%s/users/%s/password", authMethod, username)
			
			data := map[string]interface{}{
				"password": password,
			}

			_, err := client.Logical().Write(userPath, data)
			if err == nil {
				fmt.Printf("✓ Updated %s in %s\n", username, authMethod)
				updated = true
				successCount++
				break
			}
		}

		if !updated {
			fmt.Printf("✗ Failed to update %s (user not found in any auth method)\n", username)
			failCount++
		}
	}

	fmt.Printf("\n✓ Successfully updated: %d users\n", successCount)
	if failCount > 0 {
		fmt.Printf("✗ Failed to update: %d users\n", failCount)
	}
}
