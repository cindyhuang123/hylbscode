-- +goose Up
-- +goose StatementBegin
-- Direct UPDATE on an external-content FTS5 table inserts new index entries
-- without removing stale ones (DELETE needs the pre-update value, which is
-- already overwritten when the AFTER UPDATE trigger fires). Switch to the
-- documented single-row delete+insert pair and rebuild once to purge stale
-- entries accumulated by the previous trigger.
DROP TRIGGER IF EXISTS messages_fts_update;
CREATE TRIGGER IF NOT EXISTS messages_fts_update AFTER UPDATE ON messages BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, parts) VALUES ('delete', old.rowid, old.parts);
    INSERT INTO messages_fts(rowid, parts) VALUES (new.rowid, new.parts);
END;
INSERT INTO messages_fts(messages_fts) VALUES ('rebuild');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS messages_fts_update;
CREATE TRIGGER IF NOT EXISTS messages_fts_update AFTER UPDATE ON messages BEGIN
    UPDATE messages_fts SET parts = new.parts WHERE rowid = new.rowid;
END;
-- +goose StatementEnd
