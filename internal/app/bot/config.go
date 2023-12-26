package bot

type Config struct {
	Phones  []string `toml:"phones"`
	AppID   int      `toml:"app_id"`
	AppHash string   `toml:"app_hash"`
}
