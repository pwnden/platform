package application

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

// discardBackend records journal and cleanup calls like the installed service,
// so a test can see whether a record is forgotten before its resources are gone.
type discardBackend struct {
	*workspaceBackend
	cleanups    atomic.Int32
	failCleanup atomic.Bool
	openErr     error
	mu          sync.Mutex
	tracked     map[string]bool
}

func newDiscardBackend() *discardBackend {
	return &discardBackend{workspaceBackend: &workspaceBackend{}, tracked: map[string]bool{}}
}
func (b *discardBackend) TrackWorkspace(slug string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tracked[slug] = true
	return nil
}
func (b *discardBackend) ForgetWorkspace(slug string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.tracked, slug)
	return nil
}
func (b *discardBackend) isTracked(slug string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.tracked[slug]
}
func (b *discardBackend) CleanupWorkspace(context.Context, string) error {
	b.cleanups.Add(1)
	if b.failCleanup.Load() {
		return errors.New("keeper removal failed")
	}
	return nil
}
func (b *discardBackend) OpenTerminal(ctx context.Context, slug string, cols, rows int) (TerminalSession, error) {
	if b.openErr != nil {
		return nil, b.openErr
	}
	return b.workspaceBackend.OpenTerminal(ctx, slug, cols, rows)
}

func TestFileWorkspaceEndsReleaseBackendResourcesBeforeForgettingTheRecord(t *testing.T) {
	end := map[string]func(*testing.T, *discardBackend, *Workspaces){
		"shell exit": func(t *testing.T, b *discardBackend, m *Workspaces) {
			attachWorkspace(t, m, "a")
			shell, _ := b.shells.Load("a")
			shell.(*workspaceShell).Close()
		},
		"explicit end": func(t *testing.T, _ *discardBackend, m *Workspaces) {
			attachWorkspace(t, m, "a")
			if err := m.EndTerminal("a"); err != nil {
				t.Fatal(err)
			}
		},
		"open failure": func(t *testing.T, b *discardBackend, m *Workspaces) {
			b.openErr = errors.New("terminal preparation failed")
			if _, err := m.Attach("a", 80, 24); err == nil {
				t.Fatal("failed open reported success")
			}
		},
	}
	for name, run := range end {
		t.Run(name, func(t *testing.T) {
			b := newDiscardBackend()
			m := NewWorkspaces(context.Background(), b)
			t.Cleanup(func() { m.Close() })
			run(t, b, m)
			eventuallyWorkspace(t, func() bool { return len(m.List()) == 0 })
			if b.cleanups.Load() != 1 || b.isTracked("a") {
				t.Fatalf("cleanups=%d tracked=%v", b.cleanups.Load(), b.isTracked("a"))
			}
		})
	}
}

func TestFailedWorkspaceCleanupKeepsTheRecoveryRecord(t *testing.T) {
	b := newDiscardBackend()
	b.failCleanup.Store(true)
	m := NewWorkspaces(context.Background(), b)
	t.Cleanup(func() { m.Close() })
	attachWorkspace(t, m, "a")
	shell, _ := b.shells.Load("a")
	shell.(*workspaceShell).Close()
	eventuallyWorkspace(t, func() bool { return b.cleanups.Load() == 1 })
	if len(m.List()) != 1 || !b.isTracked("a") {
		t.Fatalf("entries=%d tracked=%v: record dropped while resources may remain", len(m.List()), b.isTracked("a"))
	}
}
