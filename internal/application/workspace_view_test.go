package application

import (
	"context"
	"sync"
	"testing"
	"time"
)

type viewBackend struct {
	workspaceBackend
	running sync.Map
}

func (b *viewBackend) Status(_ context.Context, slug string) (RunStatus, error) {
	if !b.service {
		return RunStatus{Slug: slug, Kind: KindFile, State: "ready"}, nil
	}
	state := "stopped"
	if _, ok := b.running.Load(slug); ok {
		state = "running"
	}
	return RunStatus{Slug: slug, Kind: KindService, State: state}, nil
}
func (b *viewBackend) Run(_ context.Context, slug string) (RunInfo, error) {
	b.running.Store(slug, true)
	return RunInfo{Slug: slug, Kind: KindService}, nil
}

func TestWorkspaceViewsPrepareWithoutShellAndExpireAfterLastView(t *testing.T) {
	backend := &viewBackend{workspaceBackend: workspaceBackend{service: true}}
	manager := NewWorkspaces(context.Background(), backend)
	manager.idle = 25 * time.Millisecond
	defer manager.Close()
	first, err := manager.View(context.Background(), "web")
	if err != nil {
		t.Fatal(err)
	}
	second, err := manager.View(context.Background(), "web")
	if err != nil {
		t.Fatal(err)
	}
	if first.Status.State != "running" || backend.opens.Load() != 0 {
		t.Fatal("web view creates shell or fails preparation")
	}
	first.Close()
	first.Close()
	time.Sleep(2 * manager.idle)
	if len(manager.List()) != 1 || !manager.List()[0].Connected || manager.List()[0].ExpiresAt != nil {
		t.Fatal("active view expired", manager.List())
	}
	second.Close()
	if manager.List()[0].ExpiresAt == nil {
		t.Fatal("leave did not schedule cleanup")
	}
	returned, err := manager.View(context.Background(), "web")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * manager.idle)
	if !returned.Active() || backend.stops.Load() != 0 {
		t.Fatal("return did not cancel cleanup")
	}
	returned.Close()
	deadline := time.Now().Add(time.Second)
	for len(manager.List()) > 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if len(manager.List()) != 0 || backend.stops.Load() != 1 {
		t.Fatal("inactive web environment not cleaned", manager.List())
	}
}

func TestWorkspaceViewLimitAndServerCleanup(t *testing.T) {
	backend := &viewBackend{workspaceBackend: workspaceBackend{service: true}}
	manager := NewWorkspaces(context.Background(), backend)
	manager.limit = 1
	view, err := manager.View(context.Background(), "first")
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.View(context.Background(), "second")
	requireCode(t, err, WorkspaceFull)
	if err := manager.Close(); err != nil {
		t.Fatal(err)
	}
	view.Close()
	if view.Active() || len(manager.List()) != 0 || backend.stops.Load() != 1 || backend.opens.Load() != 0 {
		t.Fatal("server cleanup failed")
	}
}

func TestFileViewKeepsFinishedShellEnvironmentUntilLeave(t *testing.T) {
	backend := &viewBackend{}
	manager := NewWorkspaces(context.Background(), backend)
	defer manager.Close()
	view, err := manager.View(context.Background(), "file")
	if err != nil {
		t.Fatal(err)
	}
	if view.Status.State != "ready" || backend.opens.Load() != 0 {
		t.Fatal("file view unnecessarily starts shell")
	}
	shell, err := manager.Attach("file", 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	shell.Close()
	if err := manager.EndTerminal("file"); err != nil {
		t.Fatal(err)
	}
	if !view.Active() || len(manager.List()) != 1 || manager.List()[0].ExpiresAt != nil {
		t.Fatal("shell cleanup ended visible file workspace")
	}
	view.Close()
	if manager.List()[0].ExpiresAt == nil {
		t.Fatal("file view leave did not start timer")
	}
}
