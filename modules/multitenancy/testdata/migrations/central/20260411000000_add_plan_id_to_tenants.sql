-- +goose Up
-- +goose StatementBegin
ALTER TABLE tenants ADD COLUMN plan_id TEXT NOT NULL DEFAULT 'free';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tenants DROP COLUMN plan_id;
-- +goose StatementEnd
