-- +goose Up
CREATE TABLE IF NOT EXISTS feature_flags (
    key TEXT PRIMARY KEY,
    description TEXT,
    default_value BOOLEAN NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS feature_flag_overrides (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    flag_key TEXT NOT NULL,
    principal_type TEXT NOT NULL CHECK (principal_type IN ('global','tenant','role','user')),
    principal_id TEXT NULL,
    value BOOLEAN NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(flag_key) REFERENCES feature_flags(key)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_feature_flag_overrides_unique
ON feature_flag_overrides(flag_key, principal_type, IFNULL(principal_id, ''));

CREATE INDEX IF NOT EXISTS idx_feature_flag_overrides_principal
ON feature_flag_overrides(principal_type, IFNULL(principal_id, ''));

-- +goose Down
DROP INDEX IF EXISTS idx_feature_flag_overrides_principal;
DROP INDEX IF EXISTS idx_feature_flag_overrides_unique;
DROP TABLE IF EXISTS feature_flag_overrides;
DROP TABLE IF EXISTS feature_flags;
