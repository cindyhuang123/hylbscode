package search

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"unicode/utf8"
)

type Result struct {
	MessageID string
	SessionID string
	Role      string
	Snippet   string
	CreatedAt int64
}

type Service interface {
	Search(ctx context.Context, sessionID, query string, limit int) ([]Result, error)
	SearchAll(ctx context.Context, query string, limit int) ([]Result, error)
}

type service struct {
	conn *sql.DB
}

func NewService(conn *sql.DB) Service {
	return &service{conn: conn}
}

func (s *service) Search(ctx context.Context, sessionID, query string, limit int) ([]Result, error) {
	return s.search(ctx, sessionID, query, limit)
}

func (s *service) SearchAll(ctx context.Context, query string, limit int) ([]Result, error) {
	return s.search(ctx, "", query, limit)
}

func (s *service) search(ctx context.Context, sessionID, query string, limit int) ([]Result, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	// Trigram tokenizer requires at least 3 characters; fall back to LIKE for shorter queries.
	if utf8.RuneCountInString(query) < 3 {
		return s.searchLike(ctx, sessionID, query, limit)
	}
	return s.searchFTS(ctx, sessionID, query, limit)
}

func (s *service) searchFTS(ctx context.Context, sessionID, query string, limit int) ([]Result, error) {
	phrase := `"` + strings.ReplaceAll(query, `"`, `""`) + `"`
	rows, err := s.conn.QueryContext(ctx, `
		SELECT m.id, m.session_id, m.role, snippet(messages_fts, 0, '[', ']', '...', 24), m.created_at
		FROM messages_fts
		JOIN messages m ON m.rowid = messages_fts.rowid
		WHERE messages_fts MATCH ? AND (? = '' OR m.session_id = ?)
		ORDER BY rank
		LIMIT ?`, phrase, sessionID, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	defer rows.Close()
	return scanResults(rows)
}

func (s *service) searchLike(ctx context.Context, sessionID, query string, limit int) ([]Result, error) {
	rows, err := s.conn.QueryContext(ctx, `
		SELECT id, session_id, role, substr(parts, 1, 300), created_at
		FROM messages
		WHERE (? = '' OR session_id = ?) AND parts LIKE '%' || ? || '%'
		ORDER BY created_at DESC
		LIMIT ?`, sessionID, sessionID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	defer rows.Close()
	return scanResults(rows)
}

func scanResults(rows *sql.Rows) ([]Result, error) {
	var results []Result
	for rows.Next() {
		var r Result
		if err := rows.Scan(&r.MessageID, &r.SessionID, &r.Role, &r.Snippet, &r.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
