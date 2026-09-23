package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/logging"

	"github.com/pressly/goose/v3"
)

func Connect() (*sql.DB, error) {
	dataDir := config.Get().Data.Directory
	if dataDir == "" {
		return nil, fmt.Errorf("data.dir is not set")
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}
	dbPath := filepath.Join(dataDir, "hylbscode.db")

	// PRAGMA 通过 DSN 的 _pragma 参数固化: 驱动对每条新连接都会执行,
	// 避免 database/sql 连接池中部分连接未应用 journal_mode/page_size
	// 等连接级设置, 导致多连接并发写 WAL 时数据库损坏(malformed)。
	// busy_timeout 必须最先设置(驱动要求: encryption keys → busy timeout → locking mode)。
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=page_size(4096)&_pragma=foreign_keys(ON)&_pragma=synchronous(NORMAL)&_pragma=cache_size(-8000)", url.PathEscape(dbPath))

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 限制连接池: SQLite 单写者, 多连接并发写 WAL 容易相互干扰
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Verify connection
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	goose.SetBaseFS(FS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		logging.Error("Failed to set dialect", "error", err)
		return nil, fmt.Errorf("failed to set dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		logging.Error("Failed to apply migrations", "error", err)
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}
	return db, nil
}
