# Vault Migrator

A comprehensive tool to backup and restore HashiCorp Vault configurations between servers, including:
- All secrets with complete version history (KV v1 and v2)
- Policies
- Auth methods (userpass, approle, LDAP, etc.)
- Secret engine configurations
- User password management

## Features

- **Complete Version History**: Backs up and restores all versions of secrets, not just the latest
- **Seamless Migration**: No one will notice the server change - all data is preserved
- **Flexible Filtering**: Backup/restore specific secret engines
- **Multiple Auth Methods**: Supports userpass, approle, LDAP, and more
- **Password Management**: Separate tool to update user passwords after migration
- **Safe Operations**: Validates before restore, provides detailed progress

## Installation

### From Source

```bash
cd vault-migrator
go mod download
go build -o vault-migrator
go build -o update-passwords update-passwords.go
```

### Pre-built Binaries

Download the latest release from the [Releases page](https://github.com/YOUR_USERNAME/vault-migrator/releases).

Available for:
- Linux (AMD64, ARM64)
- macOS (Intel, Apple Silicon)
- Windows (AMD64)

## Tools

### 1. vault-migrator

Main migration tool for backing up and restoring Vault data.

### 2. update-passwords

Utility to update user passwords from a JSON file after migration.

## Usage

### Backup from Old Vault

```bash
# Using environment variables
export VAULT_ADDR=https://old-vault.example.com
export VAULT_TOKEN=your-old-vault-token
./vault-migrator backup -f backup.json

# Or using flags
./vault-migrator backup \
  --address https://old-vault.example.com \
  --token your-old-vault-token \
  --file backup.json

# Backup specific engines only
./vault-migrator backup -f backup.json -e secret -e app-secrets
```

### Restore to New Vault

```bash
# Using environment variables
export VAULT_ADDR=https://new-vault.example.com
export VAULT_TOKEN=your-new-vault-token
./vault-migrator restore -f backup.json

# Or using flags
./vault-migrator restore \
  --address https://new-vault.example.com \
  --token your-new-vault-token \
  --file backup.json \
  --default-password "TempPass123!"

# Restore specific engines only
./vault-migrator restore -f backup.json -e secret -e app-secrets

# Skip policies or auth methods
./vault-migrator restore -f backup.json --skip-policies --skip-auth
```

### Update User Passwords

After migration, users are created with a default password. Use the `update-passwords` tool to restore original passwords:

```bash
# Create a user.json file with username:password pairs
{
  "user1": "password1",
  "user2": "password2"
}

# Update passwords
./update-passwords https://new-vault.example.com your-token user.json
```

The tool will update ALL occurrences of each user across all auth methods.

## Command Reference

### vault-migrator backup

```bash
vault-migrator backup [flags]

Flags:
  -a, --address string    Vault server address (or set VAULT_ADDR)
  -t, --token string      Vault token (or set VAULT_TOKEN)
  -f, --file string       Output backup file (default "vault-backup.json")
  -e, --engines strings   Specific secret engines to backup (empty = all)
```

### vault-migrator restore

```bash
vault-migrator restore [flags]

Flags:
  -a, --address string         Vault server address (or set VAULT_ADDR)
  -t, --token string           Vault token (or set VAULT_TOKEN)
  -f, --file string            Input backup file (default "vault-backup.json")
  -e, --engines strings        Specific secret engines to restore (empty = all)
  -p, --default-password string Default password for restored users (default "ChangeMe123!")
      --skip-policies          Skip restoring policies
      --skip-auth              Skip restoring auth methods
```

### update-passwords

```bash
update-passwords <vault-address> <vault-token> <users-file>

Arguments:
  vault-address    Vault server URL (e.g., https://vault.example.com:8200)
  vault-token      Vault authentication token
  users-file       JSON file with username:password pairs

Example:
  update-passwords https://vault.example.com:8200 hvs.xxx user.json
```

## Migration Workflow

### Complete Migration Process

1. **Backup from old Vault**:
   ```bash
   export VAULT_ADDR=https://old-vault.example.com
   export VAULT_TOKEN=old-token
   ./vault-migrator backup -f vault-backup.json
   ```

2. **Verify backup file**:
   ```bash
   # Check the backup file was created and contains data
   ls -lh vault-backup.json
   ```

3. **Restore to new Vault**:
   ```bash
   export VAULT_ADDR=https://new-vault.example.com
   export VAULT_TOKEN=new-token
   ./vault-migrator restore -f vault-backup.json --default-password "TempPass2024!"
   ```

4. **Update user passwords** (optional):
   ```bash
   ./update-passwords https://new-vault.example.com new-token user.json
   ```

5. **Verify migration**:
   - Test reading secrets from new Vault
   - Verify policies are in place
   - Test authentication methods
   - Confirm users can log in

## What Gets Backed Up

### Secret Engines
- KV v1 and v2 engines
- All secrets with complete paths
- All versions of each secret (for KV v2)
- Secret metadata (max versions, CAS settings, custom metadata)
- Engine configurations

### Policies
- All custom policies (excludes root and default)
- Complete policy rules

### Auth Methods
- userpass: all users with their configurations (policies, token settings)
- approle: all roles with their configurations
- LDAP: all user mappings
- Auth method configurations

**Note**: User passwords cannot be exported from Vault for security reasons. Users are created with a default password during restore, which can be updated using the `update-passwords` tool.

## Security Notes

- Backup files contain sensitive data - protect them with appropriate permissions (0600)
- Use strong tokens with appropriate permissions
- Store backup files securely (encrypted storage recommended)
- The tool requires root or admin-level tokens to access all data
- User passwords are set to a default value during restore - update them immediately
- Keep the `user.json` file secure as it contains plaintext passwords

## Requirements

- Go 1.21 or higher (for building from source)
- Vault tokens with sufficient permissions:
  - Read access to all secret engines
  - List and read policies
  - List and read auth methods
  - For restore: write access to create mounts, policies, and auth methods

## Troubleshooting

**"Permission denied" errors**: Ensure your token has sufficient permissions

**"Mount already exists" warnings**: The tool will skip creating existing mounts and restore data to them

**Missing secrets after restore**: Check that the secret engine paths match between old and new Vault

**Version mismatches**: The tool handles both KV v1 and v2, but ensure your new Vault supports the same versions

**User password errors**: This is expected - passwords cannot be exported. Use the `update-passwords` tool after migration

**Duplicate users**: The `update-passwords` tool updates ALL occurrences of a username across all auth methods

## License

MIT License - See LICENSE file for details


