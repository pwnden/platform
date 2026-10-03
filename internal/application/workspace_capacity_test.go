package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/pwnden/platform/internal/runtime"
)

type pressureBackend struct {
	workspaceBackend
	mu             sync.Mutex
	used           map[string]int
	capacity       int
	external       int
	stopped        []string
	calls          map[string]int
	failure        error
	cleanupFailure bool
}

func newPressureBackend(capacity int) *pressureBackend {
	return &pressureBackend{workspaceBackend: workspaceBackend{service: true}, capacity: capacity, used: map[string]int{}, calls: map[string]int{}}
}
func (b *pressureBackend) Status(_ context.Context, slug string) (RunStatus, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.service {
		return RunStatus{Slug: slug, Kind: KindFile, State: "ready"}, nil
	}
	state := "stopped"
	if b.used[slug] != 0 {
		state = "running"
	}
	return RunStatus{Slug: slug, Kind: KindService, State: state}, nil
}
func (b *pressureBackend) allocate(key string, cost int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.calls[key]++
	if b.failure != nil {
		return b.failure
	}
	total := b.external + cost
	for _, value := range b.used {
		total += value
	}
	if total > b.capacity {
		return operationError(context.Background(), "run", key, ExecutionFailed, runtime.ErrResourceLimit)
	}
	b.used[key] = cost
	return nil
}
func (b *pressureBackend) Run(_ context.Context, slug string) (RunInfo, error) {
	cost := 1
	if slug == "heavy" {
		cost = 2
	}
	return RunInfo{Slug: slug, Kind: KindService}, b.allocate(slug, cost)
}
func (b *pressureBackend) OpenTerminal(ctx context.Context, slug string, cols, rows int) (TerminalSession, error) {
	if err := b.allocate(slug+"-terminal", 1); err != nil {
		return nil, err
	}
	return b.workspaceBackend.OpenTerminal(ctx, slug, cols, rows)
}
func (b *pressureBackend) CleanupWorkspace(_ context.Context, slug string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cleanupFailure {
		return errors.New("container removal failed")
	}
	b.stopped = append(b.stopped, slug)
	delete(b.used, slug)
	delete(b.used, slug+"-terminal")
	return nil
}
func pressureView(t *testing.T, m *Workspaces, slug string) *WorkspaceView {
	t.Helper()
	v, err := m.View(context.Background(), slug)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestWorkspaceResourcePressureReclaimsIdleAndProtectsOtherOwners(t *testing.T) {
	b := newPressureBackend(3)
	b.external = 1 // Another installation is counted, but never owned here.
	m := NewWorkspaces(context.Background(), b)
	defer m.Close()
	old := pressureView(t, m, "old")
	old.Close()
	active := pressureView(t, m, "active")
	defer active.Close()
	next := pressureView(t, m, "next")
	defer next.Close()
	if old.Active() || !active.Active() || !next.Active() || b.external != 1 || len(b.stopped) != 1 || b.stopped[0] != "old" || b.calls["next"] != 2 {
		t.Fatalf("unsafe pressure recovery: stopped=%v calls=%v", b.stopped, b.calls)
	}
}

func TestWorkspacePressureRetriesUntilEnoughCapacityIsReleased(t *testing.T) {
	b := newPressureBackend(3)
	m := NewWorkspaces(context.Background(), b)
	defer m.Close()
	pressureView(t, m, "oldest").Close()
	pressureView(t, m, "recent").Close()
	active := pressureView(t, m, "active")
	defer active.Close()
	if _, err := m.Run(context.Background(), "heavy"); err != nil {
		t.Fatal(err)
	}
	if len(b.stopped) != 2 || b.stopped[0] != "oldest" || b.stopped[1] != "recent" || b.calls["heavy"] != 3 || !active.Active() {
		t.Fatalf("ordered bounded retries: %v %v", b.stopped, b.calls)
	}
}

func TestWorkspaceTerminalPressureProtectsCurrentServiceAndAttachedShell(t *testing.T) {
	b := newPressureBackend(4)
	m := NewWorkspaces(context.Background(), b)
	defer m.Close()
	pressureView(t, m, "old").Close()
	active := attachWorkspace(t, m, "active")
	defer active.Close()
	next := attachWorkspace(t, m, "next")
	defer next.Close()
	if len(b.stopped) != 1 || b.stopped[0] != "old" || b.calls["next-terminal"] != 2 || b.used["next"] != 1 || m.entry("active") == nil {
		t.Fatalf("terminal recovery lost a live service: %v %v", b.stopped, b.used)
	}
}

func TestWorkspaceFileTerminalPressure(t *testing.T) {
	b := newPressureBackend(2)
	b.service = false
	b.external = 1
	m := NewWorkspaces(context.Background(), b)
	defer m.Close()
	old := attachWorkspace(t, m, "old")
	old.Close()
	next := attachWorkspace(t, m, "next")
	defer next.Close()
	if len(b.stopped) != 1 || b.stopped[0] != "old" || b.calls["next-terminal"] != 2 {
		t.Fatal("file shell was not reclaimed", b.stopped)
	}
	select {
	case <-old.process.done:
	default:
		t.Fatal("evicted shell was left running")
	}
}

func TestWorkspacePressureWithoutEligibleIdleEnvironment(t *testing.T) {
	b := newPressureBackend(2)
	b.external = 1
	m := NewWorkspaces(context.Background(), b)
	defer m.Close()
	active := pressureView(t, m, "active")
	defer active.Close()
	_, err := m.View(context.Background(), "next")
	requireCode(t, err, ResourceLimit)
	if len(b.stopped) != 0 || !active.Active() || b.calls["next"] != 1 {
		t.Fatal("live environment was interrupted or retry was unbounded")
	}
}

func TestWorkspacePressureCleanupFailureRetainsOwnership(t *testing.T) {
	b := newPressureBackend(1)
	m := NewWorkspaces(context.Background(), b)
	defer func() { b.cleanupFailure = false; m.Close() }()
	old := pressureView(t, m, "old")
	old.Close()
	b.cleanupFailure = true
	_, err := m.View(context.Background(), "next")
	requireCode(t, err, CleanupFailed)
	if !old.Active() || b.calls["next"] != 1 || len(b.stopped) != 0 {
		t.Fatal("failed cleanup discarded ownership or retried creation")
	}
}

func TestWorkspaceOrdinaryAndCanceledFailuresDoNotEvict(t *testing.T) {
	for _, failure := range []error{
		errors.New("image build failed"),
		operationError(context.Background(), "run", "next", ExecutionFailed, errors.Join(runtime.ErrResourceLimit, runtime.ErrCleanupFailed)),
		operationError(context.Background(), "run", "next", ExecutionFailed, errors.Join(runtime.ErrResourceLimit, context.Canceled)),
	} {
		b := newPressureBackend(1)
		m := NewWorkspaces(context.Background(), b)
		old := pressureView(t, m, "old")
		old.Close()
		b.failure = failure
		if _, err := m.View(context.Background(), "next"); err == nil {
			t.Fatal("failure lost")
		}
		if !old.Active() || len(b.stopped) != 0 || b.calls["next"] != 1 {
			t.Fatal("non-admission failure evicted an environment")
		}
		m.Close()
	}
}

func TestWorkspaceConcurrentAdmissionsProtectViews(t *testing.T) {
	b := newPressureBackend(2)
	m := NewWorkspaces(context.Background(), b)
	m.limit = 32 // Exercise the resource budget separately from the slot limit.
	defer m.Close()
	pressureView(t, m, "old").Close()
	var wg sync.WaitGroup
	views := make(chan *WorkspaceView, 12)
	errs := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v, err := m.View(context.Background(), string(rune('a'+i)))
			if err != nil {
				errs <- err
			} else {
				views <- v
			}
		}(i)
	}
	wg.Wait()
	close(views)
	close(errs)
	if len(views) != 2 || len(errs) != 10 || len(b.stopped) != 1 || b.stopped[0] != "old" {
		t.Fatal("concurrent pressure accounting failed", len(views), len(errs), b.stopped)
	}
	for err := range errs {
		requireCode(t, err, ResourceLimit)
	}
	for v := range views {
		if !v.Active() {
			t.Fatal("connected view was evicted")
		}
		v.Close()
	}
}

