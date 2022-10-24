package apiserver

import "gowallet/internal/store"

// Config ...
type PSQLConfig struct {
	BindAddr int `toml:"bind_addr"` // Address where we run our server
	LogLevel string `toml:"log_level"` // Info, debug e.g.
	Store *store.Config
}

// NewConfig with default data
func NewPSQLConfig() *PSQLConfig{
	return &PSQLConfig{
		BindAddr: 40010,
		LogLevel: "debug",
		Store: store.NewConfig(),
	}
}
