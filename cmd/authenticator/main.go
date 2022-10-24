// Simple HTTPS server using basic authentication.
//
// Eli Bendersky [https://eli.thegreenplace.net]
// This code is in the public domain.

package main

import (
	"flag"
	"log"

	"github.com/BurntSushi/toml"

	"gowallet/internal/apiserver"
)


var (
	configPath = flag.String("configPath", "/home/ivantcov_oa/CRYPTO/GOLANG_WALLET/configs/apiauthserver.toml", "path to config")
)

	// TODO: stub variables 
var	(
	username = "Fedya D"
	password = "dsfaw4wjf3otg3o2eg"
)

func main() {
	config := apiserver.NewAuthConfig()
	_, err := toml.DecodeFile(*configPath, &config)
	if err != nil {
		log.Fatalln(err)
	}

	s := apiserver.NewApiAuthServer(config)
	s.Start()
}