func TestWorkspaceCanceledPreparationDoesNotEvict(t *testing.T) {
	b := newPressureBackend(1)
	m := NewWorkspaces(context.Background(), b)
	defer m.Close()
	old := pressureView(t, m, "old")
	old.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := m.View(ctx, "next")
	requireCode(t, err, Canceled)
	if !old.Active() || len(b.stopped) != 0 || b.calls["next"] != 0 {
		t.Fatal("canceled wait evicted a retained problem")
	}
}

func TestWorkspacePendingReconnectIsProtectedBeforeItsEntryLock(t *testing.T) {
	b := newPressureBackend(1)
	m := NewWorkspaces(context.Background(), b)
	defer m.Close()
	old := pressureView(t, m, "old")
	old.Close()
	e, err := m.reserve(context.Background(), "old")
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.View(context.Background(), "next")
	m.finishPreparation(e)
	requireCode(t, err, ResourceLimit)
	if !old.Active() || len(b.stopped) != 0 {
		t.Fatal("pending reconnect was evicted")
	}
}

func TestWorkspaceReconnectAfterCleanupClaimAutomaticallyPreparesAgain(t *testing.T) {
	b := newPressureBackend(1)
	m := NewWorkspaces(context.Background(), b)
	defer m.Close()
	old := pressureView(t, m, "old")
	old.Close()
	e := old.entry
	e.mu.Lock()
	if !m.claimIdle(e) {
		e.mu.Unlock()
		t.Fatal("idle run was not claimed")
	}
	done := make(chan error, 1)
	go func() {
		view, err := m.View(context.Background(), "old")
		if view != nil {
			view.Close()
		}
		done <- err
	}()
	if err := m.stopLocked(e); err != nil {
		m.finishReclaim(e)
		e.mu.Unlock()
		t.Fatal(err)
	}
	m.finishReclaim(e)
	e.mu.Unlock()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("reconnect waited indefinitely for completed cleanup")
	}
	if old.Active() || m.entry("old") == nil || b.calls["old"] != 2 {
		t.Fatal("reconnect reused a removed run")
	}
}
