package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pwnden/platform/internal/challenge"
	"github.com/pwnden/platform/internal/runtime"
	"github.com/pwnden/platform/internal/testutil"
)

func TestMain(m *testing.M) { testutil.Main(m) }

func fixture(t *testing.T, service bool) (*Service, *challenge.Loaded) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "challenges", "example")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, "contract.toml"), "version=1\nsolve_network='default'\nsolve_timeout_seconds=1\nattack_rejected_exit=3\n")
	digest := sha256.Sum256([]byte("pwnden{test}"))
	manifest := "schema=1\nslug='example'\nfiles=['file.txt']\n"
	if service {
		manifest += "compose='compose.yaml'\n[[endpoints]]\nname='web'\nservice='app'\nport=8000\nprotocol='http'\n[flag]\nmode='generated'\n"
		write(filepath.Join(dir, "compose.yaml"), "services: {}\n")
	} else {
		manifest += "[flag]\nmode='sha256'\nsha256='" + hex.EncodeToString(digest[:]) + "'\n"
	}
	manifest += "[solve]\nimage='image'\ncommand=['solve']\n"
	write(filepath.Join(dir, "file.txt"), "distribution")
	write(filepath.Join(dir, "challenge.toml"), manifest)
	s, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	c, err := challenge.Load(root, "example")
	if err != nil {
		t.Fatal(err)
	}
	return s, c
}

func requireCode(t *testing.T, err error, code Code) *Error {
	t.Helper()
	var result *Error
	if !errors.As(err, &result) || result.Code != code {
		t.Fatalf("want %s; got %v", code, err)
	}
	return result
}

func config(c *challenge.Loaded) string {
	return fmt.Sprintf(`{"services":{"app":{"ports":[{"target":8000}]}},"networks":{"default":{"name":%q}}}`, runtime.Project(c)+"_default")
}

func TestFileOperations(t *testing.T) {
	s, c := fixture(t, false)
	t.Setenv("PATH", t.TempDir()) // File preparation and cleanup need no Docker.
	ctx := context.Background()
	v, err := s.Validate(ctx, c.Slug)
	if err != nil || v.Kind != KindFile || v.ContractVersion != 1 || v.FileCount != 1 || v.ServiceCount != 0 {
		t.Fatalf("validate: %+v %v", v, err)
	}
	r, err := s.Run(ctx, c.Slug)
	if err != nil || r.Kind != KindFile || r.Project != "" || r.FileCount != 1 {
		t.Fatalf("run: %+v %v", r, err)
	}
	if err := os.Remove(filepath.Join(c.Dir, "file.txt")); err != nil {
		t.Fatal(err)
	}
	stop, err := s.Stop(ctx, c.Slug)
	if err != nil || stop.Slug != c.Slug {
		t.Fatalf("stop with deleted distribution: %+v %v", stop, err)
	}
}

