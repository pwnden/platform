// Package storage implements the platform's durable state in a local SQLite file.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pwnden/platform/internal/progress"
	_ "modernc.org/sqlite"
)

type SQLite struct {
	db        *sql.DB
	namespace string
}

func Open(ctx context.Context, path, namespace string) (*SQLite, error) {
	if namespace == "" {
		return nil, errors.New("storage namespace is required")
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = file.Close(); err != nil {
		return nil, err
	}
	uriPath := filepath.ToSlash(path)
	if !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	uri := url.URL{Scheme: "file", Path: uriPath}
	query := url.Values{"_pragma": {"busy_timeout(5000)", "foreign_keys(1)"}}
	uri.RawQuery = query.Encode()
	db, err := sql.Open("sqlite", uri.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	store := &SQLite{db: db, namespace: namespace}
	if err = store.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

// Migrations are transactional; newer schemas are retained and rejected.
func (s *SQLite) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	if err = tx.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version > 1 {
		return fmt.Errorf("unsupported state schema %d", version)
	}
	if version == 0 {
		_, err = tx.ExecContext(ctx, `CREATE TABLE completions (
			namespace TEXT NOT NULL, slug TEXT NOT NULL, answer TEXT NOT NULL,
			solved_at TEXT NOT NULL, PRIMARY KEY (namespace, slug)
		); PRAGMA user_version=1;`)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLite) List(ctx context.Context) (map[string]progress.Completion, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT slug, answer, solved_at FROM completions WHERE namespace=?", s.namespace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := make(map[string]progress.Completion)
	for rows.Next() {
		var item progress.Completion
		var timestamp string
		if err = rows.Scan(&item.Slug, &item.Answer, &timestamp); err != nil {
			return nil, err
		}
		item.SolvedAt, err = time.Parse(time.RFC3339Nano, timestamp)
		if err != nil {
			return nil, err
		}
		results[item.Slug] = item
	}
	return results, rows.Err()
}

func (s *SQLite) Save(ctx context.Context, item progress.Completion) error {
	if item.Slug == "" || item.Answer == "" || item.SolvedAt.IsZero() {
		return errors.New("completion is incomplete")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO completions (namespace, slug, answer, solved_at)
		VALUES (?, ?, ?, ?) ON CONFLICT (namespace, slug) DO UPDATE SET answer=excluded.answer`,
		s.namespace, item.Slug, item.Answer, item.SolvedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *SQLite) Close() error { return s.db.Close() }

var _ progress.Store = (*SQLite)(nil)
