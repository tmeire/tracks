package config

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/tmeire/tracks/session"
	sessiondb "github.com/tmeire/tracks/session/db"
	"github.com/tmeire/tracks/session/inmemory"
	"net/http"
)

type Config struct {
	Store struct {
		Type   string          `json:"type"`
		Config json.RawMessage `json:"config"`
	} `json:"store"`
}

func (c Config) Middleware(ctx context.Context, domain string) (func(handler http.Handler) (http.Handler, error), error) {
	store, err := c.store(ctx)
	if err != nil {
		return nil, err
	}

	return session.Middleware(domain, store), nil
}

func (c Config) store(ctx context.Context) (session.Store, error) {
	switch c.Store.Type {
	case "db":
		var dbConfig sessiondb.Config
		err := json.Unmarshal(c.Store.Config, &dbConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal session db config: %w", err)
		}

		return dbConfig.Create(ctx)
	case "inmemory":
		return inmemory.NewStore(), nil
	default:
		return nil, fmt.Errorf("unsupported session store type: %s", c.Store.Type)
	}
}
