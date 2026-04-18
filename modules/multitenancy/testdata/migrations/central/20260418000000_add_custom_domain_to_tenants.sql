-- +goose Up
-- +goose StatementBegin
ALTER TABLE tenants ADD COLUMN custom_domain TEXT DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tenants DROP COLUMN custom_domain;
-- +goose StatementEnd