package application

import (
	"context"
	"errors"
	"io"
	"sort"
	"sync"
	"time"
)

const WorkspaceLimit = 10
const WorkspaceIdle = 10 * time.Minute

type WorkspaceBackend interface {
	Observer
	Runner
	Terminals
}
type WorkspaceInfo struct {
	Slug      string     `json:"slug"`
	Connected bool       `json:"connected"`
	ExpiresAt *time.Time `json:"expires_at"`
}
type Workspaces struct {
	mu       sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
	backend  WorkspaceBackend
	entries  map[string]*workspace
	idle     time.Duration
	limit    int
	closed   bool
	failures error
}
type workspace struct {
	mu       sync.Mutex
	slug     string
	service  bool
	terminal *terminalProcess
	attached *TerminalAttachment
	viewers  int
	timer    *time.Timer
	expires  *time.Time
	cancel   context.CancelFunc
}

func NewWorkspaces(ctx context.Context, backend WorkspaceBackend) *Workspaces {
	base, cancel := context.WithCancel(ctx)
	return &Workspaces{ctx: base, cancel: cancel, backend: backend, entries: map[string]*workspace{}, idle: WorkspaceIdle, limit: WorkspaceLimit}
}
func workspaceError(slug string, code Code, cause error) error {
	return &Error{Code: code, Operation: "workspace", Slug: slug, Cause: cause}
}
func (m *Workspaces) reserve(slug string) (*workspace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.ctx.Err() != nil {
		return nil, workspaceError(slug, Canceled, context.Canceled)
	}
	if e := m.entries[slug]; e != nil {
		return e, nil
	}
	if len(m.entries) >= m.limit {
		return nil, workspaceError(slug, WorkspaceFull, errors.New("retained environment limit reached"))
	}
	e := &workspace{slug: slug}
	if journal, ok := m.backend.(WorkspaceJournal); ok {
		if err := journal.TrackWorkspace(slug); err != nil {
			return nil, err
		}
	}
	m.entries[slug] = e
	return e, nil
}
func (m *Workspaces) entry(slug string) *workspace {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.entries[slug]
}
func (m *Workspaces) remove(e *workspace) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.entries[e.slug] == e {
		if journal, ok := m.backend.(WorkspaceJournal); ok {
			if err := journal.ForgetWorkspace(e.slug); err != nil {
				m.failures = errors.Join(m.failures, err)
				return workspaceError(e.slug, CleanupFailed, err)
			}
		}
		delete(m.entries, e.slug)
	}
	return nil
}
func (m *Workspaces) failure(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failures = errors.Join(m.failures, err)
}
func (m *Workspaces) idleLocked(e *workspace) {
	if e.timer != nil {
		e.timer.Stop()
	}
	if e.viewers > 0 || e.attached != nil {
		e.expires = nil
		return
	}
	at := time.Now().Add(m.idle)
	e.expires = &at
	e.timer = time.AfterFunc(m.idle, func() {
		e.mu.Lock()
		defer e.mu.Unlock()
		if m.entry(e.slug) == e && e.attached == nil && e.viewers == 0 && e.expires == &at {
			m.stopLocked(e)
		}
	})
}
func (m *Workspaces) stopLocked(e *workspace) error {
	if e.timer != nil {
		e.timer.Stop()
	}
	var err error
	if e.terminal != nil {
		err = e.terminal.stop()
	}
	if e.cancel != nil {
		e.cancel()
	}
	if cleaner, ok := m.backend.(interface {
		CleanupWorkspace(context.Context, string) error
	}); ok {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err = errors.Join(err, cleaner.CleanupWorkspace(ctx, e.slug))
		cancel()
	} else if e.service {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		_, failure := m.backend.Stop(ctx, e.slug)
		cancel()
		err = errors.Join(err, failure)
	}
	if err != nil {
		m.failure(err)
		e.expires = nil
		return workspaceError(e.slug, CleanupFailed, err)
	}
	return m.remove(e)
}
func (m *Workspaces) List() []WorkspaceInfo {
	m.mu.Lock()
	entries := make([]*workspace, 0, len(m.entries))
	for _, e := range m.entries {
		entries = append(entries, e)
	}
	m.mu.Unlock()
	result := make([]WorkspaceInfo, 0, len(entries))
	for _, e := range entries {
		e.mu.Lock()
		if m.entry(e.slug) == e {
			result = append(result, WorkspaceInfo{e.slug, e.attached != nil || e.viewers > 0, e.expires})
		}
		e.mu.Unlock()
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Slug < result[j].Slug })
	return result
}
func (m *Workspaces) Run(ctx context.Context, slug string) (RunInfo, error) {
	e, err := m.reserve(slug)
	if err != nil {
		return RunInfo{}, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if m.entry(slug) != e {
		return RunInfo{}, workspaceError(slug, Canceled, context.Canceled)
	}
	result, err := m.backend.Run(ctx, slug)
	if err != nil {
		var operation *Error
		if errors.As(err, &operation) && (operation.Code == CleanupFailed || operation.Code == AlreadyRunning) {
			e.service = true
			if e.attached == nil {
				m.idleLocked(e)
			}
		} else if e.terminal == nil && !e.service {
			if cleanup := m.remove(e); cleanup != nil {
				return result, cleanup
			}
		}
		return result, err
	}
	e.service = result.Kind == KindService
	if e.attached == nil {
		m.idleLocked(e)
	}
	return result, nil
}
func (m *Workspaces) Stop(slug string) error {
	for e := m.entry(slug); e != nil; e = m.entry(slug) {
		e.mu.Lock()
		if m.entry(slug) != e {
			e.mu.Unlock()
			continue
		}
		err := m.stopLocked(e)
		e.mu.Unlock()
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := m.backend.Stop(ctx, slug)
	return err
}
func (m *Workspaces) EndTerminal(slug string) error {
	e := m.entry(slug)
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if m.entry(slug) != e {
		return nil
	}
	if e.terminal == nil {
		return nil
	}
	if err := e.terminal.stop(); err != nil {
		m.failure(err)
		return workspaceError(slug, CleanupFailed, err)
	}
	e.terminal = nil
	if e.cancel != nil {
		e.cancel()
		e.cancel = nil
	}
	if !e.service && e.viewers == 0 {
		return m.remove(e)
	}
	return nil
}
func (m *Workspaces) Attach(slug string, cols, rows int) (*TerminalAttachment, error) {
	e, err := m.reserve(slug)
	if err != nil {
		return nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if m.entry(slug) != e || m.ctx.Err() != nil {
		return nil, workspaceError(slug, Canceled, context.Canceled)
	}
	if e.attached != nil {
		return nil, workspaceError(slug, TerminalBusy, errors.New("problem already attached"))
	}
	if e.terminal != nil && e.terminal.finished() {
		if failure := e.terminal.stop(); failure != nil {
			return nil, workspaceError(slug, CleanupFailed, failure)
		}
		e.terminal = nil
		if e.cancel != nil {
			e.cancel()
			e.cancel = nil
		}
	}
	reused := e.terminal != nil
	if e.terminal == nil {
		ctx, cancel := context.WithCancel(m.ctx)
		deadline := time.AfterFunc(15*time.Minute, cancel)
		status, failure := m.backend.Status(ctx, slug)
		if failure == nil && status.Kind == KindService {
			e.service = true
			if status.State == "stopped" {
				_, failure = m.backend.Run(ctx, slug)
			}
		}
		var shell TerminalSession
		if failure == nil {
			shell, failure = m.backend.OpenTerminal(ctx, slug, cols, rows)
		}
		deadline.Stop()
		if failure != nil {
			cancel()
			var operation *Error
			if e.service || (errors.As(failure, &operation) && operation.Code == CleanupFailed) {
				m.idleLocked(e)
			} else {
				if cleanup := m.remove(e); cleanup != nil {
					return nil, cleanup
				}
			}
			return nil, failure
		}
		e.cancel = cancel
		e.terminal = newTerminalProcess(shell, cols, rows)
		process := e.terminal
		go func() {
			<-process.done
			e.mu.Lock()
			defer e.mu.Unlock()
			if m.entry(slug) == e && e.terminal == process && !e.service && e.viewers == 0 {
				if err := process.stop(); err != nil {
					m.failure(err)
				} else {
					m.remove(e)
				}
			}
		}()
	}
	a, err := e.terminal.attach(cols, rows)
	if err != nil {
		m.idleLocked(e)
		return nil, err
	}
	a.Reused = reused
	if e.timer != nil {
		e.timer.Stop()
	}
	e.expires = nil
	e.attached = a
	a.release = func() {
		e.mu.Lock()
		defer e.mu.Unlock()
		if m.entry(slug) == e && e.attached == a {
			a.process.detach(a)
			e.attached = nil
			if a.process.finished() && !e.service && e.viewers == 0 {
				if err := a.process.stop(); err != nil {
					m.failure(err)
				} else {
					m.remove(e)
				}
			} else {
				m.idleLocked(e)
			}
		}
	}
	return a, nil
}
func (m *Workspaces) Close() error {
	m.mu.Lock()
	m.closed = true
	m.cancel()
	entries := make([]*workspace, 0, len(m.entries))
	for _, e := range m.entries {
		entries = append(entries, e)
	}
	m.mu.Unlock()
	for _, e := range entries {
		e.mu.Lock()
		m.stopLocked(e)
		e.mu.Unlock()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.failures
}

type TerminalAttachment struct {
	process  *terminalProcess
	frames   chan []byte
	notify   chan struct{}
	release  func()
	once     sync.Once
	Snapshot []byte
	Reused   bool
}

func (a *TerminalAttachment) Close() {
	a.once.Do(func() {
		if a.release != nil {
			a.release()
		}
	})
}
func (a *TerminalAttachment) Input(data []byte) error { return a.process.input(data) }
func (a *TerminalAttachment) Resize(ctx context.Context, cols, rows int) error {
	return a.process.resize(ctx, cols, rows)
}
func (a *TerminalAttachment) Next(ctx context.Context) ([]byte, error) {
	select {
	case data := <-a.frames:
		return data, nil
	default:
	}
	select {
	case data := <-a.frames:
		return data, nil
	case <-a.notify:
		select {
		case data := <-a.frames:
			return data, nil
		default:
		}
		return nil, io.EOF
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func (a *TerminalAttachment) Result() (int, error) {
	a.process.mu.Lock()
	defer a.process.mu.Unlock()
	if !a.process.ended {
		return 0, errors.New("terminal output backlog exceeded")
	}
	return a.process.code, a.process.err
}
