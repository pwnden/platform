package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pwnden/platform/internal/application"
)

func TestWorkspaceStreamAuthenticatesAndRetainsWithoutTerminal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var running atomic.Bool
	var opens atomic.Int32
	backend := fakeBackend{
		status: func(_ context.Context, slug string) (application.RunStatus, error) {
			state := "stopped"
			if running.Load() {
				state = "running"
			}
			return application.RunStatus{Slug: slug, Kind: application.KindService, State: state}, nil
		},
		run: func(_ context.Context, slug string) (application.RunInfo, error) {
			running.Store(true)
			return application.RunInfo{Slug: slug, Kind: application.KindService}, nil
		},
		terminal: func(context.Context, string, int, int) (application.TerminalSession, error) {
			opens.Add(1)
			return nil, nil
		},
	}
	h := newHandler(ctx, backend, testHost, testToken, io.Discard)
	server := httptest.NewServer(h)
	defer server.Close()
	defer h.workspaces.Close()
	request := func(ctx context.Context, token string) *http.Request {
		r, _ := http.NewRequestWithContext(ctx, "GET", server.URL+BasePath+"/problems/example/workspace", nil)
		r.Host = testHost
		r.Header.Set("Authorization", "Bearer "+token)
		return r
	}
	response, err := server.Client().Do(request(ctx, "invalid"))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 401 || running.Load() || len(h.workspaces.List()) != 0 {
		t.Fatal("unauthorized view admitted")
	}
	response, err = server.Client().Do(request(ctx, testToken))
	if err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(response.Body).ReadBytes('\n')
	if err != nil {
		t.Fatal(err)
	}
	var ready struct {
		Type   string
		Status Status
	}
	if err := json.Unmarshal(line, &ready); err != nil {
		t.Fatal(err)
	}
	if ready.Type != "ready" || ready.Status.State != "running" || opens.Load() != 0 || !h.workspaces.List()[0].Connected {
		t.Fatal("invalid shell-free view", ready)
	}
	response.Body.Close()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if items := h.workspaces.List(); len(items) == 1 && items[0].ExpiresAt != nil {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if items := h.workspaces.List(); len(items) != 1 || items[0].Connected || items[0].ExpiresAt == nil {
		t.Fatal("stream disconnect did not release presence", items)
	}
	cancel()
	h.active.Wait()
}
