package application

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/charmbracelet/x/vt"
)

type workspaceShell struct {
	r      *io.PipeReader
	w      *io.PipeWriter
	once   sync.Once
	closed chan struct{}
	input  []byte
	mu     sync.Mutex
}

func newWorkspaceShell() *workspaceShell {
	r, w := io.Pipe()
	return &workspaceShell{r: r, w: w, closed: make(chan struct{})}
}
func (s *workspaceShell) Read(p []byte) (int, error) { return s.r.Read(p) }
func (s *workspaceShell) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.input = append(s.input, p...)
	return len(p), nil
}
func (s *workspaceShell) Resize(context.Context, int, int) error { return nil }
func (s *workspaceShell) Wait(context.Context) (int, error)      { return 0, nil }
func (s *workspaceShell) Close() error {
	s.once.Do(func() { s.r.Close(); s.w.Close(); close(s.closed) })
	return nil
}

type workspaceBackend struct {
	opens    atomic.Int32
	stops    atomic.Int32
	service  bool
	shells   sync.Map
	failStop bool
	runErr   error
}

func (b *workspaceBackend) Status(_ context.Context, slug string) (RunStatus, error) {
	kind := KindFile
	state := "ready"
	if b.service {
		kind = KindService
		state = "stopped"
	}
	return RunStatus{Slug: slug, Kind: kind, State: state}, nil
}
func (b *workspaceBackend) Run(_ context.Context, slug string) (RunInfo, error) {
	return RunInfo{Slug: slug, Kind: KindService}, b.runErr
}
func (b *workspaceBackend) Stop(_ context.Context, slug string) (StopInfo, error) {
	b.stops.Add(1)
	if b.failStop {
		return StopInfo{}, errors.New("cleanup failed")
	}
	return StopInfo{Slug: slug}, nil
}
func (b *workspaceBackend) OpenTerminal(_ context.Context, slug string, _, _ int) (TerminalSession, error) {
	b.opens.Add(1)
	shell := newWorkspaceShell()
	b.shells.Store(slug, shell)
	return shell, nil
}
func attachWorkspace(t *testing.T, m *Workspaces, slug string) *TerminalAttachment {
	t.Helper()
	a, err := m.Attach(slug, 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func eventuallyWorkspace(t *testing.T, fn func() bool) {
	t.Helper()
	end := time.Now().Add(time.Second)
	for !fn() {
		if time.Now().After(end) {
			t.Fatal("workspace condition timed out")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestWorkspaceRetentionExpiryAndTimerReset(t *testing.T) {
	b := &workspaceBackend{}
	m := NewWorkspaces(context.Background(), b)
	m.idle = 60 * time.Millisecond
	t.Cleanup(func() { m.Close() })
	a := attachWorkspace(t, m, "a")
	time.Sleep(80 * time.Millisecond)
	if len(m.List()) != 1 || !m.List()[0].Connected {
		t.Fatal("visible environment expired without input")
	}
	a.Close()
	first := *m.List()[0].ExpiresAt
	time.Sleep(20 * time.Millisecond)
	a = attachWorkspace(t, m, "a")
	if b.opens.Load() != 1 {
		t.Fatal("reconnect replaced the shell")
	}
	time.Sleep(65 * time.Millisecond)
	if len(m.List()) != 1 {
		t.Fatal("canceled timer deleted active environment")
	}
	a.Close()
	if !m.List()[0].ExpiresAt.After(first) {
		t.Fatal("expiry was not reset")
	}
	eventuallyWorkspace(t, func() bool { return len(m.List()) == 0 })
	attachWorkspace(t, m, "a")
	if b.opens.Load() != 2 {
		t.Fatal("expired environment was reused")
	}
}

func TestWorkspaceShellExitReleasesFileSlotAndServicePresenceSurvives(t *testing.T) {
	for _, service := range []bool{false, true} {
		b := &workspaceBackend{service: service}
		m := NewWorkspaces(context.Background(), b)
		a := attachWorkspace(t, m, "a")
		shell, _ := b.shells.Load("a")
		shell.(*workspaceShell).Close()
		eventuallyWorkspace(t, func() bool { return a.process.finished() })
		if service {
			if len(m.List()) != 1 || !m.List()[0].Connected {
				t.Fatal("exit lost service presence")
			}
		} else {
			eventuallyWorkspace(t, func() bool { return len(m.List()) == 0 })
		}
		a.Close()
		m.Close()
	}
}

func TestWorkspaceEndTerminalDetachesWithoutRemovingService(t *testing.T) {
	b := &workspaceBackend{service: true}
	m := NewWorkspaces(context.Background(), b)
	defer m.Close()
	a := attachWorkspace(t, m, "a")
	if err := m.EndTerminal("a"); err != nil {
		t.Fatal(err)
	}
	a.Close()
	if len(m.List()) != 1 || m.List()[0].ExpiresAt == nil || b.stops.Load() != 0 {
		t.Fatal("terminal-only stop lost retained service")
	}
}

func TestWorkspaceReconnectDuringFragmentedOutput(t *testing.T) {
	for _, parts := range [][2]string{{"\xea\xb0", "\x80"}, {"\x1b[3", "1mR\x1b[0m"}} {
		b := &workspaceBackend{}
		m := NewWorkspaces(context.Background(), b)
		a := attachWorkspace(t, m, "a")
		a.Close()
		shell, _ := b.shells.Load("a")
		shell.(*workspaceShell).w.Write([]byte(parts[0]))
		eventuallyWorkspace(t, func() bool { a.process.mu.Lock(); defer a.process.mu.Unlock(); return len(a.process.pending) > 0 })
		next := attachWorkspace(t, m, "a")
		mirror := vt.NewEmulator(80, 24)
		mirror.Write(next.Snapshot)
		shell.(*workspaceShell).w.Write([]byte(parts[1]))
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		frame, err := next.Next(ctx)
		cancel()
		if err != nil {
			t.Fatal(err)
		}
		mirror.Write(frame)
		want := "가"
		if parts[0][0] == 27 {
			want = "R"
		}
		if !strings.Contains(mirror.Render(), want) {
			t.Fatalf("split output lost on reattach: %q", mirror.Render())
		}
		next.Close()
		m.Close()
	}
}

func TestWorkspaceReconnectToShorterScreenPreservesRecentOutput(t *testing.T) {
	b := &workspaceBackend{}
	m := NewWorkspaces(context.Background(), b)
	defer m.Close()
	a, err := m.Attach("a", 80, 40)
	if err != nil {
		t.Fatal(err)
	}
	a.Close()
	shell, _ := b.shells.Load("a")
	shell.(*workspaceShell).w.Write([]byte(strings.Repeat("old\r\n", 60) + "recent-output\r\n$ "))
	eventuallyWorkspace(t, func() bool {
		a.process.mu.Lock()
		defer a.process.mu.Unlock()
		return strings.Contains(a.process.screen.Render(), "recent-output")
	})
	next := attachWorkspace(t, m, "a")
	if !strings.Contains(string(next.Snapshot), "recent-output") {
		t.Fatal("shorter viewport cropped recent output")
	}
	next.Close()
}
func TestWorkspaceLimitPreservesExistingEnvironments(t *testing.T) {
	b := &workspaceBackend{}
	m := NewWorkspaces(context.Background(), b)
	t.Cleanup(func() { m.Close() })
	for i := 0; i < WorkspaceLimit; i++ {
		a := attachWorkspace(t, m, string(rune('a'+i)))
		a.Close()
	}
	_, err := m.Attach("overflow", 80, 24)
	var failure *Error
	if !errors.As(err, &failure) || failure.Code != WorkspaceFull || b.opens.Load() != 10 {
		t.Fatalf("limit not enforced: %v", err)
	}
	if err := m.Stop("a"); err != nil {
		t.Fatal(err)
	}
	attachWorkspace(t, m, "overflow")
}
func TestWorkspaceDetachedOutputAndScreenRestore(t *testing.T) {
	b := &workspaceBackend{}
	m := NewWorkspaces(context.Background(), b)
	t.Cleanup(func() { m.Close() })
	a := attachWorkspace(t, m, "a")
	a.Close()
	value, _ := b.shells.Load("a")
	shell := value.(*workspaceShell)
	for _, part := range [][]byte{[]byte("\x1b[2J\x1b[Hprompt> "), []byte{0xea, 0xb0}, []byte{0x80}, []byte("\x1b[?2004h")} {
		if _, err := shell.w.Write(part); err != nil {
			t.Fatal(err)
		}
	}
	eventuallyWorkspace(t, func() bool {
		a.process.mu.Lock()
		defer a.process.mu.Unlock()
		return strings.Contains(string(a.process.snapshot()), "\x1b[?2004h")
	})
	a = attachWorkspace(t, m, "a")
	mirror := vt.NewEmulator(80, 24)
	go io.Copy(io.Discard, mirror)
	defer mirror.InputPipe().(io.Closer).Close()
	mirror.Write(a.Snapshot)
	if !strings.Contains(mirror.String(), "prompt> 가") {
		t.Fatalf("UTF-8 screen lost: %q", mirror.String())
	}
	if !strings.Contains(string(a.Snapshot), "\x1b[?2004h") {
		t.Fatal("bracketed paste mode lost")
	}
	if err := a.Input([]byte{3, 0, 255}); err != nil {
		t.Fatal(err)
	}
	eventuallyWorkspace(t, func() bool {
		shell.mu.Lock()
		defer shell.mu.Unlock()
		return string(shell.input) == string([]byte{3, 0, 255})
	})
}
func TestWorkspaceAlternateScreenRestore(t *testing.T) {
	b := &workspaceBackend{}
	m := NewWorkspaces(context.Background(), b)
	t.Cleanup(func() { m.Close() })
	a := attachWorkspace(t, m, "a")
	a.Close()
	value, _ := b.shells.Load("a")
	shell := value.(*workspaceShell)
	shell.w.Write([]byte("main\x1b[?1049h\x1b[H한글 editor\x1b[?25l"))
	eventuallyWorkspace(t, func() bool {
		a.process.mu.Lock()
		defer a.process.mu.Unlock()
		return strings.Contains(a.process.screen.String(), "한글 editor")
	})
	a = attachWorkspace(t, m, "a")
	mirror := vt.NewEmulator(80, 24)
	go io.Copy(io.Discard, mirror)
	defer mirror.InputPipe().(io.Closer).Close()
	mirror.Write(a.Snapshot)
	if !mirror.IsAltScreen() || !strings.Contains(mirror.String(), "한글 editor") {
		t.Fatalf("alternate screen lost: %q", mirror.String())
	}
	mirror.WriteString("\x1b[?1049l")
	if !strings.Contains(mirror.String(), "main") {
		t.Fatal("normal screen lost when leaving editor")
	}
}
func TestWorkspaceShutdownAndCleanupFailure(t *testing.T) {
	b := &workspaceBackend{service: true, failStop: true}
	m := NewWorkspaces(context.Background(), b)
	a := attachWorkspace(t, m, "a")
	a.Close()
	attachWorkspace(t, m, "b")
	if err := m.Close(); err == nil {
		t.Fatal("cleanup failure lost")
	}
	if b.stops.Load() != 2 {
		t.Fatal("one failure skipped remaining environments")
	}
	if len(m.List()) != 2 {
		t.Fatal("failed cleanup discarded ownership")
	}
}

type blockedWorkspaceShell struct {
	*workspaceShell
	entered chan struct{}
	once    sync.Once
}

func (s *blockedWorkspaceShell) Write([]byte) (int, error) {
	s.once.Do(func() { close(s.entered) })
	<-s.closed
	return 0, io.ErrClosedPipe
}
func TestWorkspaceBlockedInputAllowsDetachAndBoundsQueue(t *testing.T) {
	shell := &blockedWorkspaceShell{workspaceShell: newWorkspaceShell(), entered: make(chan struct{})}
	p := newTerminalProcess(shell, 80, 24)
	defer p.stop()
	a, _ := p.attach(80, 24)
	if err := a.Input([]byte("first")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-shell.entered:
	case <-time.After(time.Second):
		t.Fatal("writer did not start")
	}
	for i := 0; i < 16; i++ {
		if err := a.Input([]byte("queued")); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.Input([]byte("overflow")); err == nil {
		t.Fatal("input queue grew without bound")
	}
	p.detach(a)
	if _, err := p.attach(80, 24); err != nil {
		t.Fatal("blocked input prevented reattachment:", err)
	}
}
func TestWorkspaceDetachedTerminalQueriesKeepDraining(t *testing.T) {
	shell := newWorkspaceShell()
	p := newTerminalProcess(shell, 80, 24)
	defer p.stop()
	shell.w.Write([]byte(strings.Repeat("\x1b[6n", 1000) + "finished"))
	eventuallyWorkspace(t, func() bool { p.mu.Lock(); defer p.mu.Unlock(); return strings.Contains(p.screen.Render(), "finished") })
	eventuallyWorkspace(t, func() bool {
		shell.mu.Lock()
		defer shell.mu.Unlock()
		return strings.Contains(string(shell.input), "\x1b[1;1R")
	})
}

func TestWorkspaceFailedRunRetainsCleanupOwnership(t *testing.T) {
	for _, code := range []Code{CleanupFailed, AlreadyRunning} {
		b := &workspaceBackend{runErr: workspaceError("a", code, errors.New("run failed"))}
		m := NewWorkspaces(context.Background(), b)
		if _, err := m.Run(context.Background(), "a"); err == nil {
			t.Fatal("failure lost")
		}
		if len(m.List()) != 1 {
			t.Fatal("failed start lost ownership")
		}
		if err := m.Close(); err != nil || b.stops.Load() != 1 {
			t.Fatalf("retained failure cleanup: %v %d", err, b.stops.Load())
		}
	}
}
