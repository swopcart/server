package config

import (
	"os"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Server Server `toml:"server"`
}

type Server struct {
	Network string `toml:"network"`
	Address string `toml:"address"`
}

func DefaultConfig() Config {
	return Config{
		Server: Server{
			Network: "tcp",
			Address: "0.0.0.0:8000",
		},
	}
}

func LoadConfig(path string) (cfg Config, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	cfg = DefaultConfig()

	decoder := toml.NewDecoder(file)
	decoder.Decode(&cfg)

	return
}
