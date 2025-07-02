package db

import (
	"context"
	"fmt"
	"github.com/tmeire/tracks/database"
)

type Config struct {
	database.Config
}

func (c Config) Create(ctx context.Context) (*Store, error) {
	db, err := c.Config.Create(ctx)
	if err != nil {
		return nil, err
	}

	store, err := NewStore(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to create session store: %w", err)
	}
	return store, nil
}
