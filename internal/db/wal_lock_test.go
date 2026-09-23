package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/pressly/goose/v3"
)

// 真实场景: 2 个 goroutine 各持一条 sqlc stmt, 交替 UPDATE(含 FTS5 触发器)
// 观察是否卡死(busy_timeout 10s 内无法完成)
func TestWALLockTimeout(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "t.db")
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=page_size(4096)&_pragma=foreign_keys(ON)&_pragma=synchronous(NORMAL)&_pragma=cache_size(-8000)", url.PathEscape(dbPath))
	sdb, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer sdb.Close()
	sdb.SetMaxOpenConns(1)

	for _, s := range []string{
		`CREATE TABLE messages (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, role TEXT NOT NULL, parts TEXT NOT NULL default '[]', model TEXT, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL, finished_at INTEGER);`,
		`CREATE VIRTUAL TABLE messages_fts USING fts5(parts, content='messages', content_rowid='rowid', tokenize='trigram');`,
		`CREATE TRIGGER messages_fts_insert AFTER INSERT ON messages BEGIN INSERT INTO messages_fts(rowid, parts) VALUES (new.rowid, new.parts); END;`,
		`CREATE TRIGGER messages_fts_update AFTER UPDATE ON messages WHEN length(old.parts) > 2 AND old.parts IS NOT new.parts BEGIN INSERT INTO messages_fts(messages_fts, rowid, parts) VALUES ('delete', old.rowid, old.parts); END;`,
		`CREATE TRIGGER messages_fts_update_reindex AFTER UPDATE ON messages WHEN old.parts IS NOT new.parts BEGIN INSERT INTO messages_fts(rowid, parts) VALUES (new.rowid, new.parts); END;`,
		`CREATE TRIGGER update_messages_updated_at AFTER UPDATE ON messages BEGIN UPDATE messages SET updated_at = CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER) WHERE id = new.id; END;`,
		`INSERT INTO messages (id, session_id, role, parts, created_at, updated_at) VALUES ('m1','s1','assistant','[]',0,0);`,
		`INSERT INTO messages (id, session_id, role, parts, created_at, updated_at) VALUES ('m2','s1','assistant','[]',0,0);`,
	} {
		if _, err := sdb.Exec(s); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	// 关键: 先 Prepare 两条 stmt(模拟 sqlc 缓存), 再并发执行
	stmt1, err := sdb.Prepare("UPDATE messages SET parts=?, updated_at=CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER) WHERE id=?;")
	if err != nil {
		t.Fatal(err)
	}
	defer stmt1.Close()
	stmt2, err := sdb.Prepare("UPDATE messages SET parts=?, updated_at=CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER) WHERE id=?;")
	if err != nil {
		t.Fatal(err)
	}
	defer stmt2.Close()

	// 两条并发 stmt 执行
	type r struct {
		stage string
		err   error
	}
	ch := make(chan r, 2)
	go func() {
		for i := 0; i < 20; i++ {
			if _, err := stmt1.Exec(fmt.Sprintf("[%d 内容]", i), "m1"); err != nil {
				ch <- r{"s1-" + fmt.Sprint(i), err}
				return
			}
		}
		ch <- r{"s1-done", nil}
	}()
	go func() {
		for i := 0; i < 20; i++ {
			if _, err := stmt2.Exec(fmt.Sprintf("[%d 内容]", i), "m2"); err != nil {
				ch <- r{"s2-" + fmt.Sprint(i), err}
				return
			}
		}
		ch <- r{"s2-done", nil}
	}()
	done := 0
	timeout := time.After(20 * time.Second)
	for done < 2 {
		select {
		case res := <-ch:
			t.Logf("result: %v", res)
			if res.err != nil {
				t.Fatalf("exec error: %v", res.err)
			}
			done++
		case <-timeout:
			t.Fatalf("TIMEOUT: only %d/2 finished - 锁死!", done)
		}
	}
	var ok string
	if err := sdb.QueryRow("PRAGMA integrity_check").Scan(&ok); err != nil {
		t.Fatal(err)
	}
	if ok != "ok" {
		t.Fatalf("integrity=%q", ok)
	}
}

