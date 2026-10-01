package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
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

// This opt-in check uses a copied catalog and its own cache, never the player's
// running problems. It verifies actual Docker policy and the complete web path.
func TestIsolatedNoteVaultDocker(t *testing.T) {
	source := os.Getenv("PWNDEN_TEST_CHALLENGES")
	if source == "" {
		t.Skip("set PWNDEN_TEST_CHALLENGES for actual Docker isolation and web checks")
	}
	repo := t.TempDir()
	contract, err := os.ReadFile(filepath.Join(source, "contract.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "contract.toml"), contract, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(repo, "challenges"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(filepath.Join(repo, "challenges", "note-vault"), os.DirFS(filepath.Join(source, "challenges", "note-vault"))); err != nil {
		t.Fatal(err)
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
	result, err := h.workspaces.Run(ctx, "note-vault")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Endpoints) != 1 || result.Endpoints[0].Published || !result.Endpoints[0].Proxied {
		t.Fatal("problem endpoint bypasses isolation", result)
	}
	direct, err := backend.DialEndpoint(ctx, "note-vault", "web", result.Endpoints[0].Instance)
	if err != nil {
		t.Fatal("direct Docker stream", err)
	}
	if err := direct.Close(); err != nil {
		t.Fatal(err)
	}
	browser, err := h.browsers.open(ctx, "note-vault", "web")
	if err != nil {
		t.Fatal(err)
	}
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 15 * time.Second, Transport: &http.Transport{Proxy: nil}}
	defer client.CloseIdleConnections()
	response, err := client.Get(browser.Target + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != 200 || !strings.Contains(string(body), "Note Vault") {
		t.Fatal("problem home unavailable", response.StatusCode, string(body))
	}
	response, err = client.Post(browser.Target+"/login", "application/json", strings.NewReader(`{"username":"guest","password":"guest"}`))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatal("login failed", response.StatusCode)
	}
	response, err = client.Get(browser.Target + "/api/notes/2")
	if err != nil {
		t.Fatal(err)
	}
	var note struct{ Body string }
	err = json.NewDecoder(response.Body).Decode(&note)
	response.Body.Close()
	if err != nil || response.StatusCode != 200 {
		t.Fatal("exercise behavior changed", response.StatusCode, err)
	}
	submission, err := backend.Submit(ctx, "note-vault", note.Body)
	if err != nil || !submission.Accepted {
		t.Fatal("proxy solution rejected", err)
	}
	t.Log("Web home, login cookie, vulnerable endpoint and submission passed")
	probe := `import socket, urllib.request
for host, port in [('1.1.1.1',443),('example.com',443),('2606:4700:4700::1111',443)]:
    try:
        connection = socket.create_connection((host,port),timeout=2)
    except OSError:
        print('blocked',host)
    else:
        connection.close()
        raise SystemExit('external connection succeeded: '+host)
assert urllib.request.urlopen('http://app:8000/healthz',timeout=2).status == 200
print('internal HTTP passed')
`
	c, err := challenge.Load(repo, "note-vault")
	if err != nil {
		t.Fatal(err)
	}
	host, hostPort, peer := networkControls(t, ctx, c)
	probe += fmt.Sprintf(`
for name, host, port in [('host',%q,%d),('other problem',%q,8000)]:
    try:
        connection = socket.create_connection((host,port),timeout=2)
    except OSError:
        print('blocked',name)
    else:
        connection.close()
        raise SystemExit('forbidden connection succeeded: '+name)
`, host, hostPort, peer)
	command, err := backend.Execute(ctx, "note-vault", []string{"python3", "-c", probe})
	if err != nil || command.ExitCode != 0 || !strings.Contains(command.Stdout, "internal HTTP passed") {
		t.Fatal("network probe failed", command, err)
	}
	t.Log(strings.TrimSpace(command.Stdout))
	verified, err := backend.Verify(ctx, "note-vault")
	if err != nil || !verified.Patched {
		t.Fatal("vulnerable/patched verification failed", err)
	}
	t.Log("Vulnerable solution, patch rejection and patched functional check passed")
	terminal, err := runtime.OpenTerminal(ctx, c, runtime.Project(c), 80, 24)
	if err != nil {
		t.Fatal("interactive terminal", err)
	}
	defer func() {
		if err := terminal.Close(); err != nil {
			t.Error(err)
		}
	}()
	quoted := "'" + strings.ReplaceAll(probe, "'", "'\\''") + "'"
	if _, err := io.WriteString(terminal, "stty -echo; python3 -c "+quoted+" && printf '\\137\\137PTY_ISOLATION_PASSED__\\n'\r"); err != nil {
		t.Fatal(err)
	}
	deadline := time.AfterFunc(15*time.Second, func() { _ = terminal.Close() })
	var received strings.Builder
	buffer := make([]byte, 4096)
	for !strings.Contains(received.String(), "__PTY_ISOLATION_PASSED__\r\n") {
		n, err := terminal.Read(buffer)
		received.Write(buffer[:n])
		if err != nil {
			deadline.Stop()
			t.Fatal("PTY isolation probe did not complete", err)
		}
	}
	deadline.Stop()
	t.Log("Interactive PTY internet/host/other-problem isolation passed")
	if err := terminal.Close(); err != nil {
		t.Fatal("terminal cleanup", err)
	}
	if err := h.workspaces.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := exec.CommandContext(ctx, "docker", "ps", "-aq", "--filter", "label=com.docker.compose.project="+runtime.Project(c)).CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "" {
		t.Fatal("owned containers remain", string(output), err)
	}
	output, err = exec.CommandContext(ctx, "docker", "network", "ls", "--filter", "label=com.docker.compose.project="+runtime.Project(c), "--format", "{{.ID}}").CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "" {
		t.Fatal("owned networks remain", string(output), err)
	}
	t.Log("Terminal and service/connector cleanup passed")
}

