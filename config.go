package tracks

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/tmeire/tracks/database"
	"github.com/tmeire/tracks/session/config"
)

type Config struct {
	Name        string                     `json:"name"`
	Version     string                     `json:"version"`
	Port        int                        `json:"port"`
	Development bool                       `json:"development"`
	BaseDomain  string                     `json:"base_domain"`
	Domains     []string                   `json:"domains"`
	Sessions    config.Config              `json:"sessions"`
	Database    database.Config            `json:"database"`
	Cache       CacheConfig                `json:"cache"`
	Modules     map[string]json.RawMessage `json:"modules"`
}

type CacheConfig struct {
	Driver string `json:"driver"`
}

func configFileName() (string, error) {
	if f := os.Getenv("TRACKS_CONFIG_FILE"); f != "" {
		return f, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(cwd, "config", "config.json"), nil
}

func loadConfig() (config Config, err error) {
	fn, err := configFileName()
	if err != nil {
		return
	}

	conf, err := os.Open(fn)
	if err != nil {
		return
	}
	err = json.NewDecoder(conf).Decode(&config)

	return
}