// 对照1: 同一 stmt 并发执行(非双 stmt)
func TestWALSameStmtConcurrent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "t2.db")
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=page_size(4096)", url.PathEscape(dbPath))
	sdb, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer sdb.Close()
	sdb.SetMaxOpenConns(1)
	for _, s := range []string{
		`CREATE TABLE messages (id TEXT PRIMARY KEY, parts TEXT NOT NULL default '[]');`,
		`CREATE VIRTUAL TABLE messages_fts USING fts5(parts, content='messages', content_rowid='rowid', tokenize='trigram');`,
		`CREATE TRIGGER messages_fts_update AFTER UPDATE ON messages WHEN length(old.parts) > 2 AND old.parts IS NOT new.parts BEGIN INSERT INTO messages_fts(messages_fts, rowid, parts) VALUES ('delete', old.rowid, old.parts); END;`,
		`CREATE TRIGGER messages_fts_update_reindex AFTER UPDATE ON messages WHEN old.parts IS NOT new.parts BEGIN INSERT INTO messages_fts(rowid, parts) VALUES (new.rowid, new.parts); END;`,
		`INSERT INTO messages (id, parts) VALUES ('m1','[]');`,
		`INSERT INTO messages (id, parts) VALUES ('m2','[]');`,
	} {
		if _, err := sdb.Exec(s); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}
	stmt, err := sdb.Prepare("UPDATE messages SET parts=? WHERE id=?;")
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()
	type r struct {
		stage string
		err   error
	}
	ch := make(chan r, 2)
	go func() {
		for i := 0; i < 20; i++ {
			if _, err := stmt.Exec(fmt.Sprintf("[%d]", i), "m1"); err != nil {
				ch <- r{"g1-" + fmt.Sprint(i), err}
				return
			}
		}
		ch <- r{"g1-done", nil}
	}()
	go func() {
		for i := 0; i < 20; i++ {
			if _, err := stmt.Exec(fmt.Sprintf("[%d]", i), "m2"); err != nil {
				ch <- r{"g2-" + fmt.Sprint(i), err}
				return
			}
		}
		ch <- r{"g2-done", nil}
	}()
	done := 0
	for done < 2 {
		res := <-ch
		t.Logf("same-stmt result: %v", res)
		if res.err != nil {
			t.Fatalf("exec error: %v", res.err)
		}
		done++
	}
}

// 对照2: 无 FTS5 触发器时双 stmt 并发
func TestWALNoFTSSameStmt(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "t3.db")
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=page_size(4096)", url.PathEscape(dbPath))
	sdb, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer sdb.Close()
	sdb.SetMaxOpenConns(1)
	if _, err := sdb.Exec(`CREATE TABLE messages (id TEXT PRIMARY KEY, parts TEXT NOT NULL default '[]');`); err != nil {
		t.Fatal(err)
	}
	if _, err := sdb.Exec(`INSERT INTO messages (id, parts) VALUES ('m1','[]');`); err != nil {
		t.Fatal(err)
	}
	stmt, err := sdb.Prepare("UPDATE messages SET parts=? WHERE id=?;")
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()
	type r struct {
		err error
	}
	ch := make(chan r, 2)
	go func() {
		for i := 0; i < 20; i++ {
			if _, err := stmt.Exec(fmt.Sprintf("[%d]", i), "m1"); err != nil {
				ch <- r{err}
				return
			}
		}
		ch <- r{}
	}()
	go func() {
		for i := 0; i < 20; i++ {
			if _, err := stmt.Exec(fmt.Sprintf("[%d]", i), "m1"); err != nil {
				ch <- r{err}
				return
			}
		}
		ch <- r{}
	}()
	done := 0
	for done < 2 {
		res := <-ch
		if res.err != nil {
			t.Fatalf("no-fts exec error: %v", res.err)
		}
		done++
	}
	t.Log("no-fts OK")
}

