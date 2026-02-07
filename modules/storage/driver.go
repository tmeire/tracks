package storage

import (
	"context"
	"io"
	"time"
)

// Driver is the interface that every storage backend must implement
type Driver interface {
	// Put streams data to the storage backend
	Put(ctx context.Context, key string, r io.Reader) error
	// Get retrieves a reader for the file
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes the file from the backend
	Delete(ctx context.Context, key string) error
	// URL generates a public or signed URL for downloading the file
	URL(ctx context.Context, key string, expires time.Duration) (string, error)
	// SignUpload generates a signed URL for direct client-side upload (e.g., via PUT)
	SignUpload(ctx context.Context, key string, expires time.Duration, contentType string) (string, error)
}
