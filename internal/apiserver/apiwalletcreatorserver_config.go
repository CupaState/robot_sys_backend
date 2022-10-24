package apiserver

// WallerCreatorConfig...
type WallerCreatorConfig struct {
	BindAddr int `toml:"bind_addr"`
	VaultPort int `toml:"vault_port"`
	LogLevel string `toml:"log_level"`
}

// NewWalletCreatorConfig...
func NewWalletCreatorConfig() *WallerCreatorConfig {
	return &WallerCreatorConfig{}
}
