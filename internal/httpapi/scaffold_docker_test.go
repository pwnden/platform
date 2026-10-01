package httpapi

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pwnden/platform/internal/application"
	"github.com/pwnden/platform/internal/challenge"
	"github.com/pwnden/platform/internal/runtime"
)

// Point at a disposable catalog made by challenges/tools/create.py containing
// service-scaffold --kind service --category web --patched.
func TestGeneratedServiceDocker(t *testing.T) {
	repo := os.Getenv("PWNDEN_TEST_SCAFFOLD")
	if repo == "" {
		t.Skip("set PWNDEN_TEST_SCAFFOLD for generated service and patch execution")
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	backend, err := application.New(repo)
	if err != nil {
		t.Fatal(err)
	}
	h := newHandler(ctx, backend, testHost, testToken, io.Discard)
	defer func() {
		h.browsers.close()
		if err := h.workspaces.Close(); err != nil {
			t.Error("cleanup", err)
		}
	}()
	if _, err := backend.Validate(ctx, "service-scaffold"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.workspaces.Run(ctx, "service-scaffold"); err != nil {
		t.Fatal(err)
	}
	browser, err := h.browsers.open(ctx, "service-scaffold", "web")
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{Proxy: nil}}
	defer client.CloseIdleConnections()
	for _, check := range []struct {
		path string
		code int
	}{{"/healthz", 200}, {"/", 503}} {
		response, err := client.Get(browser.Target + check.path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()
		if response.StatusCode != check.code || (check.path == "/healthz" && string(body) != "ok") {
			t.Fatal("scaffold HTTP behavior", check.path, response.StatusCode, string(body))
		}
	}
	if _, err := backend.Verify(ctx, "service-scaffold"); err == nil {
		t.Fatal("unfinished solution passed verification")
	}
	c, err := challenge.Load(repo, "service-scaffold")
	if err != nil {
		t.Fatal(err)
	}
	state, err := runtime.ReadState(c)
	if err != nil {
		t.Fatal(err)
	}
	patched, err := runtime.StartPatched(ctx, c, state.Flag)
	if err != nil {
		t.Fatal("generated patch build/start", err)
	}
	// Stop owns both projects, including teardown if a following check fails.
	result, err := runtime.RunCommand(ctx, c, patched, c.Solve.Image, []string{"python3", "-c", `import urllib.request, urllib.error
assert urllib.request.urlopen('http://app:8000/healthz',timeout=2).read() == b'ok'
try:
    urllib.request.urlopen('http://app:8000/',timeout=2)
except urllib.error.HTTPError as error:
    assert error.code == 503
else:
    raise SystemExit('unfinished patch appeared complete')
print('generated patch readiness passed')`})
	if err != nil || result.ExitCode != 0 || !strings.Contains(result.Stdout, "generated patch readiness passed") {
		t.Fatal("patch readiness", result, err)
	}
	result, err = runtime.RunCommand(ctx, c, patched, c.Solve.Image, c.Patched.Check)
	if err != nil || result.ExitCode != 1 || result.Stdout != "" {
		t.Fatal("unfinished functional-check stub passed", result, err)
	}
	if err := h.workspaces.Close(); err != nil {
		t.Fatal(err)
	}
	t.Log("Generated service/patch build, local ingress, explicit unfinished stubs and cleanup passed")
}
