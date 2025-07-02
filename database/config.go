package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tmeire/tracks/database/sqlite"
	"log"
)

type Config struct {
	Type   string `json:"type"`
	Config json.RawMessage
}

func (c Config) Create(ctx context.Context) (Database, error) {
	db, err := c.create()
	if err != nil {
		return nil, err
	}

	err = MigrateUp(ctx, db, CentralDatabase)
	if err != nil {
		log.Fatalf("failed to apply migrations: %v", err)
	}

	return db, nil
}

func (c Config) create() (Database, error) {
	switch c.Type {
	case "sqlite":
		var conf sqlite.Config
		err := json.Unmarshal(c.Config, &conf)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal sqlite config: %w", err)
		}
		return conf.Create()
	default:
		return nil, errors.New("unsupported database type")
	}
}
