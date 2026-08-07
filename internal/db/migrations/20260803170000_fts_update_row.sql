-- +goose Up
-- +goose StatementBegin
-- Streaming updates call messages.Update for every content delta; a full FTS
-- rebuild on each one stalls rendering (2.4s at ~300 messages). Update only
-- the affected row instead: external content tables support per-row UPDATE.
DROP TRIGGER IF EXISTS messages_fts_update;
CREATE TRIGGER IF NOT EXISTS messages_fts_update AFTER UPDATE ON messages BEGIN
    UPDATE messages_fts SET parts = new.parts WHERE rowid = new.rowid;
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS messages_fts_update;
CREATE TRIGGER IF NOT EXISTS messages_fts_update AFTER UPDATE ON messages BEGIN
    INSERT INTO messages_fts(messages_fts) VALUES ('rebuild');
END;
-- +goose StatementEnd
