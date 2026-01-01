package vault

import (
	"fmt"
	"strings"

	"github.com/hashicorp/vault/api"
)

func (c *Client) Restore(backup *BackupData, opts RestoreOptions) error {
	// Restore secret engines first
	fmt.Println("\nRestoring secret engines...")
	if err := c.restoreSecretEngines(backup, opts.Engines); err != nil {
		return fmt.Errorf("failed to restore secret engines: %w", err)
	}

	// Restore policies
	if !opts.SkipPolicies {
		fmt.Println("\nRestoring policies...")
		if err := c.restorePolicies(backup); err != nil {
			return fmt.Errorf("failed to restore policies: %w", err)
		}
	}

	// Restore auth methods
	if !opts.SkipAuth {
		fmt.Println("\nRestoring auth methods...")
		if err := c.restoreAuthMethods(backup, opts); err != nil {
			return fmt.Errorf("failed to restore auth methods: %w", err)
		}
	}

	return nil
}

func (c *Client) restoreSecretEngines(backup *BackupData, filterEngines []string) error {
	for _, engine := range backup.SecretEngines {
		// Filter engines if specified
		if len(filterEngines) > 0 && !contains(filterEngines, strings.TrimSuffix(engine.Path, "/")) {
			continue
		}

		fmt.Printf("  Restoring engine: %s (type: %s)\n", engine.Path, engine.Type)

		// Check if mount exists
		mounts, err := c.client.Sys().ListMounts()
		if err != nil {
			return err
		}

		mountExists := false
		for path := range mounts {
			if path == engine.Path {
				mountExists = true
				break
			}
		}

		// Create mount if it doesn't exist
		if !mountExists {
			mountInput := &api.MountInput{
				Type:        engine.Type,
				Description: engine.Description,
				Config:      api.MountConfigInput{},
				Options:     convertInterfaceMapToString(engine.Options),
			}

			if err := c.client.Sys().Mount(strings.TrimSuffix(engine.Path, "/"), mountInput); err != nil {
				fmt.Printf("    Warning: failed to create mount %s: %v\n", engine.Path, err)
				continue
			}
		}

		// Restore secrets
		if engine.Type == "kv" || engine.Type == "generic" {
			version := 1
			if engine.Options != nil && engine.Options["version"] == "2" {
				version = 2
			}

			if version == 2 {
				if err := c.restoreKVv2Secrets(engine.Path, engine.Secrets); err != nil {
					fmt.Printf("    Warning: failed to restore secrets to %s: %v\n", engine.Path, err)
				} else {
					fmt.Printf("    Restored %d secrets\n", len(engine.Secrets))
				}
			} else {
				if err := c.restoreKVv1Secrets(engine.Path, engine.Secrets); err != nil {
					fmt.Printf("    Warning: failed to restore secrets to %s: %v\n", engine.Path, err)
				} else {
					fmt.Printf("    Restored %d secrets\n", len(engine.Secrets))
				}
			}
		}
	}

	return nil
}

func (c *Client) restoreKVv2Secrets(mountPath string, secrets []SecretBackup) error {
	for _, secret := range secrets {
		// Restore versions in order
		for _, version := range secret.Versions {
			if version.Destroyed {
				continue // Skip destroyed versions
			}

			dataPath := mountPath + "data/" + secret.Path
			data := map[string]interface{}{
				"data": version.Data,
			}

			// Don't use CAS for version control during restore - just write sequentially
			// The versions will be created in order automatically

			_, err := c.client.Logical().Write(dataPath, data)
			if err != nil {
				fmt.Printf("      Warning: failed to restore %s version %d: %v\n", secret.Path, version.Version, err)
			}
		}

		// Update metadata if needed
		if secret.Metadata.MaxVersions > 0 || secret.Metadata.CasRequired || len(secret.Metadata.CustomMetadata) > 0 {
			metadataPath := mountPath + "metadata/" + secret.Path
			metadataData := map[string]interface{}{}

			if secret.Metadata.MaxVersions > 0 {
				metadataData["max_versions"] = secret.Metadata.MaxVersions
			}
			if secret.Metadata.CasRequired {
				metadataData["cas_required"] = true
			}
			if len(secret.Metadata.CustomMetadata) > 0 {
				metadataData["custom_metadata"] = secret.Metadata.CustomMetadata
			}
			if secret.Metadata.DeleteVersionAfter != "" {
				metadataData["delete_version_after"] = secret.Metadata.DeleteVersionAfter
			}

			if len(metadataData) > 0 {
				_, err := c.client.Logical().Write(metadataPath, metadataData)
				if err != nil {
					fmt.Printf("      Warning: failed to update metadata for %s: %v\n", secret.Path, err)
				}
			}
		}
	}

	return nil
}

