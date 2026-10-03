package application

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/pwnden/platform/internal/runtime"
)

// Reservation pins an entry before waiting for its lock, protecting reconnects
// from pressure cleanup without serializing independent problem preparation.
func (m *Workspaces) finishPreparation(e *workspace) {
	m.mu.Lock()
	e.preparing--
	m.mu.Unlock()
}
func (m *Workspaces) idleEntry(e *workspace) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.entries[e.slug] == e && e.preparing == 0
}

type workspaceReclaim struct{ done <-chan struct{} }

func (*workspaceReclaim) Error() string { return "workspace cleanup in progress" }

// Claim under the same mutex as reservation: a reconnect either pins the old
// environment first or waits for cleanup and prepares a new one afterward.
func (m *Workspaces) claimIdle(e *workspace) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.entries[e.slug] != e || e.preparing != 0 || e.reclaiming != nil {
		return false
	}
	e.reclaiming = make(chan struct{})
	return true
}
func (m *Workspaces) finishReclaim(e *workspace) {
	m.mu.Lock()
	close(e.reclaiming)
	e.reclaiming = nil
	m.mu.Unlock()
}

func (m *Workspaces) startService(ctx context.Context, e *workspace) error {
	err := m.retryResources(ctx, e, func() error {
		_, failure := m.backend.Run(ctx, e.slug)
		return failure
	})
	// Startup rolls back on failure. Keep ownership only when cleanup failed or
	// a previously running service was observed; a budget refusal owns no run.
	var failure *Error
	e.service = err == nil || (errors.As(err, &failure) && (failure.Code == CleanupFailed || failure.Code == AlreadyRunning))
	return err
}

// The caller holds the current entry lock. Each retry follows a successful
// removal. Bound attempts even when concurrent requests keep adding idle runs.
func (m *Workspaces) retryResources(ctx context.Context, current *workspace, attempt func() error) error {
	for attempts := 0; ; attempts++ {
		if err := errors.Join(ctx.Err(), m.ctx.Err()); err != nil {
			return operationError(ctx, "workspace", current.slug, Canceled, err)
		}
		err := attempt()
		if err == nil {
			return nil
		}
		var failure *Error
		if !errors.Is(err, runtime.ErrResourceLimit) ||
			(errors.As(err, &failure) && failure.Code != ResourceLimit) ||
			errors.Is(err, runtime.ErrCleanupFailed) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if attempts >= m.limit {
			return operationError(ctx, "workspace", current.slug, ResourceLimit, err)
		}
		reclaimed, cleanup := m.reclaimIdle(ctx, current)
		if cleanup != nil {
			return cleanup
		}
		if !reclaimed {
			return operationError(ctx, "workspace", current.slug, ResourceLimit, err)
		}
	}
}

// Only disconnected entries owned by this manager are eligible. Never infer
// ownership from Docker labels: CLI runs and other installations share budgets.
func (m *Workspaces) reclaimIdle(ctx context.Context, current *workspace) (bool, error) {
	m.mu.Lock()
	entries := make([]*workspace, 0, len(m.entries))
	for _, e := range m.entries {
		if e != current {
			entries = append(entries, e)
		}
	}
	m.mu.Unlock()
	type candidate struct {
		entry *workspace
		at    time.Time
	}
	var candidates []candidate
	for _, e := range entries {
		// Never wait on a second entry while holding the current one: parallel
		// admissions or cleanup timers could otherwise deadlock.
		if !e.mu.TryLock() {
			continue
		}
		if m.idleEntry(e) && e.attached == nil && e.viewers == 0 && e.expires != nil {
			candidates = append(candidates, candidate{e, *e.expires})
		}
		e.mu.Unlock()
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].at.Before(candidates[j].at) })
	for _, c := range candidates {
		e := c.entry
		if !e.mu.TryLock() {
			continue
		}
		if err := errors.Join(ctx.Err(), m.ctx.Err()); err != nil {
			e.mu.Unlock()
			return false, operationError(ctx, "workspace", "", Canceled, err)
		}
		if e.attached != nil || e.viewers != 0 || e.expires == nil || !m.claimIdle(e) {
			e.mu.Unlock()
			continue
		}
		err := m.stopLocked(e)
		m.finishReclaim(e)
		e.mu.Unlock()
		return err == nil, err
	}
	return false, nil
}
