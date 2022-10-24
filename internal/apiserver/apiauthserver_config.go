package apiserver

// AuthConfig...
type AuthConfig struct {
	BindAddr int `toml:"bind_addr"`
	LogLevel string `toml:"log_level"`
	VaultPort int `toml:"vault_port"`
	PSQLPort int `toml:"psql_port"`
	WalletCreatorPort int `toml:"wallet_creator_port"`
}

// NewAuthConfig...
func NewAuthConfig() *AuthConfig {
	return &AuthConfig{}
}
