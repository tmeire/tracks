-- +goose Up
-- +goose StatementBegin
CREATE TABLE profiles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    bio TEXT,
    portfolio_url TEXT,
    specialties TEXT,
    is_public BOOLEAN NOT NULL DEFAULT 0,
    availability_status TEXT NOT NULL DEFAULT 'available',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
CREATE INDEX idx_profiles_user_id ON profiles(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE profiles;
-- +goose StatementEnd