func TestLoadErrors(t *testing.T) {
	s, c := fixture(t, false)
	_, err := New("")
	requireCode(t, err, InvalidArgument)
	_, err = s.Run(context.Background(), "../outside")
	requireCode(t, err, InvalidArgument)
	if !errors.Is(err, challenge.ErrInvalidSlug) {
		t.Fatal("original slug error lost")
	}
	_, err = s.Validate(context.Background(), "missing")
	requireCode(t, err, NotFound)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatal("original filesystem error lost")
	}
	if err := os.WriteFile(filepath.Join(c.RepoRoot, "contract.toml"), []byte("version=8\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = s.Validate(context.Background(), c.Slug)
	requireCode(t, err, IncompatibleContract)
	var version *challenge.VersionError
	if !errors.As(err, &version) || version.Version != 8 {
		t.Fatal("original version error lost")
	}
}

func TestServiceRunAndStop(t *testing.T) {
	s, c := fixture(t, true)
	calls := testutil.Docker(t, testutil.WithIsolatedNetwork(runtime.Project(c), "network-id",
		testutil.Reply{Match: []string{"config"}, Out: config(c)},
		testutil.Reply{Match: []string{"up"}},
		testutil.Reply{Match: []string{"down"}},
	)...)
	ctx := context.Background()
	v, err := s.Validate(ctx, c.Slug)
	if err != nil || v.Kind != KindService || v.ServiceCount != 1 {
		t.Fatalf("validate service: %+v %v", v, err)
	}
	r, err := s.Run(ctx, c.Slug)
	if err != nil || r.Project != runtime.Project(c) || len(r.Endpoints) != 1 || r.Endpoints[0].URL != "http://app:8000" || !r.Endpoints[0].Proxied || r.Endpoints[0].Published || r.Endpoints[0].Instance == "" {
		t.Fatalf("run service: %+v %v", r, err)
	}
	_, err = s.Run(ctx, c.Slug)
	requireCode(t, err, AlreadyRunning)
	if !errors.Is(err, runtime.ErrAlreadyRunning) {
		t.Fatal("conflict cause lost")
	}
	// Explicit cleanup continues even when its caller has been canceled.
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.Stop(canceled, c.Slug); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.ReadState(c); !errors.Is(err, runtime.ErrNotRunning) {
		t.Fatalf("state remains: %v", err)
	}
	ups := 0
	for _, call := range calls() {
		for _, arg := range call {
			if arg == "create" {
				ups++
			}
		}
	}
	if ups != 1 {
		t.Fatalf("duplicate start reached Docker: %d", ups)
	}
}

func TestUnsafeNetworkRollsBackStartup(t *testing.T) {
	s, c := fixture(t, true)
	calls := testutil.Docker(t,
		testutil.Reply{Match: []string{"version"}, Out: "29.4.1"},
		testutil.Reply{Match: []string{"config"}, Out: config(c)},
		testutil.Reply{Match: []string{"up"}},
		testutil.Reply{Match: []string{"network", "ls"}, Out: "network-id"},
		testutil.Reply{Match: []string{"network", "inspect"}, Out: `[{"Name":"unsafe","Driver":"bridge","Internal":false}]`},
		testutil.Reply{Match: []string{"container", "ls", "--filter"}, Out: "service-id"},
		testutil.Reply{Match: []string{"container", "inspect", "service-id"}, Out: `[{"Mounts":[]}]`},
		testutil.Reply{Match: []string{"down"}},
	)
	r, err := s.Run(context.Background(), c.Slug)
	requireCode(t, err, ExecutionFailed)
	if r.Slug != "" {
		t.Fatal("partial run result exposed")
	}
	inspected, stopped := false, false
	for _, call := range calls() {
		inspected = inspected || strings.Contains(strings.Join(call, " "), "network inspect")
		for _, arg := range call {
			stopped = stopped || arg == "down"
		}
	}
	if !inspected || !stopped {
		t.Fatal("unsafe live network did not reach inspection and cleanup", calls())
	}
	if _, err := runtime.ReadState(c); !errors.Is(err, runtime.ErrNotRunning) {
		t.Fatalf("failed run not rolled back: %v", err)
	}
}

func TestCleanupFailurePreservesState(t *testing.T) {
	s, c := fixture(t, true)
	testutil.Docker(t,
		testutil.Reply{Match: []string{"config"}, Out: config(c)},
		testutil.Reply{Match: []string{"down"}, Err: "cleanup blocked", Code: 1},
	)
	if err := runtime.SaveState(c, runtime.State{Project: runtime.Project(c), Flag: "pwnden{test}"}); err != nil {
		t.Fatal(err)
	}
	_, err := s.Stop(context.Background(), c.Slug)
	requireCode(t, err, CleanupFailed)
	if !errors.Is(err, runtime.ErrCleanupFailed) || !strings.Contains(err.Error(), "cleanup blocked") {
		t.Fatal("cleanup cause lost")
	}
	if _, err := runtime.ReadState(c); err != nil {
		t.Fatalf("state lost: %v", err)
	}
}

func TestVerification(t *testing.T) {
	for _, service := range []bool{false, true} {
		t.Run(fmt.Sprintf("service=%t", service), func(t *testing.T) {
			s, c := fixture(t, service)
			testutil.Docker(t, testutil.WithIsolatedNetwork(runtime.Project(c), "network-id",
				testutil.Reply{Match: []string{"config"}, Out: config(c)},
				testutil.Reply{Match: []string{"image", "inspect"}},
				testutil.Reply{Match: []string{"network", "ls"}, Out: "network-id"},
				testutil.Reply{Match: []string{"run"}, Out: "pwnden{test}\n"},
			)...)
			if service {
				_, err := s.Verify(context.Background(), c.Slug)
				requireCode(t, err, NotRunning)
				if err := runtime.SaveState(c, runtime.State{Project: runtime.Project(c), Flag: "pwnden{test}"}); err != nil {
					t.Fatal(err)
				}
			}
			result, err := s.Verify(context.Background(), c.Slug)
			if err != nil || result.Slug != c.Slug || result.Flag != "pwnden{test}" || result.Patched {
				t.Fatalf("verify: %+v %v", result, err)
			}
		})
	}
}

func TestVerificationFailureAndTimeout(t *testing.T) {
	for _, test := range []struct {
		name  string
		out   string
		delay int
		code  Code
	}{
		{"wrong flag", "wrong", 0, VerificationFailed},
		{"toolbox deadline", "pwnden{test}", 1500, DeadlineExceeded},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, c := fixture(t, false)
			testutil.Docker(t,
				testutil.Reply{Match: []string{"image", "inspect"}},
				testutil.Reply{Match: []string{"run"}, Out: test.out, DelayMS: test.delay},
				testutil.Reply{Match: []string{"rm"}},
			)
			result, err := s.Verify(context.Background(), c.Slug)
			requireCode(t, err, test.code)
			if result.Flag != "" {
				t.Fatal("failed verification exposed a partial flag")
			}
			if test.code == DeadlineExceeded && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal("inner timeout cause lost")
			}
		})
	}
}

func TestCanceledCallsAndErrorPrecedence(t *testing.T) {
	s, c := fixture(t, false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := s.Run(ctx, c.Slug)
	requireCode(t, err, Canceled)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("cancel cause lost")
	}
	ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	_, err = s.Validate(ctx, c.Slug)
	requireCode(t, err, DeadlineExceeded)
	err = operationError(ctx, "run", c.Slug, ExecutionFailed, runtime.ErrCleanupFailed)
	requireCode(t, err, CleanupFailed)
	if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, runtime.ErrCleanupFailed) {
		t.Fatal("joined failure causes lost")
	}
}

func TestCanceledStartupCleansUp(t *testing.T) {
	s, c := fixture(t, true)
	calls := testutil.Docker(t,
		testutil.Reply{Match: []string{"version"}, Out: "29.4.1"},
		testutil.Reply{Match: []string{"config"}, Out: config(c)},
		testutil.Reply{Match: []string{"up"}, DelayMS: 1500},
		testutil.Reply{Match: []string{"down"}},
	)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := s.Run(ctx, c.Slug)
	requireCode(t, err, DeadlineExceeded)
	down := false
	for _, call := range calls() {
		for _, arg := range call {
			if arg == "down" {
				down = true
			}
		}
	}
	if !down {
		t.Fatal("canceled startup skipped independent cleanup")
	}
	if _, err := runtime.ReadState(c); !errors.Is(err, runtime.ErrNotRunning) {
		t.Fatalf("canceled start retained state: %v", err)
	}
}
