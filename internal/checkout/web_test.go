package checkout

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWebMountsOnlyLiveSourcesAndCleansOwnedVolumes(t *testing.T) {
	l, _ := fixture(t)
	root := filepath.Join(l.Root, "web")
	for _, name := range []string{"package.json", "pnpm-lock.yaml", "pnpm-workspace.yaml", "patches/vue-router@5.3.1.patch", "patches/README.md", "apps/player/package.json", "apps/player/src/App.vue", "node_modules/hidden/package.json"} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("fixture"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/@vite/client" {
			t.Errorf("unexpected readiness path: %s", r.URL.Path)
		}
		w.WriteHeader(200)
	}))
	defer upstream.Close()
	u, _ := url.Parse(upstream.URL)
	var calls [][]string
	l.execute = func(_ context.Context, _ string, args []string, stdout, _ io.Writer) error {
		calls = append(calls, args)
		if args[0] == "port" {
			_, err := io.WriteString(stdout, u.Host+"\n")
			return err
		}
		return nil
	}
	directory := filepath.Join(l.Root, "dist", "dev.fixture")
	target, nonce, stop, err := l.web(context.Background(), directory)
	if err != nil || target != upstream.URL || len(nonce) != 64 {
		t.Fatalf("frontend startup: %s %v", target, err)
	}
	if content, err := os.ReadFile(filepath.Join(directory, "frontend", "patches/vue-router@5.3.1.patch")); err != nil || string(content) != "fixture" {
		t.Fatalf("dependency patch missing from development image context: %q %v", content, err)
	}
	for _, name := range []string{"apps/player/src/App.vue", "node_modules/hidden/package.json", "patches/README.md"} {
		if _, err := os.Stat(filepath.Join(directory, "frontend", name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("source/dependency copied into tool image: %s", name)
		}
	}
	run := strings.Join(calls[1], " ")
	for _, required := range []string{"127.0.0.1::5173", "source=" + root, "readonly", "PWNDEN_READONLY_SOURCES=1", "/web/node_modules", "/web/apps/player/node_modules", "--cap-drop ALL", "no-new-privileges"} {
		if !strings.Contains(run, required) {
			t.Fatalf("missing container boundary %q: %s", required, run)
		}
	}
	if err := stop(); err != nil {
		t.Fatal(err)
	}
	if last := strings.Join(calls[len(calls)-1], " "); last != "rm --force --volumes pwnden-web-dev.fixture" {
		t.Fatal(last)
	}
	// Failure after container creation still removes the exact owned resources.
	l.execute = func(_ context.Context, _ string, args []string, _, _ io.Writer) error {
		calls = append(calls, args)
		if args[0] == "port" {
			return io.ErrClosedPipe
		}
		return nil
	}
	if _, _, _, err := l.web(context.Background(), directory); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
	if last := strings.Join(calls[len(calls)-1], " "); last != "rm --force --volumes pwnden-web-dev.fixture" {
		t.Fatal("failed startup leaked resources", last)
	}
}

func TestWebManifestLinksStayInsideCheckout(t *testing.T) {
	for _, name := range []string{"package.json", "patches/external.patch"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			outside := filepath.Join(t.TempDir(), "package.json")
			if err := os.WriteFile(outside, []byte("private"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, filepath.Join(root, name)); err != nil {
				t.Fatal(err)
			}
			if err := copyWebManifests(root, t.TempDir()); err == nil {
				t.Fatal("external manifest accepted")
			}
		})
	}
}
