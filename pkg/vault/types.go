package vault

import "time"

type BackupData struct {
	Timestamp     time.Time             `json:"timestamp"`
	VaultVersion  string                `json:"vault_version"`
	SecretEngines []SecretEngineBackup  `json:"secret_engines"`
	Policies      []PolicyBackup        `json:"policies"`
	AuthMethods   []AuthMethodBackup    `json:"auth_methods"`
}

type SecretEngineBackup struct {
	Path        string                 `json:"path"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	Options     map[string]interface{} `json:"options"`
	Secrets     []SecretBackup         `json:"secrets"`
}

type SecretBackup struct {
	Path     string            `json:"path"`
	Versions []SecretVersion   `json:"versions"`
	Metadata SecretMetadata    `json:"metadata"`
}

type SecretVersion struct {
	Version      int                    `json:"version"`
	Data         map[string]interface{} `json:"data"`
	CreatedTime  time.Time              `json:"created_time"`
	DeletionTime string                 `json:"deletion_time,omitempty"`
	Destroyed    bool                   `json:"destroyed"`
}

type SecretMetadata struct {
	CasRequired      bool              `json:"cas_required"`
	CreatedTime      time.Time         `json:"created_time"`
	CurrentVersion   int               `json:"current_version"`
	MaxVersions      int               `json:"max_versions"`
	OldestVersion    int               `json:"oldest_version"`
	UpdatedTime      time.Time         `json:"updated_time"`
	CustomMetadata   map[string]string `json:"custom_metadata,omitempty"`
	DeleteVersionAfter string          `json:"delete_version_after,omitempty"`
}

type PolicyBackup struct {
	Name   string `json:"name"`
	Policy string `json:"policy"`
}

type AuthMethodBackup struct {
	Path        string                 `json:"path"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	Options     map[string]interface{} `json:"options"`
	Roles       []RoleBackup           `json:"roles,omitempty"`
	Users       []UserBackup           `json:"users,omitempty"`
}

type RoleBackup struct {
	Name string                 `json:"name"`
	Data map[string]interface{} `json:"data"`
}

type UserBackup struct {
	Name string                 `json:"name"`
	Data map[string]interface{} `json:"data"`
}

type RestoreOptions struct {
	Engines         []string
	SkipPolicies    bool
	SkipAuth        bool
	DefaultPassword string
}