// Establish positive controls: the host listener and another problem-like HTTP
// service are reachable from their own network before the exercise probes them.
func networkControls(t *testing.T, ctx context.Context, c *challenge.Loaded) (string, int, string) {
	t.Helper()
	name := runtime.Project(c) + "-control"
	run := func(ctx context.Context, args ...string) string {
		out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
		if err != nil {
			t.Fatalf("Docker control %v: %s %v", args, string(out), err)
		}
		return strings.TrimSpace(string(out))
	}
	run(ctx, "network", "create", "--internal", name)
	t.Cleanup(func() {
		clean, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		run(clean, "network", "rm", name)
	})
	host := run(ctx, "network", "inspect", "--format", "{{(index .IPAM.Config 0).Gateway}}", name)
	listener, err := net.Listen("tcp4", net.JoinHostPort(host, "0"))
	if err != nil {
		t.Fatal("host positive control requires a locally accessible Docker bridge", err)
	}
	t.Cleanup(func() { listener.Close() })
	port := listener.Addr().(*net.TCPAddr).Port
	run(ctx, "run", "-d", "--name", name, "--network", name, "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", c.Solve.Image, "python3", "-m", "http.server", "8000")
	t.Cleanup(func() {
		clean, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		run(clean, "rm", "-f", name)
	})
	peer := run(ctx, "inspect", "--format", "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}", name)
	run(ctx, "exec", name, "python3", "-c", fmt.Sprintf(`import socket,time
socket.create_connection((%q,%d),timeout=2).close()
deadline=time.monotonic()+3
while True:
    try:
        socket.create_connection(('127.0.0.1',8000),timeout=1).close()
        break
    except OSError:
        if time.monotonic()>=deadline: raise
        time.sleep(0.05)
`, host, port))
	t.Log("Host and separate-network HTTP positive controls are reachable")
	return host, port, peer
}
