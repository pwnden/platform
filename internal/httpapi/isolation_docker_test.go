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
	"regexp"
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
	metadata, err := challenge.Load(repo, "note-vault")
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Schema >= 7 {
		if err := os.CopyFS(filepath.Join(repo, "knowledge"), os.DirFS(filepath.Join(source, "knowledge"))); err != nil {
			t.Fatal("copy shared learning resources", err)
		}
	}
	_, frontendErr := os.Stat(filepath.Join(source, "web", "package.json"))
	vueTarget := frontendErr == nil
	if frontendErr != nil && !os.IsNotExist(frontendErr) {
		t.Fatal(frontendErr)
	}
	if vueTarget {
		if err := os.Mkdir(filepath.Join(repo, "web"), 0755); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"package.json", "pnpm-lock.yaml", "vite.config.ts", "tsconfig.json", "index.html", "server.py", "__init__.py"} {
			data, err := os.ReadFile(filepath.Join(source, "web", name))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(repo, "web", name), data, 0644); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.CopyFS(filepath.Join(repo, "web", "src"), os.DirFS(filepath.Join(source, "web", "src"))); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	backend, err := application.New(repo)
	if err != nil {
		t.Fatal(err)
	}
	detail, err := backend.Detail(ctx, "note-vault")
	if err != nil || len(detail.Tools) != 1 || detail.Tools[0] != "web" {
		t.Fatal("web-only tool declaration", detail.Tools, err)
	}
	if metadata.Schema >= 7 {
		if detail.Learning == nil {
			t.Fatal("copied catalog lost learning metadata")
		}
		for _, concept := range append(append([]challenge.Concept{}, detail.Learning.Requires...), detail.Learning.Teaches...) {
			content, err := backend.Guidance(ctx, "note-vault", "concept-"+concept.ID)
			if err != nil || strings.TrimSpace(content) == "" {
				t.Fatal("copied concept document", concept.ID, err)
			}
		}
	}
	h := newHandler(ctx, backend, testHost, testToken, io.Discard)
	defer func() {
		h.browsers.close()
		if err := h.workspaces.Close(); err != nil {
			t.Error("cleanup", err)
		}
	}()
	view, err := h.workspaces.View(ctx, "note-vault")
	if err != nil {
		t.Fatal(err)
	}
	defer view.Close()
	result := view.Status
	if result.State != "running" || !h.workspaces.List()[0].Connected || h.workspaces.List()[0].ExpiresAt != nil {
		t.Fatal("web-only problem presence failed")
	}
	if e := h.workspaces.List(); len(e) != 1 {
		t.Fatal("workspace count", e)
	}
	t.Log("Web-only environment prepared and retained without terminal attachment")
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
	if response.StatusCode != 200 || !strings.Contains(string(body), "개인 메모 사이트") {
		t.Fatal("problem home unavailable", response.StatusCode, string(body))
	}
	request, err := http.NewRequestWithContext(ctx, "GET", browser.Target+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Accept", "text/html")
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != 200 || (vueTarget && !strings.Contains(string(body), `type="module"`)) {
		t.Fatal("Vue document unavailable", response.StatusCode, string(body))
	}
	if vueTarget {
		assets := regexp.MustCompile(`(?:src|href)="(/assets/[^" ]+)"`).FindAllStringSubmatch(string(body), -1)
		if len(assets) < 2 {
			t.Fatal("Vue script and style declarations missing")
		}
		for _, asset := range assets {
			r, err := client.Get(browser.Target + asset[1])
			if err != nil {
				t.Fatal(err)
			}
			payload, _ := io.ReadAll(r.Body)
			r.Body.Close()
			if r.StatusCode != 200 || len(payload) == 0 {
				t.Fatal("Vue asset unavailable", asset[1], r.StatusCode)
			}
			if strings.HasSuffix(asset[1], ".js") && !strings.HasPrefix(r.Header.Get("Content-Type"), "text/javascript") {
				t.Fatal("Vue module MIME", r.Header)
			}
		}
	}
	// Compare identical warm routes with live endpoint inspection versus the full
	// environment observer. Both modes keep identity and isolation enforcement.
	observer := h.browsers.observe
	if err := observer.CheckEndpoint(ctx, "note-vault", "web", result.Endpoints[0].Instance, "http://app:9000"); err == nil {
		t.Fatal("changed destination accepted")
	}
	for _, mode := range []string{"full status", "live endpoint"} {
		if mode == "full status" {
			h.browsers.observe = nil
		} else {
			h.browsers.observe = observer
		}
		started := time.Now()
		for i := 0; i < 5; i++ {
			r, err := client.Get(browser.Target + "/")
			if err != nil {
				t.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, r.Body)
			r.Body.Close()
			if r.StatusCode != 200 {
				t.Fatal(r.Status)
			}
		}
		t.Logf("%s: mean route time %s (5 requests)", mode, time.Since(started)/5)
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
	cpuQuota := "200000"
	if os.Getenv("PWNDEN_CONTAINER_CPUS") == "1" {
		cpuQuota = "100000"
	}
	probe := fmt.Sprintf(`import errno, os, pathlib, socket, urllib.request
assert os.geteuid() == 10001, 'toolbox must use a non-root UID'
assert pathlib.Path('/sys/fs/cgroup/cpu.max').read_text().split() == ['%s', '100000']
assert pathlib.Path('/sys/fs/cgroup/memory.max').read_text().strip() == str(2*1024**3)
assert pathlib.Path('/sys/fs/cgroup/memory.swap.max').read_text().strip() == '0'
assert pathlib.Path('/sys/fs/cgroup/pids.max').read_text().strip() == '256'
assert os.statvfs('/tmp').f_blocks * os.statvfs('/tmp').f_frsize == 128*1024**2
pathlib.Path('/tmp/pwnden-resource-check').write_text('writable')
for path in ['/var/tmp/pwnden-forbidden', '/workspace/pwnden-forbidden']:
    try:
        pathlib.Path(path).write_text('forbidden')
    except OSError as error:
        assert error.errno in (errno.EROFS, errno.EACCES)
    else:
        raise SystemExit('read-only write succeeded: '+path)
print('non-root, cgroup limits, bounded tmpfs and read-only root passed')
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
`, cpuQuota)
	c, err := challenge.Load(repo, "note-vault")
	if err != nil {
		t.Fatal(err)
	}
	terminals, err := exec.CommandContext(ctx, "docker", "ps", "--all", "--filter", "label=pwnden.kind=terminal", "--filter", "label=pwnden.project="+runtime.Project(c), "--format", "{{.ID}}").Output()
	if err != nil || strings.TrimSpace(string(terminals)) != "" {
		t.Fatal("web-only solve created a terminal", err)
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
