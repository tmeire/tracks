package storage

import (
	"context"
	"crypto/md5"
	"embed"
	"encoding/base64"
	"fmt"
	"hash"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/tmeire/tracks/database"
)

//go:embed migrations
var migrations embed.FS

// TenantIDExtractor is a function that extracts the tenant ID from the context.
// It returns an empty string if no tenant is found or if multi-tenancy is not used.
type TenantIDExtractor func(context.Context) (string, error)

type StorageService struct {
	driver            Driver
	db                database.Database
	repo              *database.Repository[*StorageService, *Blob]
	tenantIDExtractor TenantIDExtractor
}

// NewStorageService creates a new storage service.
// extractor can be nil if multi-tenancy is not required.
func NewStorageService(ctx context.Context, db database.Database, driver Driver, extractor TenantIDExtractor) (*StorageService, error) {
	// Apply migrations
	err := database.MigrateUpFS(ctx, db, database.CentralDatabase, migrations)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate storage database: %w", err)
	}

	s := &StorageService{
		driver:            driver,
		db:                db,
		tenantIDExtractor: extractor,
	}
	s.repo = database.NewRepository[*StorageService, *Blob](s)
	return s, nil
}

type trackingReader struct {
	r    io.Reader
	n    int64
	hash hash.Hash
}

func (t *trackingReader) Read(p []byte) (int, error) {
	n, err := t.r.Read(p)
	if n > 0 {
		t.n += int64(n)
		t.hash.Write(p[:n])
	}
	return n, err
}

func (s *StorageService) getStorageKey(ctx context.Context, key string) string {
	if s.tenantIDExtractor != nil {
		tenantID, err := s.tenantIDExtractor(ctx)
		if err == nil && tenantID != "" {
			return fmt.Sprintf("tenants/%s/blobs/%s", tenantID, key)
		}
	}
	return key
}

// Attach stores the file and creates a blob record.
func (s *StorageService) Attach(ctx context.Context, r io.Reader, filename string, contentType string) (*Blob, error) {
	key := uuid.New().String()
	storageKey := s.getStorageKey(ctx, key)

	// Create a reader that calculates checksum and size while reading
	tracker := &trackingReader{
		r:    r,
		hash: md5.New(),
	}

	// Stream to driver
	if err := s.driver.Put(ctx, storageKey, tracker); err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	checksum := base64.StdEncoding.EncodeToString(tracker.hash.Sum(nil))

	blob := &Blob{
		Key:         key,
		Filename:    filename,
		ContentType: contentType,
		ByteSize:    tracker.n,
		Checksum:    checksum,
		Status:      "active",
		CreatedAt:   time.Now(),
	}

	// Ensure we use the database associated with this service (Central DB)
	// even if the context has a Tenant DB attached.
	dbCtx := database.WithDB(ctx, s.db)
	createdBlob, err := s.repo.Create(dbCtx, blob)
	if err != nil {
		// Try to cleanup the file if DB insert fails
		_ = s.driver.Delete(ctx, storageKey)
		return nil, fmt.Errorf("failed to create blob record: %w", err)
	}

	return createdBlob, nil
}

// Get retrieves a reader for the file content of a blob
func (s *StorageService) Get(ctx context.Context, blob *Blob) (io.ReadCloser, error) {
	return s.driver.Get(ctx, s.getStorageKey(ctx, blob.Key))
}

// URL generates a signed URL for the blob
func (s *StorageService) URL(ctx context.Context, blob *Blob, expires time.Duration) (string, error) {
	return s.driver.URL(ctx, s.getStorageKey(ctx, blob.Key), expires)
}