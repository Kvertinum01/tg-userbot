package main

import (
	"flag"
	"log"

	"github.com/BurntSushi/toml"
	"github.com/Kvertinum01/userbot-api/internal/app/bot"
	"github.com/Kvertinum01/userbot-api/internal/app/server"
)

var (
	configPath string
)

func init() {
	flag.StringVar(&configPath, "config-path", "configs/config.toml", "path to config file")
}

func main() {
	flag.Parse()

	conf := &server.Config{}
	if _, err := toml.DecodeFile(configPath, conf); err != nil {
		log.Fatal(err)
	}

	sessions := bot.Setup(conf.Bot)

	if err := server.SetupServer(conf, sessions); err != nil {
		log.Fatalln(err)
	}
}
