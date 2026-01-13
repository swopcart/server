package config

import (
	"log/slog"
	"os"
	"path/filepath"
)

func DataDir() (p string) {
	logger := slog.With("group", "config.DataDir")

	p, ok := os.LookupEnv("SWOPCART_DATA")
	if ok {
		logger.Debug("using env var")
		return
	}

	exePath, err := os.Executable()
	if err == nil {
		p = filepath.Join(exePath, "data")
		_, err := os.Stat(p)
		if err == nil {
			logger.Debug("relative to executable")
			return
		}

		slog.Debug("tried relative to executable, but errored", "err", err)
	}

	slog.Debug("can't get executable. using relative to cwd", "err", err)
	return "data"
}
