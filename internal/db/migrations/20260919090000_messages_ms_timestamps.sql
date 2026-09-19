-- +goose Up
-- +goose StatementBegin
-- messages 时间戳实际按秒存储 (strftime('%s','now')), 与 schema 注释
-- "Unix timestamp in milliseconds" 不符, 导致 GUI 毫秒解析显示 1970 年.
-- 先删旧触发器再转换历史秒值(否则转换 UPDATE 会被旧触发器改回秒),
-- 最后重建毫秒触发器.
DROP TRIGGER IF EXISTS update_messages_updated_at;

UPDATE messages SET
    created_at = created_at * 1000,
    updated_at = updated_at * 1000,
    finished_at = CASE WHEN finished_at IS NOT NULL THEN finished_at * 1000 ELSE NULL END;

CREATE TRIGGER IF NOT EXISTS update_messages_updated_at
AFTER UPDATE ON messages
BEGIN
UPDATE messages SET updated_at = CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER)
WHERE id = new.id;
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_messages_updated_at;

UPDATE messages SET
    created_at = created_at / 1000,
    updated_at = updated_at / 1000,
    finished_at = CASE WHEN finished_at IS NOT NULL THEN finished_at / 1000 ELSE NULL END;

CREATE TRIGGER IF NOT EXISTS update_messages_updated_at
AFTER UPDATE ON messages
BEGIN
UPDATE messages SET updated_at = strftime('%s', 'now')
WHERE id = new.id;
END;
-- +goose StatementEnd