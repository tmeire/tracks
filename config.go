package tracks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmeire/tracks/database"
	"github.com/tmeire/tracks/i18n"
	"github.com/tmeire/tracks/session/config"
)

type Config struct {
	Name        string                     `json:"name"`
	Version     string                     `json:"version"`
	Port        int                        `json:"port"`
	Development bool                       `json:"development"`
	Secure      bool                       `json:"secure"`
	BaseDomain  string                     `json:"base_domain"`
	Domains     []string                   `json:"domains"`
	Sessions    config.Config              `json:"sessions"`
	Database    database.Config            `json:"database"`
	Cache       CacheConfig                `json:"cache"`
	Jobs        JobsConfig                 `json:"jobs"`
	Secrets     SecretsConfig              `json:"secrets"`
	Modules     map[string]json.RawMessage `json:"modules"`
	// I18n configures the supported locales. When nil, tracks detects the language from the
	// request (query, cookie, session, Accept-Language) with "en" as default.
	I18n *i18n.Config `json:"i18n"`
}

// BaseURL returns the absolute origin of the application, e.g. https://example.com.
func (c Config) BaseURL() string {
	if c.I18n != nil && c.I18n.BaseURL != "" {
		return strings.TrimSuffix(c.I18n.BaseURL, "/")
	}
	if c.BaseDomain == "" {
		return ""
	}
	scheme := "http"
	if c.Secure {
		scheme = "https"
	}
	return scheme + "://" + c.BaseDomain
}

// i18nConfig returns the normalized i18n configuration and whether one was provided.
func (c Config) i18nConfig() (i18n.Config, bool) {
	if c.I18n == nil {
		return i18n.Config{}.Normalize(), false
	}
	cfg := *c.I18n
	cfg.BaseURL = c.BaseURL()
	return cfg.Normalize(), true
}

type SecretsConfig struct {
	Signup string `json:"signup"`
}

type CacheConfig struct {
	Driver string `json:"driver"`
}

type JobsConfig struct {
	Driver  string `json:"driver"`
	Workers int    `json:"workers"`
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
