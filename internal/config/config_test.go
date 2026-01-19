package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pelletier/go-toml/v2"
	"github.com/swopcart/server/internal/config"
)

func TestLoadConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")

	expected := config.Config{
		Server: config.Server{
			Network: "tcp",
			Address: "127.0.0.1:420",
		},
		Database: config.Database{
			Conn: "postgres://dbhost/potato",
		},
		Auth: config.Auth{
			KeyPath:         "jwt.key",
			AccessTokenTTL:  3000,
			RefreshTokenTTL: 365,
			JWTIssuer:       "foobar",
		},
	}

	err := os.WriteFile(configPath, []byte(`
[server]
network = "tcp"
address = "127.0.0.1:420"

[database]
conn = "postgres://dbhost/potato"

[auth]
key-path = "jwt.key"
access-token-ttl = 3000
refresh-token-ttl = 365
jwt-issuer = "foobar"
`), 0o777)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	actual, err := config.LoadConfig(configPath)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Error(diff)
	}
}

func TestLoadConfigParseError(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")

	err := os.WriteFile(configPath, []byte(`
[server]
network = "tcp
address = "127.0.0.1:420"

[database]
conn = "postgres://dbhost/potato"
`), 0o777)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	_, err = config.LoadConfig(configPath)
	if err == nil {
		t.Log("err is nil")
		t.FailNow()
	}

	var decodeErr *toml.DecodeError
	if !errors.As(err, &decodeErr) {
		t.Log("err is not a DecodeError")
		t.FailNow()
	}
}

func TestLoadConfigStructureError(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")

	err := os.WriteFile(configPath, []byte(`
[server]
port = 420
timeout = 69

[database]
hostname = "db"
username = "swopcart"
password = "swopcart"

[foobar]
bottles = 99
`), 0o777)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	_, err = config.LoadConfig(configPath)
	if err == nil {
		t.Log("err is nil")
		t.FailNow()
	}

	var missingErr *toml.StrictMissingError
	if !errors.As(err, &missingErr) {
		t.Log("err is not a DecodeError")
		t.FailNow()
	}
}
