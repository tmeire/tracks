-- +goose Up
-- +goose StatementBegin
ALTER TABLE tenants ADD COLUMN active BOOLEAN NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tenants DROP COLUMN active;
-- +goose StatementEnd