func (c *Client) restoreKVv1Secrets(mountPath string, secrets []SecretBackup) error {
	for _, secret := range secrets {
		if len(secret.Versions) == 0 {
			continue
		}

		// KV v1 only has one version
		version := secret.Versions[len(secret.Versions)-1]
		secretPath := mountPath + secret.Path

		_, err := c.client.Logical().Write(secretPath, version.Data)
		if err != nil {
			fmt.Printf("      Warning: failed to restore %s: %v\n", secret.Path, err)
		}
	}

	return nil
}

func (c *Client) restorePolicies(backup *BackupData) error {
	for _, policy := range backup.Policies {
		if err := c.client.Sys().PutPolicy(policy.Name, policy.Policy); err != nil {
			fmt.Printf("  Warning: failed to restore policy %s: %v\n", policy.Name, err)
			continue
		}
	}

	fmt.Printf("  Restored %d policies\n", len(backup.Policies))
	return nil
}

func (c *Client) restoreAuthMethods(backup *BackupData, opts RestoreOptions) error {
	for _, auth := range backup.AuthMethods {
		fmt.Printf("  Restoring auth method: %s (type: %s)\n", auth.Path, auth.Type)

		// Check if auth method exists
		auths, err := c.client.Sys().ListAuth()
		if err != nil {
			return err
		}

		authExists := false
		for path := range auths {
			if path == auth.Path {
				authExists = true
				break
			}
		}

		// Enable auth method if it doesn't exist
		if !authExists {
			enableInput := &api.EnableAuthOptions{
				Type:        auth.Type,
				Description: auth.Description,
				Config:      api.AuthConfigInput{},
				Options:     convertInterfaceMapToString(auth.Options),
			}

			if err := c.client.Sys().EnableAuthWithOptions(strings.TrimSuffix(auth.Path, "/"), enableInput); err != nil {
				fmt.Printf("    Warning: failed to enable auth method %s: %v\n", auth.Path, err)
				continue
			}
		}

		// Restore roles and users
		switch auth.Type {
		case "userpass":
			if err := c.restoreUserpassUsers(auth.Path, auth.Users, opts.DefaultPassword); err != nil {
				fmt.Printf("    Warning: failed to restore userpass users: %v\n", err)
			}
		case "approle":
			if err := c.restoreAppRoles(auth.Path, auth.Roles); err != nil {
				fmt.Printf("    Warning: failed to restore approles: %v\n", err)
			}
		case "ldap":
			if err := c.restoreLDAPUsers(auth.Path, auth.Users); err != nil {
				fmt.Printf("    Warning: failed to restore LDAP users: %v\n", err)
			}
		}
	}

	fmt.Printf("  Restored %d auth methods\n", len(backup.AuthMethods))
	return nil
}

func (c *Client) restoreUserpassUsers(authPath string, users []UserBackup, defaultPassword string) error {
	basePath := "auth/" + strings.TrimSuffix(authPath, "/") + "/users"
	
	for _, user := range users {
		userPath := basePath + "/" + user.Name
		
		// Add default password to user data
		userData := make(map[string]interface{})
		for k, v := range user.Data {
			userData[k] = v
		}
		userData["password"] = defaultPassword
		
		_, err := c.client.Logical().Write(userPath, userData)
		if err != nil {
			fmt.Printf("      Warning: failed to restore user %s: %v\n", user.Name, err)
		}
	}

	return nil
}

func (c *Client) restoreAppRoles(authPath string, roles []RoleBackup) error {
	basePath := "auth/" + strings.TrimSuffix(authPath, "/") + "/role"
	
	for _, role := range roles {
		rolePath := basePath + "/" + role.Name
		_, err := c.client.Logical().Write(rolePath, role.Data)
		if err != nil {
			fmt.Printf("      Warning: failed to restore role %s: %v\n", role.Name, err)
		}
	}

	return nil
}

func (c *Client) restoreLDAPUsers(authPath string, users []UserBackup) error {
	basePath := "auth/" + strings.TrimSuffix(authPath, "/") + "/users"
	
	for _, user := range users {
		userPath := basePath + "/" + user.Name
		_, err := c.client.Logical().Write(userPath, user.Data)
		if err != nil {
			fmt.Printf("      Warning: failed to restore user %s: %v\n", user.Name, err)
		}
	}

	return nil
}

func convertInterfaceMapToString(m map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		if str, ok := v.(string); ok {
			result[k] = str
		} else {
			result[k] = fmt.Sprintf("%v", v)
		}
	}
	return result
}
