package tracks

import (
	"encoding/json"
	"github.com/tmeire/tracks/session"
	"os"
	"path/filepath"
)

type Config struct {
	Port       int            `json:"port"`
	BaseDomain string         `json:"base_domain"`
	Sessions   session.Config `json:"sessions"`
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
