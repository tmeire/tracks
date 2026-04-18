-- +goose Up
-- +goose StatementBegin
ALTER TABLE tenants ADD COLUMN staff_seat_count INTEGER DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tenants DROP COLUMN staff_seat_count;
-- +goose StatementEnd
