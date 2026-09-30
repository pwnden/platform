package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pwnden/platform/internal/application"
	"github.com/pwnden/platform/internal/challenge"
	"github.com/pwnden/platform/internal/runtime"
	"github.com/pwnden/platform/internal/testutil"
)

func TestMain(m *testing.M) { testutil.Main(m) }

func fileRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "challenges", "example")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("pwnden{test}"))
	for path, content := range map[string]string{
		filepath.Join(root, "contract.toml"): "version=1\nsolve_network='default'\nsolve_timeout_seconds=1\nattack_rejected_exit=3\n",
		filepath.Join(dir, "file.txt"):       "distribution",
		filepath.Join(dir, "challenge.toml"): "schema=1\nfiles=['file.txt']\n[flag]\nmode='sha256'\nsha256='" + hex.EncodeToString(digest[:]) + "'\n[solve]\nimage='image'\ncommand=['solve']\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestVerifyCommand(t *testing.T) {
	for _, output := range []string{"pwnden{test}\n", "wrong"} {
		t.Run(output, func(t *testing.T) {
			repo := fileRepository(t)
			testutil.Docker(t,
				testutil.Reply{Match: []string{"image", "inspect"}},
				testutil.Reply{Match: []string{"run"}, Out: output},
			)
			r, err := New(DefaultCommands()...)
			if err != nil {
				t.Fatal(err)
			}
			var out, stderr bytes.Buffer
			err = r.Execute(context.Background(), []string{"--repo", repo, "verify", "example"}, &out, &stderr)
			if output == "pwnden{test}\n" {
				if err != nil || out.String() != "verified example: pwnden{test}\n" {
					t.Fatalf("verification output: %q %v", out.String(), err)
				}
			} else {
				var typed *application.Error
				if !errors.As(err, &typed) || typed.Code != application.VerificationFailed || out.Len() != 0 {
					t.Fatalf("failed verification output: %q %v", out.String(), err)
				}
			}
		})
	}
}

func TestFileCommandOutputAndTypedError(t *testing.T) {
	repo := fileRepository(t)
	r, err := New(DefaultCommands()...)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	for _, test := range []struct{ command, output string }{
		{"validate", "compatible file challenge example (contract 1; 1 files)\n"},
		{"run", "file challenge example: 1 files; run verify to check the solution\n"},
		{"stop", "stopped example\n"},
	} {
		var out, stderr bytes.Buffer
		err := r.Execute(context.Background(), []string{"--repo", repo, test.command, "example"}, &out, &stderr)
		if err != nil || out.String() != test.output || stderr.Len() != 0 {
			t.Fatalf("%s: %q %q %v", test.command, out.String(), stderr.String(), err)
		}
	}
	err = r.Execute(context.Background(), []string{"--repo", repo, "run", "missing"}, &bytes.Buffer{}, &bytes.Buffer{})
	var typed *application.Error
	if !errors.As(err, &typed) || typed.Code != application.NotFound || typed.Operation != "run" {
		t.Fatalf("application error lost: %v", err)
	}
}

type failingWriter struct{ cause error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.cause }

func TestRunOutputFailureCleansUp(t *testing.T) {
	repo := fileRepository(t)
	dir := filepath.Join(repo, "challenges", "example")
	for path, content := range map[string]string{
		filepath.Join(dir, "compose.yaml"):   "services: {}\n",
		filepath.Join(dir, "challenge.toml"): "schema=1\ncompose='compose.yaml'\n[flag]\nmode='generated'\n[solve]\nimage='image'\ncommand=['solve']\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	c, err := challenge.Load(repo, "example")
	if err != nil {
		t.Fatal(err)
	}
	testutil.Docker(t,
		testutil.Reply{Match: []string{"config"}, Out: fmt.Sprintf(`{"services":{"app":{}},"networks":{"default":{"name":%q}}}`, runtime.Project(c)+"_default")},
		testutil.Reply{Match: []string{"up"}},
		testutil.Reply{Match: []string{"down"}},
	)
	r, err := New(DefaultCommands()...)
	if err != nil {
		t.Fatal(err)
	}
	writeError := errors.New("output closed")
	err = r.Execute(context.Background(), []string{"--repo", repo, "run", "example"}, failingWriter{cause: writeError}, &bytes.Buffer{})
	if !errors.Is(err, writeError) {
		t.Fatalf("output failure lost: %v", err)
	}
	if _, err := runtime.ReadState(c); !errors.Is(err, runtime.ErrNotRunning) {
		t.Fatalf("failed command left a run: %v", err)
	}
}
