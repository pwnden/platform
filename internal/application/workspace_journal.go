package application

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/gofrs/flock"
	"github.com/pwnden/platform/internal/runtime"
)

type WorkspaceJournal interface {
	TrackWorkspace(string) error
	ForgetWorkspace(string) error
}

type workspaceJournal struct {
	mu    sync.Mutex
	path  string
	slugs map[string]bool
}

func (s *Service) workspacePath() (string, error) {
	root, err := filepath.Abs(s.repo)
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	config, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(config, "pwnden", "workspaces")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, fmt.Sprintf("%x", sha256.Sum256([]byte(root)))), nil
}

// LockWorkspace serializes server ownership with standalone mutating commands.
func (s *Service) LockWorkspace() (func() error, error) {
	path, err := s.workspacePath()
	if err != nil {
		return nil, err
	}
	lock := flock.New(path + ".lock")
	owned, err := lock.TryLock()
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, workspaceError("", WorkspaceBusy, errors.New("the problem installation is in use"))
	}
	return lock.Close, nil
}

// AcquireWorkspace removes only environments recorded by this installation's
// preceding server. The lock remains held until the new server finishes cleanup.
func (s *Service) AcquireWorkspace(ctx context.Context) (func() error, error) {
	release, err := s.LockWorkspace()
	if err != nil {
		return nil, err
	}
	path, err := s.workspacePath()
	if err != nil {
		release()
		return nil, err
	}
	j := &workspaceJournal{path: path + ".json", slugs: map[string]bool{}}
	data, err := os.ReadFile(j.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		release()
		return nil, err
	}
	if len(data) > 64<<10 {
		release()
		return nil, errors.New("workspace journal exceeds its limit")
	}
	if len(data) > 0 {
		var slugs []string
		if err := json.Unmarshal(data, &slugs); err != nil {
			release()
			return nil, err
		}
		for _, slug := range slugs {
			j.slugs[slug] = true
		}
	}
	s.journal = j
	var failure error
	for slug := range j.slugs {
		c, err := s.load(ctx, "recover", slug, true)
		if err == nil {
			cleanup, cancel := context.WithTimeout(ctx, 30*time.Second)
			err = runtime.CleanupTerminals(cleanup, c)
			if err == nil {
				_, err = s.Stop(cleanup, slug)
			}
			cancel()
		}
		if err == nil {
			err = s.ForgetWorkspace(slug)
		}
		failure = errors.Join(failure, err)
	}
	if failure != nil {
		release()
		return nil, workspaceError("", CleanupFailed, failure)
	}
	return release, nil
}

func (s *Service) TrackWorkspace(slug string) error {
	if s.journal == nil {
		return nil
	}
	if _, err := s.load(context.Background(), "workspace", slug, true); err != nil {
		return err
	}
	j := s.journal
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.slugs[slug] {
		return nil
	}
	j.slugs[slug] = true
	if err := j.save(); err != nil {
		delete(j.slugs, slug)
		return err
	}
	return nil
}
func (s *Service) ForgetWorkspace(slug string) error {
	if s.journal == nil {
		return nil
	}
	j := s.journal
	j.mu.Lock()
	defer j.mu.Unlock()
	if !j.slugs[slug] {
		return nil
	}
	delete(j.slugs, slug)
	if err := j.save(); err != nil {
		j.slugs[slug] = true
		return err
	}
	return nil
}
func (j *workspaceJournal) save() error {
	slugs := make([]string, 0, len(j.slugs))
	for slug := range j.slugs {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	data, err := json.Marshal(slugs)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(j.path), ".workspace-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, writeErr := f.Write(data)
	syncErr := f.Sync()
	closeErr := f.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return err
	}
	return os.Rename(f.Name(), j.path)
}

func (s *Service) CleanupWorkspace(ctx context.Context, slug string) error {
	c, err := s.load(ctx, "cleanup", slug, true)
	if err != nil {
		return err
	}
	terminalErr := runtime.CleanupTerminals(ctx, c)
	_, stopErr := s.Stop(ctx, slug)
	return errors.Join(terminalErr, stopErr)
}
