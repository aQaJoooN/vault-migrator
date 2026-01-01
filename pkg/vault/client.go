package vault

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
)

type Client struct {
	client *api.Client
}

func NewClient(address, token string) (*Client, error) {
	config := api.DefaultConfig()
	config.Address = address

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	client.SetToken(token)

	return &Client{client: client}, nil
}

func (c *Client) Backup(engines []string) (*BackupData, error) {
	backup := &BackupData{
		Timestamp: time.Now(),
	}

	// Get Vault version
	health, err := c.client.Sys().Health()
	if err == nil {
		backup.VaultVersion = health.Version
	}

	// Backup secret engines
	fmt.Println("\nBacking up secret engines...")
	if err := c.backupSecretEngines(backup, engines); err != nil {
		return nil, fmt.Errorf("failed to backup secret engines: %w", err)
	}

	// Backup policies
	fmt.Println("\nBacking up policies...")
	if err := c.backupPolicies(backup); err != nil {
		return nil, fmt.Errorf("failed to backup policies: %w", err)
	}

	// Backup auth methods
	fmt.Println("\nBacking up auth methods...")
	if err := c.backupAuthMethods(backup); err != nil {
		return nil, fmt.Errorf("failed to backup auth methods: %w", err)
	}

	return backup, nil
}

func (c *Client) backupSecretEngines(backup *BackupData, filterEngines []string) error {
	mounts, err := c.client.Sys().ListMounts()
	if err != nil {
		return err
	}

	for path, mount := range mounts {
		// Skip system mounts
		if strings.HasPrefix(path, "sys/") || strings.HasPrefix(path, "identity/") || strings.HasPrefix(path, "cubbyhole/") {
			continue
		}

		// Filter engines if specified
		if len(filterEngines) > 0 && !contains(filterEngines, strings.TrimSuffix(path, "/")) {
			continue
		}

		fmt.Printf("  Processing engine: %s (type: %s)\n", path, mount.Type)

		engineBackup := SecretEngineBackup{
			Path:        path,
			Type:        mount.Type,
			Description: mount.Description,
			Config:      convertToMap(mount.Config),
			Options:     convertStringMapToInterface(mount.Options),
		}

		// Backup secrets based on engine type
		if mount.Type == "kv" || mount.Type == "generic" {
			version := 1
			if mount.Options != nil && mount.Options["version"] == "2" {
				version = 2
			}
			
			if version == 2 {
				secrets, err := c.backupKVv2Secrets(path)
				if err != nil {
					fmt.Printf("    Warning: failed to backup secrets from %s: %v\n", path, err)
				} else {
					engineBackup.Secrets = secrets
					fmt.Printf("    Backed up %d secrets\n", len(secrets))
				}
			} else {
				secrets, err := c.backupKVv1Secrets(path)
				if err != nil {
					fmt.Printf("    Warning: failed to backup secrets from %s: %v\n", path, err)
				} else {
					engineBackup.Secrets = secrets
					fmt.Printf("    Backed up %d secrets\n", len(secrets))
				}
			}
		}

		backup.SecretEngines = append(backup.SecretEngines, engineBackup)
	}

	return nil
}

func (c *Client) backupKVv2Secrets(mountPath string) ([]SecretBackup, error) {
	var secrets []SecretBackup
	
	paths, err := c.listAllPaths(mountPath, "metadata/")
	if err != nil {
		return nil, err
	}
	
	if len(paths) == 0 {
		return secrets, nil
	}

	for _, path := range paths {
		// Get metadata
		metadataPath := mountPath + "metadata/" + path
		metadataResp, err := c.client.Logical().Read(metadataPath)
		if err != nil || metadataResp == nil {
			continue
		}

		secretBackup := SecretBackup{
			Path: path,
		}

		// Parse metadata
		if metadataResp.Data != nil {
			secretBackup.Metadata = parseMetadata(metadataResp.Data)
		}

		// Get all versions
		versions := secretBackup.Metadata.CurrentVersion
		if versions == 0 {
			continue
		}
		
		for v := 1; v <= versions; v++ {
			dataPath := fmt.Sprintf("%sdata/%s", mountPath, path)
			
			// Use ReadWithData to pass version as a query parameter
			versionResp, err := c.client.Logical().ReadWithData(dataPath, map[string][]string{
				"version": {fmt.Sprintf("%d", v)},
			})
			if err != nil || versionResp == nil {
				continue
			}

			if versionResp.Data != nil && versionResp.Data["data"] != nil {
				version := SecretVersion{
					Version: v,
					Data:    versionResp.Data["data"].(map[string]interface{}),
				}

				if metadata, ok := versionResp.Data["metadata"].(map[string]interface{}); ok {
					if ct, ok := metadata["created_time"].(string); ok {
						version.CreatedTime, _ = time.Parse(time.RFC3339, ct)
					}
					if dt, ok := metadata["deletion_time"].(string); ok {
						version.DeletionTime = dt
					}
					if destroyed, ok := metadata["destroyed"].(bool); ok {
						version.Destroyed = destroyed
					}
				}

				secretBackup.Versions = append(secretBackup.Versions, version)
			}
		}

		if len(secretBackup.Versions) > 0 {
			secrets = append(secrets, secretBackup)
		}
	}

	return secrets, nil
}

