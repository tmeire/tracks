# Proposal: `tracks/storage` (Active Storage for Go)

## Overview
This document proposes a new storage module for the `tracks` framework, inspired by Rails' Active Storage. The goal is to provide a unified, driver-based API for handling file uploads, metadata tracking, and multi-tenant isolation across local and cloud environments.

## 1. Core Objectives
- **Driver Abstraction:** Support Local Disk, Google Cloud Storage (GCS), and Amazon S3 with a single API.
- **Database Tracking:** Record file metadata (filename, size, content type, checksum) in a `tracks_blobs` table.
- **Multi-tenancy:** Enforce data isolation by prefixing storage keys with the `tenant_id`.
- **Signed URLs:** Support secure, time-limited access to private files.

## 2. Technical Architecture

### A. The Storage Driver Interface
Every backend must implement the `Driver` interface to ensure hot-swappability.

```go
package storage

import (
	"context"
	"io"
	"time"
)

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
```

### B. Database Schema (Central or Tenant)
To maintain framework consistency, a migration should be added to `tracks` to create the following table:

```sql
CREATE TABLE tracks_blobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,          -- UUID or hash
    filename TEXT NOT NULL,            -- Original name (e.g. "invoice.pdf")
    content_type TEXT,                 -- e.g. "application/pdf"
    byte_size INTEGER NOT NULL,
    checksum TEXT NOT NULL,            -- Base64 MD5/SHA
    status TEXT DEFAULT 'active',      -- 'pending', 'active'
    created_at DATETIME NOT NULL
);
```

## 3. Implementation Workflow

### Phase 1: Storage Service
Implement a `StorageService` that coordinates between the `Driver` and the `database`.

```go
type StorageService struct {
    driver Driver
    db     *database.DB
}

func (s *StorageService) Attach(ctx context.Context, r io.Reader, filename string) (*Blob, error) {
    // 1. Calculate checksum and size while reading
    // 2. Generate a unique key (UUID)
    // 3. Call driver.Put(...)
    // 4. Create record in tracks_blobs
    // 5. Return Blob model
}

// SignUpload prepares the database for an external upload and returns a signed URL
func (s *StorageService) SignUpload(ctx context.Context, filename string, contentType string) (string, *Blob, error) {
    // 1. Generate unique key
    // 2. Create 'pending' blob record in tracks_blobs
    // 3. Return signed URL from driver.SignUpload
}
```

### Phase 2: Drivers
- **`DiskDriver`**: Stores files in `data/storage/{tenant_id}/{key}`.
- **`GCSDriver`**: Uses `cloud.google.com/go/storage` to store in a bucket with tenant prefixes.
- **`S3Driver`**: Uses `aws-sdk-go-v2`.

### Phase 3: Developer API (Usage)
In a `tracks` controller:

```go
func (c *MyController) Create(r *http.Request) (any, error) {
    file, header, _ := r.FormFile("document")
    
    // Attach uses the configured driver automatically
    blob, err := tracks.Storage.Attach(r.Context(), file, header.Filename)
    
    // Store the ID in your domain model
    item := &models.Product{
        ImageBlobID: sql.NullInt64{Int64: blob.ID, Valid: true},
    }
    // ...
}
```

## 4. Multi-tenant Isolation
To prevent cross-tenant access, the `key` should be generated or prefixed based on the tenant context:
- `tenants/{tenant_uuid}/blobs/{blob_uuid}`

## 5. Benefits
1. **Security:** Files are stored using opaque keys, preventing path traversal attacks.
2. **Efficiency:** Metadata is available locally without hitting cloud APIs.
3. **Flexibility:** Projects can start with `DiskDriver` and move to `GCSDriver` in production with zero code changes.
4. **Auditability:** Checksums ensure file integrity over time.