// 对照3: prepared stmt + FTS5 但完全顺序执行
func TestWALPreparedSequentialFTS(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "t4.db")
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=page_size(4096)", url.PathEscape(dbPath))
	sdb, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer sdb.Close()
	sdb.SetMaxOpenConns(1)
	for _, s := range []string{
		`CREATE TABLE messages (id TEXT PRIMARY KEY, parts TEXT NOT NULL default '[]');`,
		`CREATE VIRTUAL TABLE messages_fts USING fts5(parts, content='messages', content_rowid='rowid', tokenize='trigram');`,
		`CREATE TRIGGER messages_fts_update AFTER UPDATE ON messages WHEN length(old.parts) > 2 AND old.parts IS NOT new.parts BEGIN INSERT INTO messages_fts(messages_fts, rowid, parts) VALUES ('delete', old.rowid, old.parts); END;`,
		`CREATE TRIGGER messages_fts_update_reindex AFTER UPDATE ON messages WHEN old.parts IS NOT new.parts BEGIN INSERT INTO messages_fts(rowid, parts) VALUES (new.rowid, new.parts); END;`,
		`INSERT INTO messages (id, parts) VALUES ('m1','[]');`,
		`INSERT INTO messages (id, parts) VALUES ('m2','[]');`,
	} {
		if _, err := sdb.Exec(s); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}
	stmt, err := sdb.Prepare("UPDATE messages SET parts=? WHERE id=?;")
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()
	for i := 0; i < 40; i++ {
		if _, err := stmt.Exec(fmt.Sprintf("[%d]", i), "m1"); err != nil {
			t.Fatalf("sequential exec %d err: %v", i, err)
		}
		if _, err := stmt.Exec(fmt.Sprintf("[%d]", i), "m2"); err != nil {
			t.Fatalf("sequential exec %d (m2) err: %v", i, err)
		}
	}
	t.Log("sequential OK")
}

// 对照4: 相同 SQL 与 schema, 用 db.Exec(非 prepared) 顺序执行
func TestWALExecSequentialFTS(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "t5.db")
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=page_size(4096)", url.PathEscape(dbPath))
	sdb, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer sdb.Close()
	sdb.SetMaxOpenConns(1)
	for _, s := range []string{
		`CREATE TABLE messages (id TEXT PRIMARY KEY, parts TEXT NOT NULL default '[]');`,
		`CREATE VIRTUAL TABLE messages_fts USING fts5(parts, content='messages', content_rowid='rowid', tokenize='trigram');`,
		`CREATE TRIGGER messages_fts_update AFTER UPDATE ON messages WHEN length(old.parts) > 2 AND old.parts IS NOT new.parts BEGIN INSERT INTO messages_fts(messages_fts, rowid, parts) VALUES ('delete', old.rowid, old.parts); END;`,
		`CREATE TRIGGER messages_fts_update_reindex AFTER UPDATE ON messages WHEN old.parts IS NOT new.parts BEGIN INSERT INTO messages_fts(rowid, parts) VALUES (new.rowid, new.parts); END;`,
		`INSERT INTO messages (id, parts) VALUES ('m1','[]');`,
		`INSERT INTO messages (id, parts) VALUES ('m2','[]');`,
	} {
		if _, err := sdb.Exec(s); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}
	for i := 0; i < 40; i++ {
		if _, err := sdb.Exec("UPDATE messages SET parts=? WHERE id=?;", fmt.Sprintf("[%d]", i), "m1"); err != nil {
			t.Fatalf("exec %d err: %v", i, err)
		}
		if _, err := sdb.Exec("UPDATE messages SET parts=? WHERE id=?;", fmt.Sprintf("[%d]", i), "m2"); err != nil {
			t.Fatalf("exec %d (m2) err: %v", i, err)
		}
	}
	t.Log("exec sequential OK")
}