func (c *Client) backupKVv1Secrets(mountPath string) ([]SecretBackup, error) {
	var secrets []SecretBackup
	
	paths, err := c.listAllPaths(mountPath, "")
	if err != nil {
		return nil, err
	}

	for _, path := range paths {
		secretPath := mountPath + path
		resp, err := c.client.Logical().Read(secretPath)
		if err != nil || resp == nil || resp.Data == nil {
			continue
		}

		secretBackup := SecretBackup{
			Path: path,
			Versions: []SecretVersion{
				{
					Version:     1,
					Data:        resp.Data,
					CreatedTime: time.Now(),
				},
			},
			Metadata: SecretMetadata{
				CurrentVersion: 1,
				MaxVersions:    1,
			},
		}

		secrets = append(secrets, secretBackup)
	}

	return secrets, nil
}

func (c *Client) listAllPaths(mountPath, prefix string) ([]string, error) {
	var allPaths []string
	
	listPath := mountPath + prefix
	resp, err := c.client.Logical().List(listPath)
	if err != nil || resp == nil {
		return allPaths, nil
	}

	if keys, ok := resp.Data["keys"].([]interface{}); ok {
		for _, key := range keys {
			keyStr := key.(string)
			
			if strings.HasSuffix(keyStr, "/") {
				// It's a directory, recurse
				subPaths, err := c.listAllPaths(mountPath, prefix+keyStr)
				if err == nil {
					for _, subPath := range subPaths {
						allPaths = append(allPaths, keyStr+subPath)
					}
				}
			} else {
				allPaths = append(allPaths, keyStr)
			}
		}
	}

	return allPaths, nil
}

func (c *Client) backupPolicies(backup *BackupData) error {
	policies, err := c.client.Sys().ListPolicies()
	if err != nil {
		return err
	}

	for _, policyName := range policies {
		// Skip default policies
		if policyName == "root" || policyName == "default" {
			continue
		}

		policy, err := c.client.Sys().GetPolicy(policyName)
		if err != nil {
			fmt.Printf("  Warning: failed to get policy %s: %v\n", policyName, err)
			continue
		}

		backup.Policies = append(backup.Policies, PolicyBackup{
			Name:   policyName,
			Policy: policy,
		})
	}

	fmt.Printf("  Backed up %d policies\n", len(backup.Policies))
	return nil
}

func (c *Client) backupAuthMethods(backup *BackupData) error {
	auths, err := c.client.Sys().ListAuth()
	if err != nil {
		return err
	}

	for path, auth := range auths {
		// Skip token auth (always present)
		if path == "token/" {
			continue
		}

		fmt.Printf("  Processing auth method: %s (type: %s)\n", path, auth.Type)

		authBackup := AuthMethodBackup{
			Path:        path,
			Type:        auth.Type,
			Description: auth.Description,
			Config:      convertToMap(auth.Config),
			Options:     convertStringMapToInterface(auth.Options),
		}

		// Backup roles and users based on auth type
		switch auth.Type {
		case "userpass":
			users, err := c.backupUserpassUsers(path)
			if err == nil {
				authBackup.Users = users
			}
		case "approle":
			roles, err := c.backupAppRoles(path)
			if err == nil {
				authBackup.Roles = roles
			}
		case "ldap":
			users, err := c.backupLDAPUsers(path)
			if err == nil {
				authBackup.Users = users
			}
		}

		backup.AuthMethods = append(backup.AuthMethods, authBackup)
	}

	fmt.Printf("  Backed up %d auth methods\n", len(backup.AuthMethods))
	return nil
}

