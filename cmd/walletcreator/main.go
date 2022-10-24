package main

import (
	"flag"
	"log"

	"github.com/BurntSushi/toml"

	"gowallet/internal/apiserver"
)

var VAULTADDR = flag.String("addr", "localhost:40000", "the address to connect to")
var PRIVATE_KEY string // private key of wallet
var WALLET_ADDR string // internal wallet

var (
	configPath = flag.String(
		"configPath", 
		"/home/ivantcov_oa/CRYPTO/GOLANG_WALLET/configs/apiwalletcreatorserver.toml", 
		"path to config",
	)
)

func main() {
	flag.Parse()

	config := apiserver.NewWalletCreatorConfig()

	_, err := toml.DecodeFile(*configPath, &config)
	if err != nil {
		log.Fatalln(err)
	}

	// Initializing...
	s := apiserver.NewApiWalletCreatorServer(config)
	s.Start()
}
