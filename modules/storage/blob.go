package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/tmeire/tracks/database"
)

type Blob struct {
	ID          int64
	Key         string
	Filename    string
	ContentType string
	ByteSize    int64
	Checksum    string
	Status      string
	CreatedAt   time.Time
}

// Ensure Blob implements database.Model
var _ database.Model[*StorageService, *Blob] = (*Blob)(nil)

func (b *Blob) TableName() string {
	return "tracks_blobs"
}

func (b *Blob) Fields() []string {
	// Must match the order in Values() and Scan() (after ID)
	return []string{"key", "filename", "content_type", "byte_size", "checksum", "status", "created_at"}
}

func (b *Blob) Values() []any {
	return []any{b.Key, b.Filename, b.ContentType, b.ByteSize, b.Checksum, b.Status, b.CreatedAt}
}

func (b *Blob) Scan(ctx context.Context, _ *StorageService, row database.Scanner) (*Blob, error) {
	var blob Blob
	var contentType sql.NullString
	// ID is prepended in SELECT query by Repository
	err := row.Scan(&blob.ID, &blob.Key, &blob.Filename, &contentType, &blob.ByteSize, &blob.Checksum, &blob.Status, &blob.CreatedAt)
	if err != nil {
		return nil, err
	}
	blob.ContentType = contentType.String
	return &blob, nil
}

func (b *Blob) HasAutoIncrementID() bool {
	return true
}

func (b *Blob) GetID() any {
	return b.ID
}
