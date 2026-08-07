-- +goose Up
-- +goose StatementBegin
DROP TRIGGER IF EXISTS messages_fts_delete;
CREATE TRIGGER IF NOT EXISTS messages_fts_delete AFTER DELETE ON messages BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, parts) VALUES ('delete', old.rowid, old.parts);
END;
INSERT INTO messages_fts(messages_fts) VALUES ('rebuild');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS messages_fts_delete;
CREATE TRIGGER IF NOT EXISTS messages_fts_delete AFTER DELETE ON messages BEGIN
    INSERT INTO messages_fts(messages_fts) VALUES ('rebuild');
END;
-- +goose StatementEnd
