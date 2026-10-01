package application

import (
	"context"
	"errors"
	"sync"
)

// WorkspaceView owns problem presence independently of an interactive shell.
type WorkspaceView struct {
	Status  RunStatus
	manager *Workspaces
	entry   *workspace
	once    sync.Once
}

func (m *Workspaces) View(ctx context.Context, slug string) (*WorkspaceView, error) {
	e, err := m.reserve(slug)
	if err != nil {
		return nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if m.entry(slug) != e {
		return nil, workspaceError(slug, Canceled, context.Canceled)
	}
	status, err := m.backend.Status(ctx, slug)
	if err == nil && status.Kind == KindService {
		e.service = true
		if status.State == "stopped" {
			_, err = m.backend.Run(ctx, slug)
			if err == nil {
				status, err = m.backend.Status(ctx, slug)
			}
		}
	}
	if err == nil && status.State != "running" && status.State != "ready" {
		err = workspaceError(slug, NotRunning, errors.New("problem environment unavailable"))
	}
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		if e.service || e.terminal != nil || e.viewers > 0 {
			m.idleLocked(e)
		} else {
			err = errors.Join(err, m.remove(e))
		}
		return nil, err
	}
	e.viewers++
	if e.timer != nil {
		e.timer.Stop()
	}
	e.expires = nil
	return &WorkspaceView{Status: status, manager: m, entry: e}, nil
}

func (v *WorkspaceView) Active() bool { return v.manager.entry(v.entry.slug) == v.entry }

func (v *WorkspaceView) Close() {
	v.once.Do(func() {
		e := v.entry
		e.mu.Lock()
		defer e.mu.Unlock()
		if v.Active() {
			e.viewers--
			v.manager.idleLocked(e)
		}
	})
}
