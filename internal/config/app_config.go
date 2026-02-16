package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type AppConfig struct {
	Format  string `toml:"format"`
	Profile string `toml:"profile"`
}

func ConfigPath(configDir string) string {
	return filepath.Join(configDir, "config.toml")
}

func LoadAppConfig(configDir string) (AppConfig, error) {
	path := ConfigPath(configDir)

	var cfg AppConfig
	_, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		// If the file doesn't exist, treat as empty config.
		var pErr *os.PathError
		if errors.As(err, &pErr) && errors.Is(pErr.Err, os.ErrNotExist) {
			return AppConfig{}, nil
		}
		if errors.Is(err, os.ErrNotExist) {
			return AppConfig{}, nil
		}
		return AppConfig{}, err
	}

	return cfg, nil
}

