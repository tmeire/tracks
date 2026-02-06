-- +goose Up
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

-- +goose Down
DROP TABLE tracks_blobs;
