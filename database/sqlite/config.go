package sqlite

import (
	"database/sql"
)

type Config struct {
	Path string `json:"path"`
}

func (c Config) Create() (*sql.DB, error) {
	return New(c.Path)
}
