package disk

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/tmeire/tracks/storage"
)

// Driver implements storage.Driver for local disk storage
type Driver struct {
	root string
}

// Ensure Driver implements storage.Driver
var _ storage.Driver = (*Driver)(nil)

// NewDriver creates a new disk storage driver
func NewDriver(root string) *Driver {
	return &Driver{root: root}
}

func (d *Driver) Put(ctx context.Context, key string, r io.Reader) error {
	path := filepath.Join(d.root, key)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	return err
}

func (d *Driver) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	path := filepath.Join(d.root, key)
	return os.Open(path)
}

func (d *Driver) Delete(ctx context.Context, key string) error {
	path := filepath.Join(d.root, key)
	return os.Remove(path)
}

func (d *Driver) URL(ctx context.Context, key string, expires time.Duration) (string, error) {
	// For disk storage, we return a path that should be served by the application
	// This assumes the application has a route handler for /storage/*
	return "/storage/" + key, nil
}

func (d *Driver) SignUpload(ctx context.Context, key string, expires time.Duration, contentType string) (string, error) {
	// Disk driver usually doesn't support direct client-side uploads (signed URLs)
	// unless there's a specific endpoint handling it.
	return "", fmt.Errorf("disk driver does not support signed uploads")
}