func runFTSUpdateLoop(t *testing.T, dsn string, name string) {
	t.Helper()
	sdb, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer sdb.Close()
	sdb.SetMaxOpenConns(1)
	for _, s := range []string{
		`CREATE TABLE messages (id TEXT PRIMARY KEY, parts TEXT NOT NULL default '[]');`,
		`CREATE VIRTUAL TABLE messages_fts USING fts5(parts, content='messages', content_rowid='rowid', tokenize='trigram');`,
		`CREATE TRIGGER messages_fts_update AFTER UPDATE ON messages WHEN length(old.parts) > 2 AND old.parts IS NOT new.parts BEGIN INSERT INTO messages_fts(messages_fts, rowid, parts) VALUES ('delete', old.rowid, old.parts); END;`,
		`CREATE TRIGGER messages_fts_update_reindex AFTER UPDATE ON messages WHEN old.parts IS NOT new.parts BEGIN INSERT INTO messages_fts(rowid, parts) VALUES (new.rowid, new.parts); END;`,
		`INSERT INTO messages (id, parts) VALUES ('m1','[]');`,
		`INSERT INTO messages (id, parts) VALUES ('m2','[]');`,
	} {
		if _, err := sdb.Exec(s); err != nil {
			t.Fatalf("[%s] setup: %v", name, err)
		}
	}
	for i := 0; i < 10; i++ {
		if _, err := sdb.Exec("UPDATE messages SET parts=? WHERE id=?;", fmt.Sprintf("[%d %d]", i, i), "m1"); err != nil {
			t.Fatalf("[%s] exec %d err: %v", name, i, err)
		}
	}
	t.Logf("[%s] OK", name)
}

func TestFTSVariantCacheSize(t *testing.T) {
	dir := t.TempDir()
	variants := map[string]string{
		"with_cache": fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=page_size(4096)&_pragma=cache_size(-8000)", url.PathEscape(filepath.Join(dir, "a.db"))),
		"no_cache":   fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=page_size(4096)", url.PathEscape(filepath.Join(dir, "b.db"))),
	}
	for name, dsn := range variants {
		t.Run(name, func(t *testing.T) {
			runFTSUpdateLoop(t, dsn, name)
		})
	}
}

func runVariant(t *testing.T, ddl string, name string) {
	t.Helper()
	dir := t.TempDir()
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=page_size(4096)", url.PathEscape(filepath.Join(dir, "v.db")))
	sdb, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer sdb.Close()
	sdb.SetMaxOpenConns(1)
	for _, s := range []string{
		ddl,
		`CREATE VIRTUAL TABLE messages_fts USING fts5(parts, content='messages', content_rowid='rowid', tokenize='trigram');`,
		`CREATE TRIGGER messages_fts_update AFTER UPDATE ON messages WHEN length(old.parts) > 2 AND old.parts IS NOT new.parts BEGIN INSERT INTO messages_fts(messages_fts, rowid, parts) VALUES ('delete', old.rowid, old.parts); END;`,
		`CREATE TRIGGER messages_fts_update_reindex AFTER UPDATE ON messages WHEN old.parts IS NOT new.parts BEGIN INSERT INTO messages_fts(rowid, parts) VALUES (new.rowid, new.parts); END;`,
		`INSERT INTO messages (id, parts) VALUES ('m1','[]');`,
	} {
		if _, err := sdb.Exec(s); err != nil {
			t.Fatalf("[%s] setup: %v", name, err)
		}
	}
	for i := 0; i < 5; i++ {
		if _, err := sdb.Exec("UPDATE messages SET parts=? WHERE id=?;", fmt.Sprintf("[%d]", i), "m1"); err != nil {
			t.Fatalf("[%s] exec %d err: %v", name, i, err)
		}
	}
	t.Logf("[%s] OK", name)
}

