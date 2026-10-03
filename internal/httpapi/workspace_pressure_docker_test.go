package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pwnden/platform/internal/application"
	"github.com/pwnden/platform/internal/challenge"
	"github.com/pwnden/platform/internal/runtime"
)

// This creates disposable problems, including a CLI-owned run. A clean managed
// Docker budget is required; existing player environments are never removed.
func TestWorkspaceResourcePressureDocker(t *testing.T) {
	source := os.Getenv("PWNDEN_TEST_CHALLENGES")
	if source == "" {
		t.Skip("set PWNDEN_TEST_CHALLENGES for actual automatic pressure recovery")
	}
	out, err := exec.Command("docker", "container", "ls", "--all", "--filter", "label=pwnden.managed", "--format", "{{.ID}}").Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Skip("existing managed environments occupy the daemon; leave them intact")
	}
	original, err := challenge.Load(source, "note-vault")
	if err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	write := func(path, value string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(repo, "contract.toml"), "version=5\nsolve_network='default'\nsolve_timeout_seconds=60\nattack_rejected_exit=3\n")
	for _, slug := range []string{"outside", "old", "active", "next"} {
		dir := filepath.Join(repo, "challenges", slug)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		write(filepath.Join(dir, "README.md"), "Disposable resource-pressure fixture.\n")
		write(filepath.Join(dir, "challenge.toml"), fmt.Sprintf(`schema=5
slug=%q
title="Pressure fixture"
category="web"
difficulty=1
compose="compose.yaml"
[player]
tools=["web"]
[content]
description="README.md"
[flag]
mode="generated"
[[endpoints]]
name="web"
service="app"
port=8000
protocol="http"
[solve]
image=%q
command=["python3", "-c", "print('unused fixture solution')"]
`, slug, original.Solve.Image))
		write(filepath.Join(dir, "compose.yaml"), fmt.Sprintf(`services:
  app:
    image: %q
    command: ["python3", "-m", "http.server", "8000"]
    healthcheck:
      test: ["CMD", "python3", "-c", "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8000/',timeout=2).read()"]
      interval: 1s
      timeout: 3s
      retries: 10
`, original.Solve.Image))
	}
	t.Setenv("PWNDEN_RUNTIME_CONTAINERS", "3")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	backend, err := application.New(repo)
	if err != nil {
		t.Fatal(err)
	}
	release, err := backend.AcquireWorkspace(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if _, err := backend.Run(ctx, "outside"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := backend.Stop(context.Background(), "outside"); err != nil {
			t.Error(err)
		}
	}()
	state := func(slug string) runtime.State {
		t.Helper()
		c, err := challenge.Load(repo, slug)
		if err != nil {
			t.Fatal(err)
		}
		s, err := runtime.ReadState(c)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	outside := state("outside")
	h := newHandler(ctx, backend, testHost, testToken, io.Discard)
	server := httptest.NewServer(h)
	defer func() {
		cancel()
		server.Close()
		h.browsers.close()
		if err := h.workspaces.Close(); err != nil {
			t.Error(err)
		}
	}()
	open := func(slug string) *http.Response {
		t.Helper()
		r, err := http.NewRequestWithContext(ctx, "GET", server.URL+BasePath+"/problems/"+slug+"/workspace", nil)
		if err != nil {
			t.Fatal(err)
		}
		r.Host = testHost
		r.Header.Set("Authorization", "Bearer "+testToken)
		response, err := server.Client().Do(r)
		if err != nil {
			t.Fatal(err)
		}
		return response
	}
	ready := func(slug string) *http.Response {
		t.Helper()
		r := open(slug)
		if r.StatusCode != 200 {
			data, _ := io.ReadAll(r.Body)
			r.Body.Close()
			t.Fatalf("prepare %s: %d %s", slug, r.StatusCode, data)
		}
		line, err := bufio.NewReader(r.Body).ReadBytes('\n')
		var event struct {
			Type   string
			Status Status
		}
		if err != nil || json.Unmarshal(line, &event) != nil || event.Type != "ready" || event.Status.State != "running" {
			r.Body.Close()
			t.Fatal("invalid readiness", string(line), err)
		}
		return r
	}
	old := ready("old")
	old.Body.Close()
	deadline := time.Now().Add(time.Second)
	for {
		items := h.workspaces.List()
		if len(items) == 1 && !items[0].Connected {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("closed stream retained presence")
		}
		time.Sleep(time.Millisecond)
	}
	active := ready("active")
	defer active.Body.Close()
	activeState := state("active")
	next := ready("next") // Three containers already reserve the entire budget.
	defer next.Body.Close()
	c, err := challenge.Load(repo, "old")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.ReadState(c); !errors.Is(err, runtime.ErrNotRunning) {
		t.Fatal("idle run was not reclaimed", err)
	}
	if state("outside") != outside || state("active") != activeState {
		t.Fatal("connected or CLI-owned environment restarted")
	}
	if items := h.workspaces.List(); len(items) != 2 || !items[0].Connected || !items[1].Connected {
		t.Fatal("connected environments were not protected", items)
	}
	refused := open("old") // All remaining owned environments are connected.
	data, err := io.ReadAll(refused.Body)
	refused.Body.Close()
	var failure ErrorResponse
	if err != nil || json.Unmarshal(data, &failure) != nil || refused.StatusCode != 409 || failure.Error.Code != "resource_limit" {
		t.Fatal("protected exhaustion was not classified", refused.StatusCode, string(data), err)
	}
	if state("outside") != outside || state("active") != activeState {
		t.Fatal("exhaustion changed protected runs")
	}
	t.Log("HTTP preparation automatically reclaimed the idle run, retained connected and CLI-owned runs, and returned safe 409 only when no eligible environment remained")
}
