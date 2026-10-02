package cli

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pwnden/platform/internal/application"
	"github.com/pwnden/platform/internal/storage"
)

func playerStorage(ctx context.Context, service *application.Service, repo string) (*storage.SQLite, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	namespace := "official"
	if repo != "" {
		path, err := filepath.Abs(repo)
		if err != nil {
			return nil, err
		}
		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			return nil, err
		}
		namespace = fmt.Sprintf("development:%x", sha256.Sum256([]byte(path)))
	}
	store, err := storage.Open(ctx, filepath.Join(base, "pwnden", "progress.db"), namespace)
	if err != nil {
		return nil, err
	}
	service.WithProgress(store)
	return store, nil
}
