# Vault Migrator

A comprehensive tool to backup and restore HashiCorp Vault configurations between servers, including:
- All secrets with complete version history (KV v1 and v2)
- Policies
- Auth methods (userpass, approle, LDAP, etc.)
- Secret engine configurations

## Features

- **Complete Version History**: Backs up and restores all versions of secrets, not just the latest
- **Seamless Migration**: No one will notice the server change - all data is preserved
- **Flexible Filtering**: Backup/restore specific secret engines
- **Multiple Auth Methods**: Supports userpass, approle, LDAP, and more
- **Safe Operations**: Validates before restore, provides detailed progress

## Installation

```bash
cd vault-migrator
go mod download
go build -o vault-migrator
```

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
  --file backup.json

# Restore specific engines only
./vault-migrator restore -f backup.json -e secret -e app-secrets

# Skip policies or auth methods
./vault-migrator restore -f backup.json --skip-policies --skip-auth
```

## Command Reference

### Backup Command

```bash
vault-migrator backup [flags]

Flags:
  -a, --address string    Vault server address (or set VAULT_ADDR)
  -t, --token string      Vault token (or set VAULT_TOKEN)
  -f, --file string       Output backup file (default "vault-backup.json")
  -e, --engines strings   Specific secret engines to backup (empty = all)
```

### Restore Command

```bash
vault-migrator restore [flags]

Flags:
  -a, --address string    Vault server address (or set VAULT_ADDR)
  -t, --token string      Vault token (or set VAULT_TOKEN)
  -f, --file string       Input backup file (default "vault-backup.json")
  -e, --engines strings   Specific secret engines to restore (empty = all)
      --skip-policies     Skip restoring policies
      --skip-auth         Skip restoring auth methods
```

## Migration Workflow

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
   ./vault-migrator restore -f vault-backup.json
   ```

4. **Verify migration**:
   - Test reading secrets from new Vault
   - Verify policies are in place
   - Test authentication methods

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
- userpass: all users with their configurations
- approle: all roles with their configurations
- LDAP: all user mappings
- Auth method configurations

## Security Notes

- Backup files contain sensitive data - protect them with appropriate permissions (0600)
- Use strong tokens with appropriate permissions
- Store backup files securely (encrypted storage recommended)
- The tool requires root or admin-level tokens to access all data

## Requirements

- Go 1.21 or higher
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


