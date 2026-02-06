package gcs

import (
	"context"
	"io"
	"time"

	"cloud.google.com/go/storage"
	tstorage "github.com/tmeire/tracks/storage"
)

type Driver struct {
	client *storage.Client
	bucket string
}

var _ tstorage.Driver = (*Driver)(nil)

func NewDriver(client *storage.Client, bucket string) *Driver {
	return &Driver{
		client: client,
		bucket: bucket,
	}
}

func (d *Driver) Put(ctx context.Context, key string, r io.Reader) error {
	w := d.client.Bucket(d.bucket).Object(key).NewWriter(ctx)
	if _, err := io.Copy(w, r); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}

func (d *Driver) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return d.client.Bucket(d.bucket).Object(key).NewReader(ctx)
}

func (d *Driver) Delete(ctx context.Context, key string) error {
	return d.client.Bucket(d.bucket).Object(key).Delete(ctx)
}

func (d *Driver) URL(ctx context.Context, key string, expires time.Duration) (string, error) {
	return d.client.Bucket(d.bucket).SignedURL(key, &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(expires),
	})
}

func (d *Driver) SignUpload(ctx context.Context, key string, expires time.Duration, contentType string) (string, error) {
	return d.client.Bucket(d.bucket).SignedURL(key, &storage.SignedURLOptions{
		Scheme:      storage.SigningSchemeV4,
		Method:      "PUT",
		ContentType: contentType,
		Expires:     time.Now().Add(expires),
	})
}
