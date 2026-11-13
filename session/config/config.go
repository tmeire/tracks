package config

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/tmeire/tracks/database"
	"github.com/tmeire/tracks/session"
	sessiondb "github.com/tmeire/tracks/session/db"
	"github.com/tmeire/tracks/session/inmemory"
	"net/http"
	"time"
)

type Config struct {
	Store struct {
		Type   string          `json:"type"`
		Config json.RawMessage `json:"config"`
	} `json:"store"`
	// AbsoluteTimeout is the maximum age of a session as a Go duration string (e.g. "720h").
	// Empty means the default (30 days); "0" disables the limit.
	AbsoluteTimeout string `json:"absolute_timeout,omitempty"`
	// IdleTimeout is the maximum time between two requests of a session as a Go duration string
	// (e.g. "168h"). Empty means the default (7 days); "0" disables the limit.
	IdleTimeout string `json:"idle_timeout,omitempty"`
}

func parseTimeout(name, v string) (time.Duration, error) {
	if v == "" {
		return 0, nil // default
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid session %s %q: %w", name, v, err)
	}
	if d <= 0 {
		return -1, nil // disabled
	}
	return d, nil
}

// Lifetime returns the configured session lifetime.
func (c Config) Lifetime() (session.Lifetime, error) {
	abs, err := parseTimeout("absolute_timeout", c.AbsoluteTimeout)
	if err != nil {
		return session.Lifetime{}, err
	}
	idle, err := parseTimeout("idle_timeout", c.IdleTimeout)
	if err != nil {
		return session.Lifetime{}, err
	}
	return session.Lifetime{Absolute: abs, Idle: idle}, nil
}

func (c Config) Middleware(ctx context.Context, domain string, db database.Database) (func(handler http.Handler) (http.Handler, error), error) {
	store, err := c.store(ctx, db)
	if err != nil {
		return nil, err
	}

	lifetime, err := c.Lifetime()
	if err != nil {
		return nil, err
	}
	if ls, ok := store.(session.LifetimeSetter); ok {
		ls.SetLifetime(lifetime)
	}

	return session.Middleware(domain, store), nil
}

func (c Config) store(ctx context.Context, db database.Database) (session.Store, error) {
	switch c.Store.Type {
	case "db":
		var dbConfig sessiondb.Config
		err := json.Unmarshal(c.Store.Config, &dbConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal session db config: %w", err)
		}

		if db != nil {
			return dbConfig.CreateWithDB(ctx, db)
		}
		return dbConfig.Create(ctx)
	case "inmemory":
		return inmemory.NewStore(), nil
	default:
		return nil, fmt.Errorf("unsupported session store type: %s", c.Store.Type)
	}
}
