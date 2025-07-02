package tracks

import (
	"encoding/json"
	"github.com/tmeire/tracks/database"
	"github.com/tmeire/tracks/session/config"
	"os"
	"path/filepath"
)

type Config struct {
	Port       int             `json:"port"`
	BaseDomain string          `json:"base_domain"`
	Sessions   config.Config   `json:"sessions"`
	Database   database.Config `json:"database"`
}

func loadConfig() (config Config, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	conf, err := os.Open(filepath.Join(cwd, "config", "config.json"))
	if err != nil {
		return
	}
	err = json.NewDecoder(conf).Decode(&config)

	return
}
