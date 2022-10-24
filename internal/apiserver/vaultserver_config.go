package apiserver

// VaultConfig...
type VaultConfig struct {
	BindAddr string `toml:"bind_addr"`
	BindPort int `toml:"bind_port"`
	Token string `toml:"token"`
	SecretName string `toml:"secret_name"`
	PrivateEncryptionKey string `toml:"private_encryption_key"`
	LogLevel string `toml:"log_level"`
}

// NewVaultConfig returns empty VaultConfig object
func NewVaultConfig() *VaultConfig {
	return &VaultConfig{}
}
