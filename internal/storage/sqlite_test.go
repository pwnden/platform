package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/pwnden/platform/internal/progress"
)

func TestCompletionSurvivesRestartAndNamespaces(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state", "progress.db")
	s, err := Open(ctx, path, "official")
	if err != nil {
		t.Fatal(err)
	}
	first := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	for _, item := range []progress.Completion{{Slug: "test", Answer: "first", SolvedAt: first}, {Slug: "test", Answer: "second", SolvedAt: first.Add(time.Hour)}} {
		if err := s.Save(ctx, item); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(ctx, path, "official")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	items, err := s.List(ctx)
	if err != nil || len(items) != 1 || items["test"].Answer != "second" || !items["test"].SolvedAt.Equal(first) {
		t.Fatalf("restored: %+v %v", items, err)
	}
	dev, err := Open(ctx, path, "development")
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()
	items, err = dev.List(ctx)
	if err != nil || len(items) != 0 {
		t.Fatal("namespace leak", items, err)
	}
	if err := dev.Save(ctx, progress.Completion{}); err == nil {
		t.Fatal("empty completion stored")
	}
}

func TestNewerSchemaIsRejectedWithoutLosingRecords(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "progress.db")
	s, err := Open(ctx, path, "official")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(ctx, progress.Completion{Slug: "test", Answer: "saved", SolvedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("PRAGMA user_version=2"); err != nil {
		t.Fatal(err)
	}
	_ = s.Close()
	if _, err := Open(ctx, path, "official"); err == nil {
		t.Fatal("newer schema opened")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var answer string
	if err := db.QueryRow("SELECT answer FROM completions").Scan(&answer); err != nil || answer != "saved" {
		t.Fatal("record changed", answer, err)
	}
}
