-- +goose Up
-- The mcp module stores nothing in the database; this placeholder migration
-- exists so Tracks' test harness finds the migrations directory.

-- +goose Down
SELECT 1;
