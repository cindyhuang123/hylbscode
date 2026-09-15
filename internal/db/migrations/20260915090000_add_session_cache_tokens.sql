-- +goose Up
-- +goose StatementBegin
ALTER TABLE sessions
ADD COLUMN cache_tokens INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE sessions
DROP COLUMN cache_tokens;
-- +goose StatementEnd