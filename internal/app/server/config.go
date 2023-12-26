package server

import "github.com/Kvertinum01/userbot-api/internal/app/bot"

type Config struct {
	Addr string      `toml:"addr"`
	Bot  *bot.Config `toml:"bot"`
}
