package main

import (
	"flag"
	"log"

	"gowallet/internal/apiserver"

	"github.com/BurntSushi/toml"
	_ "github.com/lib/pq"
)

var (
	configPath = flag.String("configPath", "/home/ivantcov_oa/CRYPTO/GOLANG_WALLET/configs/apipsqlserver.toml", "path to config")
)

func main() {
	flag.Parse()

	config := apiserver.NewPSQLConfig()

	_, err := toml.DecodeFile(*configPath, &config)
	if err != nil {
		log.Fatalln(err)
	}

	s := apiserver.NewPSQLServer(config)
	s.Start()
}
