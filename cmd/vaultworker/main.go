// Functions to encrypt and decrypt private keys via Vault
package main

import (
	"flag"
	"log"

	"gowallet/internal/apiserver"

	"github.com/BurntSushi/toml"
)

var (
	configPath = flag.String(
		"configPath",
		"/home/ivantcov_oa/CRYPTO/GOLANG_WALLET/configs/apivaultserver.toml",
		"path to config file")
)

// entry point
func main() {
	flag.Parse()

	config := apiserver.NewVaultConfig()
	_, err := toml.DecodeFile(*configPath, &config)
	if err != nil {
		log.Fatalln(err)
	}

	s, err := apiserver.NewVaultServer(config)
	if err != nil {
		log.Fatalln(err)
	}
	
	s.Start()
}
