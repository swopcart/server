package config

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Server   Server   `toml:"server"`
	Database Database `toml:"database"`
	Auth     Auth     `toml:"auth"`
	Jobs     Jobs     `toml:"jobs"`
}

type Server struct {
	Network string `toml:"network"`
	Address string `toml:"address"`
}

type Database struct {
	Conn string `toml:"conn"` // PostgreSQL connection string (see https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING)
}

type Auth struct {
	KeyPath         string `toml:"key-path"          comment:"path to private key used to sign JWT tokens"`
	AccessTokenTTL  uint   `toml:"access-token-ttl"  comment:"number of seconds an access token should last"`
	RefreshTokenTTL uint   `toml:"refresh-token-ttl" comment:"number of days a refresh token should last"`
	JWTIssuer       string `toml:"jwt-issuer"        comment:"issuer to use in JWT tokens"`
}

type Jobs struct {
	DefaultWorkers  int                    `toml:"default-workers"  comment:"number of workers per queue (defaults to CPU core count)"`
	ShutdownTimeout time.Duration          `toml:"shutdown-timeout" comment:"time to wait for running jobs during shutdown"`
	Queues          map[string]QueueConfig `toml:"queues" comment:"per-queue worker configuration"`
}

type QueueConfig struct {
	Workers int `toml:"workers" comment:"number of workers for this queue"`
}

func DefaultConfig() Config {
	return Config{
		Server: Server{
			Network: "tcp",
			Address: "0.0.0.0:8000",
		},
		Database: Database{
			Conn: "postgres://swopcart:swopcart@localhost:5432/swopcart",
		},
		Auth: Auth{
			KeyPath:         "auth.key",
			AccessTokenTTL:  300,
			RefreshTokenTTL: 90,
			JWTIssuer:       "swopcart",
		},
		Jobs: Jobs{
			DefaultWorkers:  runtime.NumCPU(),
			ShutdownTimeout: 30 * time.Second,
			Queues:          make(map[string]QueueConfig),
		},
	}
}

func LoadConfig(path string) (cfg Config, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	cfg = DefaultConfig()

	decoder := toml.NewDecoder(file).
		DisallowUnknownFields()
	err = decoder.Decode(&cfg)

	return
}

func (a *Auth) AbsoluteKeyPath() string {
	if filepath.IsAbs(a.KeyPath) {
		return a.KeyPath
	}

	return filepath.Join(DataDir(), a.KeyPath)
}
