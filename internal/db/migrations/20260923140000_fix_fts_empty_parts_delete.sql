-- +goose Up
-- +goose StatementBegin
-- FTS5 external-content 表的 'delete' 命令要求索引中已存在与 (rowid, parts)
-- 完全匹配的条目, 否则报 "database disk image is malformed" (SQLITE_CORRUPT).
-- 根因: update_messages_updated_at 嵌套触发器对同一行再次 UPDATE
-- (old.parts == new.parts, 仅 updated_at 变化), 仍会触发 messages_fts_update
-- 的 'delete' 命令, 而该行在 FTS 索引中的条目与 old.parts 已不同步 -> 损坏.
-- 修复: 两个触发器都加 WHEN old.parts IS NOT new.parts, 仅在 parts 真正
-- 变化时操作 FTS 索引; delete 额外要求 old.parts 长度 > 2 (trigram 对
-- '[]' 等短文本不产生索引条目, delete 会找不到匹配).
DROP TRIGGER IF EXISTS messages_fts_update;
CREATE TRIGGER IF NOT EXISTS messages_fts_update
AFTER UPDATE ON messages
WHEN length(old.parts) > 2 AND old.parts IS NOT new.parts
BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, parts) VALUES ('delete', old.rowid, old.parts);
END;
DROP TRIGGER IF EXISTS messages_fts_update_reindex;
CREATE TRIGGER IF NOT EXISTS messages_fts_update_reindex
AFTER UPDATE ON messages
WHEN old.parts IS NOT new.parts
BEGIN
    INSERT INTO messages_fts(rowid, parts) VALUES (new.rowid, new.parts);
END;
-- 仅在 messages 已有数据时才 rebuild (空表上 rebuild 会破坏后续增量索引);
-- 全新安装走完此迁移时表为空, 触发器增量插入不受影响.
INSERT INTO messages_fts(messages_fts) SELECT 'rebuild' WHERE EXISTS (SELECT 1 FROM messages);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS messages_fts_update_reindex;
DROP TRIGGER IF EXISTS messages_fts_update;
CREATE TRIGGER IF NOT EXISTS messages_fts_update AFTER UPDATE ON messages BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, parts) VALUES ('delete', old.rowid, old.parts);
    INSERT INTO messages_fts(rowid, parts) VALUES (new.rowid, new.parts);
END;
INSERT INTO messages_fts(messages_fts) VALUES ('rebuild');
-- +goose StatementEnd