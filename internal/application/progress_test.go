package application

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/pwnden/platform/internal/progress"
	"github.com/pwnden/platform/internal/storage"
)

func TestAcceptedAnswerIsDurableAcrossCatalogPaths(t *testing.T) {
	ctx := context.Background()
	s, c := fixture(t, false)
	path := filepath.Join(t.TempDir(), "progress.db")
	store, err := storage.Open(ctx, path, "official")
	if err != nil {
		t.Fatal(err)
	}
	s.WithProgress(store)
	result, err := s.Submit(ctx, c.Slug, "wrong")
	if err != nil || result.Accepted {
		t.Fatal(result, err)
	}
	items, err := store.List(ctx)
	if err != nil || len(items) != 0 {
		t.Fatal(items, err)
	}
	result, err = s.Submit(ctx, c.Slug, "  pwnden{test}\n")
	if err != nil || !result.Accepted {
		t.Fatal(result, err)
	}
	_ = store.Close()
	store, err = storage.Open(ctx, path, "official")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	// A different checkout path represents an updated installed catalog.
	next, nextChallenge := fixture(t, false)
	next.WithProgress(store)
	list, err := next.List(ctx)
	if err != nil || len(list) != 1 || list[0].SolvedAt == nil {
		t.Fatal(list, err)
	}
	detail, err := next.Detail(ctx, nextChallenge.Slug)
	if err != nil || detail.Completion == nil || detail.Completion.Answer != "pwnden{test}" {
		t.Fatal(detail, err)
	}
}

type failedProgress struct{}

func (failedProgress) List(context.Context) (map[string]progress.Completion, error) {
	return nil, errors.New("unavailable")
}
func (failedProgress) Save(context.Context, progress.Completion) error {
	return errors.New("unavailable")
}

func TestProgressFailureIsNotReportedAsSavedSuccess(t *testing.T) {
	s, c := fixture(t, false)
	s.WithProgress(failedProgress{})
	result, err := s.Submit(context.Background(), c.Slug, "pwnden{test}")
	requireCode(t, err, StorageFailed)
	if result.Accepted {
		t.Fatal("failed save reported success")
	}
	_, err = s.List(context.Background())
	requireCode(t, err, StorageFailed)
}