func (c *Client) backupUserpassUsers(authPath string) ([]UserBackup, error) {
	var users []UserBackup
	
	listPath := "auth/" + strings.TrimSuffix(authPath, "/") + "/users"
	resp, err := c.client.Logical().List(listPath)
	if err != nil || resp == nil {
		return users, nil
	}

	if keys, ok := resp.Data["keys"].([]interface{}); ok {
		for _, key := range keys {
			username := key.(string)
			userPath := listPath + "/" + username
			userResp, err := c.client.Logical().Read(userPath)
			if err == nil && userResp != nil {
				users = append(users, UserBackup{
					Name: username,
					Data: userResp.Data,
				})
			}
		}
	}

	return users, nil
}

func (c *Client) backupAppRoles(authPath string) ([]RoleBackup, error) {
	var roles []RoleBackup
	
	listPath := "auth/" + strings.TrimSuffix(authPath, "/") + "/role"
	resp, err := c.client.Logical().List(listPath)
	if err != nil || resp == nil {
		return roles, nil
	}

	if keys, ok := resp.Data["keys"].([]interface{}); ok {
		for _, key := range keys {
			roleName := key.(string)
			rolePath := listPath + "/" + roleName
			roleResp, err := c.client.Logical().Read(rolePath)
			if err == nil && roleResp != nil {
				roles = append(roles, RoleBackup{
					Name: roleName,
					Data: roleResp.Data,
				})
			}
		}
	}

	return roles, nil
}

func (c *Client) backupLDAPUsers(authPath string) ([]UserBackup, error) {
	var users []UserBackup
	
	listPath := "auth/" + strings.TrimSuffix(authPath, "/") + "/users"
	resp, err := c.client.Logical().List(listPath)
	if err != nil || resp == nil {
		return users, nil
	}

	if keys, ok := resp.Data["keys"].([]interface{}); ok {
		for _, key := range keys {
			username := key.(string)
			userPath := listPath + "/" + username
			userResp, err := c.client.Logical().Read(userPath)
			if err == nil && userResp != nil {
				users = append(users, UserBackup{
					Name: username,
					Data: userResp.Data,
				})
			}
		}
	}

	return users, nil
}

func parseMetadata(data map[string]interface{}) SecretMetadata {
	metadata := SecretMetadata{}

	if v, ok := data["cas_required"].(bool); ok {
		metadata.CasRequired = v
	}
	if v, ok := data["created_time"].(string); ok {
		metadata.CreatedTime, _ = time.Parse(time.RFC3339, v)
	}
	
	// Handle current_version - can be int, float64, or json.Number
	if v, ok := data["current_version"].(int); ok {
		metadata.CurrentVersion = v
	} else if v, ok := data["current_version"].(float64); ok {
		metadata.CurrentVersion = int(v)
	} else if v, ok := data["current_version"].(json.Number); ok {
		val, _ := v.Int64()
		metadata.CurrentVersion = int(val)
	} else {
		// Try to convert from any numeric type
		fmt.Printf("        Debug: current_version type: %T, value: %v\n", data["current_version"], data["current_version"])
	}
	
	// Handle max_versions
	if v, ok := data["max_versions"].(int); ok {
		metadata.MaxVersions = v
	} else if v, ok := data["max_versions"].(float64); ok {
		metadata.MaxVersions = int(v)
	} else if v, ok := data["max_versions"].(json.Number); ok {
		val, _ := v.Int64()
		metadata.MaxVersions = int(val)
	}
	
	// Handle oldest_version
	if v, ok := data["oldest_version"].(int); ok {
		metadata.OldestVersion = v
	} else if v, ok := data["oldest_version"].(float64); ok {
		metadata.OldestVersion = int(v)
	} else if v, ok := data["oldest_version"].(json.Number); ok {
		val, _ := v.Int64()
		metadata.OldestVersion = int(val)
	}
	
	if v, ok := data["updated_time"].(string); ok {
		metadata.UpdatedTime, _ = time.Parse(time.RFC3339, v)
	}
	if v, ok := data["custom_metadata"].(map[string]string); ok {
		metadata.CustomMetadata = v
	}
	if v, ok := data["delete_version_after"].(string); ok {
		metadata.DeleteVersionAfter = v
	}

	return metadata
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func convertToMap(v interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	if v == nil {
		return result
	}
	
	// Use type assertion or reflection to convert
	switch val := v.(type) {
	case map[string]interface{}:
		return val
	case map[string]string:
		for k, v := range val {
			result[k] = v
		}
	}
	return result
}

func convertStringMapToInterface(m map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return result
}
