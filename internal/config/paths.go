package config

import (
	"os"
	"path/filepath"
)

func Dir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "clustertui")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "clustertui")
}

func DefaultPath() string {
	return filepath.Join(Dir(), "config.toml")
}
