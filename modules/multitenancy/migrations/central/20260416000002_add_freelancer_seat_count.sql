-- +goose Up
ALTER TABLE tenants ADD COLUMN freelancer_seat_count INTEGER DEFAULT 0;

-- +goose Down
-- SQLite doesn't support DROP COLUMN easily