func TestFTSVariantTrigger(t *testing.T) {
	dir := t.TempDir()
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=page_size(4096)", url.PathEscape(filepath.Join(dir, "v.db")))
	sdb, _ := sql.Open("sqlite3", dsn)
	defer sdb.Close()
	sdb.SetMaxOpenConns(1)
	for _, s := range []string{
		`CREATE TABLE messages (id TEXT PRIMARY KEY, parts TEXT NOT NULL default '[]');`,
		`CREATE VIRTUAL TABLE messages_fts USING fts5(parts, content='messages', content_rowid='rowid', tokenize='trigram');`,
		`CREATE TRIGGER messages_fts_update AFTER UPDATE ON messages BEGIN INSERT INTO messages_fts(messages_fts) VALUES ('rebuild'); END;`,
		`INSERT INTO messages (id, parts) VALUES ('m1','[]');`,
	} {
		if _, err := sdb.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 5; i++ {
		if _, err := sdb.Exec("UPDATE messages SET parts=? WHERE id=?;", fmt.Sprintf("[%d]", i), "m1"); err != nil {
			t.Fatalf("rebuild-trigger exec %d err: %v", i, err)
		}
	}
	t.Log("[rebuild-trigger] OK")
}

func TestWALFullTriggersNcruces(t *testing.T) {
	dir := t.TempDir()
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=page_size(4096)", url.PathEscape(filepath.Join(dir, "v.db")))
	sdb, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer sdb.Close()
	sdb.SetMaxOpenConns(1)
	for _, s := range []string{
		`CREATE TABLE messages (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, role TEXT NOT NULL, parts TEXT NOT NULL default '[]', model TEXT, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL, finished_at INTEGER);`,
		`CREATE VIRTUAL TABLE messages_fts USING fts5(parts, content='messages', content_rowid='rowid', tokenize='trigram');`,
		`CREATE TRIGGER messages_fts_insert AFTER INSERT ON messages BEGIN INSERT INTO messages_fts(rowid, parts) VALUES (new.rowid, new.parts); END;`,
		`CREATE TRIGGER messages_fts_delete AFTER DELETE ON messages BEGIN INSERT INTO messages_fts(messages_fts, rowid, parts) VALUES ('delete', old.rowid, old.parts); END;`,
		`CREATE TRIGGER messages_fts_update AFTER UPDATE ON messages WHEN length(old.parts) > 2 AND old.parts IS NOT new.parts BEGIN INSERT INTO messages_fts(messages_fts, rowid, parts) VALUES ('delete', old.rowid, old.parts); END;`,
		`CREATE TRIGGER messages_fts_update_reindex AFTER UPDATE ON messages WHEN old.parts IS NOT new.parts BEGIN INSERT INTO messages_fts(rowid, parts) VALUES (new.rowid, new.parts); END;`,
		`INSERT INTO messages (id, session_id, role, parts, created_at, updated_at) VALUES ('m1','s1','assistant','[]',1000,1000);`,
		`INSERT INTO messages (id, session_id, role, parts, created_at, updated_at) VALUES ('m2','s1','assistant','[]',1000,1000);`,
	} {
		if _, err := sdb.Exec(s); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}
	for i := 1; i <= 10; i++ {
		parts := fmt.Sprintf(`[{"type":"text","text":"流式内容 %d 中文测试"}]`, i)
		if _, err := sdb.Exec(`UPDATE messages SET parts=?, finished_at=? WHERE id=?;`, parts, i, "m1"); err != nil {
			t.Fatalf("full-triggers update %d err: %v", i, err)
		}
		if _, err := sdb.Exec(`UPDATE messages SET parts=?, finished_at=? WHERE id=?;`, parts, i, "m2"); err != nil {
			t.Fatalf("full-triggers update %d m2 err: %v", i, err)
		}
	}
	var ok string
	if err := sdb.QueryRow("PRAGMA integrity_check").Scan(&ok); err != nil {
		t.Fatal(err)
	}
	t.Logf("full triggers ncruces: integrity=%s OK", ok)
}

// 完整真实触发器组 + 双 goroutine 并发 + ncruces - 对齐真实双 agent 场景
func TestWALFullTriggersConcurrent(t *testing.T) {
	dir := t.TempDir()
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=page_size(4096)&_pragma=synchronous(NORMAL)&_pragma=cache_size(-8000)", url.PathEscape(filepath.Join(dir, "v.db")))
	sdb, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer sdb.Close()
	sdb.SetMaxOpenConns(1)
	for _, s := range []string{
		`CREATE TABLE sessions (id TEXT PRIMARY KEY, title TEXT NOT NULL, message_count INTEGER NOT NULL DEFAULT 0, updated_at INTEGER NOT NULL, created_at INTEGER NOT NULL);`,
		`CREATE TABLE messages (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, role TEXT NOT NULL, parts TEXT NOT NULL default '[]', model TEXT, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL, finished_at INTEGER);`,
		`CREATE VIRTUAL TABLE messages_fts USING fts5(parts, content='messages', content_rowid='rowid', tokenize='trigram');`,
		`CREATE TRIGGER messages_fts_insert AFTER INSERT ON messages BEGIN INSERT INTO messages_fts(rowid, parts) VALUES (new.rowid, new.parts); END;`,
		`CREATE TRIGGER messages_fts_delete AFTER DELETE ON messages BEGIN INSERT INTO messages_fts(messages_fts, rowid, parts) VALUES ('delete', old.rowid, old.parts); END;`,
		`CREATE TRIGGER messages_fts_update AFTER UPDATE ON messages WHEN length(old.parts) > 2 AND old.parts IS NOT new.parts BEGIN INSERT INTO messages_fts(messages_fts, rowid, parts) VALUES ('delete', old.rowid, old.parts); END;`,
		`CREATE TRIGGER messages_fts_update_reindex AFTER UPDATE ON messages WHEN old.parts IS NOT new.parts BEGIN INSERT INTO messages_fts(rowid, parts) VALUES (new.rowid, new.parts); END;`,
		`CREATE TRIGGER update_session_message_count_on_insert AFTER INSERT ON messages BEGIN UPDATE sessions SET message_count = message_count + 1 WHERE id = new.session_id; END;`,
		`INSERT INTO sessions (id, title, message_count, updated_at, created_at) VALUES ('s1','t',0,0,0);`,
	} {
		if _, err := sdb.Exec(s); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}
	// 用户消息 + 两条 assistant 初始
	for _, s := range []string{
		`INSERT INTO messages (id, session_id, role, parts, created_at, updated_at) VALUES ('u1','s1','user','[{"type":"text","text":"hello 中文"}]',1000,1000);`,
		`INSERT INTO messages (id, session_id, role, parts, created_at, updated_at) VALUES ('a1','s1','assistant','[]',1000,1000);`,
		`INSERT INTO messages (id, session_id, role, parts, created_at, updated_at) VALUES ('a2','s1','assistant','[]',1000,1000);`,
	} {
		if _, err := sdb.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	// 双 agent 并发: EventComplete UpdateMessage + TrackUsage UpdateSession
	var wg sync.WaitGroup
	errs := make(chan error, 10)
	agent := func(id string, n int) {
		defer wg.Done()
		for j := 0; j < 10; j++ {
			if _, err := sdb.Exec(`UPDATE messages SET parts=?, finished_at=?, updated_at=CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER) WHERE id=?;`, fmt.Sprintf(`[{"type":"text","text":"agent %s 第 %d 轮 内容 delta 中文"}]`, id, j), j, id); err != nil {
				errs <- fmt.Errorf("agent %s j%d: %w", id, j, err)
				return
			}
			if _, err := sdb.Exec(`UPDATE sessions SET message_count=message_count+0 WHERE id='s1';`); err != nil {
				errs <- fmt.Errorf("sess j%d: %w", j, err)
				return
			}
		}
	}
	wg.Add(2)
	go agent("a1", 0)
	go agent("a2", 1)
	wg.Wait()
	close(errs)
	bad := 0
	for e := range errs {
		bad++
		if bad <= 8 {
			t.Logf("ERR: %v", e)
		}
	}
	var ok string
	_ = sdb.QueryRow("PRAGMA integrity_check").Scan(&ok)
	if bad > 0 || ok != "ok" {
		t.Fatalf("full triggers concurrent: %d errors, integrity=%q", bad, ok)
	}
	t.Log("full triggers concurrent OK")
}

// 完整还原真实迁移后 schema: 含 update_messages_updated_at(AFTER UPDATE 第二个触发器)
func TestWALRealSchemaExact(t *testing.T) {
	dir := t.TempDir()
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=page_size(4096)&_pragma=foreign_keys(ON)&_pragma=synchronous(NORMAL)&_pragma=cache_size(-8000)", url.PathEscape(filepath.Join(dir, "v.db")))
	sdb, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer sdb.Close()
	sdb.SetMaxOpenConns(1)
	// 依迁移文件最终状态(20250424 initial + 20260804 fix + 20260919 ms)
	for _, s := range []string{
		`CREATE TABLE sessions (id TEXT PRIMARY KEY, parent_session_id TEXT, title TEXT NOT NULL, message_count INTEGER NOT NULL DEFAULT 0 CHECK (message_count >= 0), prompt_tokens INTEGER NOT NULL DEFAULT 0 CHECK (prompt_tokens >= 0), completion_tokens INTEGER NOT NULL DEFAULT 0 CHECK (completion_tokens>= 0), cost REAL NOT NULL DEFAULT 0.0 CHECK (cost >= 0.0), updated_at INTEGER NOT NULL, created_at INTEGER NOT NULL);`,
		`CREATE TRIGGER update_sessions_updated_at AFTER UPDATE ON sessions BEGIN UPDATE sessions SET updated_at = CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER) WHERE id = new.id; END;`,
		`CREATE TABLE messages (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, role TEXT NOT NULL, parts TEXT NOT NULL default '[]', model TEXT, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL, finished_at INTEGER, FOREIGN KEY (session_id) REFERENCES sessions (id) ON DELETE CASCADE);`,
		`CREATE INDEX idx_messages_session_id ON messages (session_id);`,
		`CREATE TRIGGER update_messages_updated_at AFTER UPDATE ON messages BEGIN UPDATE messages SET updated_at = CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER) WHERE id = new.id; END;`,
		`CREATE TRIGGER update_session_message_count_on_insert AFTER INSERT ON messages BEGIN UPDATE sessions SET message_count = message_count + 1 WHERE id = new.session_id; END;`,
		`CREATE TRIGGER update_session_message_count_on_delete AFTER DELETE ON messages BEGIN UPDATE sessions SET message_count = message_count - 1 WHERE id = old.session_id; END;`,
		`CREATE VIRTUAL TABLE messages_fts USING fts5(parts, content='messages', content_rowid='rowid', tokenize='trigram');`,
		`CREATE TRIGGER messages_fts_insert AFTER INSERT ON messages BEGIN INSERT INTO messages_fts(rowid, parts) VALUES (new.rowid, new.parts); END;`,
		`CREATE TRIGGER messages_fts_delete AFTER DELETE ON messages BEGIN INSERT INTO messages_fts(messages_fts, rowid, parts) VALUES ('delete', old.rowid, old.parts); END;`,
		`CREATE TRIGGER messages_fts_update AFTER UPDATE ON messages WHEN length(old.parts) > 2 AND old.parts IS NOT new.parts BEGIN INSERT INTO messages_fts(messages_fts, rowid, parts) VALUES ('delete', old.rowid, old.parts); END;`,
		`CREATE TRIGGER messages_fts_update_reindex AFTER UPDATE ON messages WHEN old.parts IS NOT new.parts BEGIN INSERT INTO messages_fts(rowid, parts) VALUES (new.rowid, new.parts); END;`,
		`INSERT INTO sessions (id, title, message_count, updated_at, created_at) VALUES ('s1','t',0,0,0);`,
		`INSERT INTO messages (id, session_id, role, parts, created_at, updated_at) VALUES ('u1','s1','user','[{"type":"text","text":"新建一个项目用于分析本地目录下的代码,并整理成md文件"}]',1790137663980,1790137663980);`,
		`INSERT INTO messages (id, session_id, role, parts, model, created_at, updated_at) VALUES ('assist-title','s1','assistant','[]','deepseek-v4-flash',1790137663981,1790137663981);`,
		`INSERT INTO messages (id, session_id, role, parts, model, created_at, updated_at) VALUES ('assist-main','s1','assistant','[]','deepseek-v4-flash',1790137663982,1790137663982);`,
	} {
		if _, err := sdb.Exec(s); err != nil {
			t.Fatalf("setup: %v\nstmt: %s", err, s)
		}
	}
	// 并发 title + main agent, 模拟 EventComplete: UpdateMessage(含 finished_at)
	var wg sync.WaitGroup
	errs := make(chan error, 10)
	agent := func(id string, finishedAt int64) {
		defer wg.Done()
		for j := 0; j < 8; j++ {
			parts := fmt.Sprintf(`[{"type":"text","text":"第%d轮 响应内容 流式 中文 %s"},{"reason":"stop","time":0}]`, j, id)
			if _, err := sdb.Exec(`UPDATE messages SET parts=?, finished_at=?, updated_at=CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER) WHERE id=?;`, parts, finishedAt, id); err != nil {
				errs <- fmt.Errorf("update %s j%d: %w", id, j, err)
				return
			}
		}
	}
	wg.Add(2)
	go agent("assist-title", 1790137664000)
	go agent("assist-main", 1790137664001)
	wg.Wait()
	close(errs)
	bad := 0
	for e := range errs {
		bad++
		if bad <= 5 {
			t.Logf("ERR: %v", e)
		}
	}
	var ok string
	_ = sdb.QueryRow("PRAGMA integrity_check").Scan(&ok)
	if bad > 0 || ok != "ok" {
		t.Fatalf("exact real schema concurrent: %d errors, integrity=%q", bad, ok)
	}
	t.Log("exact real schema OK")
}

// 完整迁移链 + FTS 空 parts 场景回归: 修复前 messages_fts_update 触发器
// 对 parts='[]' 的行 UPDATE 时, 'delete' 命令报 malformed。
func TestFTSMigratedEmptyPartsUpdate(t *testing.T) {
	dbPath := "/tmp/migtest.db"
	os.Remove(dbPath)
	os.Remove(dbPath + "-wal")
	os.Remove(dbPath + "-shm")
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=page_size(4096)&_pragma=foreign_keys(ON)&_pragma=synchronous(NORMAL)&_pragma=cache_size(-8000)", url.PathEscape(dbPath))
	sdb, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer sdb.Close()
	sdb.SetMaxOpenConns(1)

	goose.SetBaseFS(FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.Up(sdb, "migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if _, err := sdb.Exec(`INSERT INTO sessions (id, title, message_count, prompt_tokens, completion_tokens, cost, updated_at, created_at) VALUES ('s1','t',0,0,0,0,0,0);`); err != nil {
		t.Fatal(err)
	}
	if _, err := sdb.Exec(`INSERT INTO messages (id, session_id, role, parts, created_at, updated_at) VALUES ('m1','s1','assistant','[]',1000,1000);`); err != nil {
		t.Fatal(err)
	}
	if _, err := sdb.Exec(`UPDATE messages SET parts=?, finished_at=?, updated_at=CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER) WHERE id=?`,
		`[{"type":"tool_call","data":{"id":"c1","name":"bash","arguments":{"command":"ls"},"state":"running"}}]`, int64(2000), "m1"); err != nil {
		t.Fatalf("update empty->toolcalls: %v", err)
	}
	if _, err := sdb.Exec(`UPDATE messages SET parts=?, finished_at=? WHERE id=?`,
		`[{"type":"text","data":{"text":"最终回复"}}]`, int64(3000), "m1"); err != nil {
		t.Fatalf("update 2nd: %v", err)
	}
	if _, err := sdb.Exec(`UPDATE messages SET parts=?, finished_at=? WHERE id=?`,
		`[{"type":"tool_call","data":{"id":"c2","name":"read_skill","arguments":{"name":"planning"},"state":"running"}}]`, int64(4000), "m1"); err != nil {
		t.Fatalf("update 2nd content: %v", err)
	}
	var ok string
	if err := sdb.QueryRow("PRAGMA integrity_check").Scan(&ok); err != nil || ok != "ok" {
		t.Fatalf("integrity=%q err=%v", ok, err)
	}
	var n int
	if err := sdb.QueryRow(`SELECT count(*) FROM messages_fts WHERE messages_fts MATCH '"planning"'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("fts search new content n=%d err=%v", n, err)
	}
	if err := sdb.QueryRow(`SELECT count(*) FROM messages_fts WHERE messages_fts MATCH '"最终回复"'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("fts search stale content n=%d err=%v", n, err)
	}
	t.Log("migrated FTS empty-parts update OK")
}
