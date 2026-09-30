package checkout

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func fixture(t *testing.T) (*Launcher, *[][]string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "platform with spaces")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	calls := &[][]string{}
	l := &Launcher{Root: root, Stdout: io.Discard, Stderr: io.Discard}
	l.startWeb = func(context.Context, string) (string, string, func() error, error) {
		return "http://127.0.0.1:5173", strings.Repeat("a", 64), func() error { return nil }, nil
	}
	l.execute = func(_ context.Context, name string, args []string, stdout, _ io.Writer) error {
		*calls = append(*calls, append([]string{name}, args...))
		if name == "docker" && len(args) > 0 && args[0] == "info" {
			_, err := io.WriteString(stdout, "linux\n")
			return err
		}
		if name == "docker" && len(args) > 1 && args[1] == "build" {
			for i, arg := range args {
				if arg == "--output" {
					if err := os.Mkdir(args[i+1], 0700); err != nil {
						return err
					}
					return os.WriteFile(l.binary(filepath.Dir(args[i+1])), []byte("fixture"), 0700)
				}
			}
		}
		return nil
	}
	return l, calls
}

func TestSetupActivationFailureCancellationAndArgumentForwarding(t *testing.T) {
	l, calls := fixture(t)
	if err := l.Run(context.Background(), []string{"setup"}); err != nil {
		t.Fatal(err)
	}
	active := filepath.Join(l.Root, "dist", "active")
	original, err := os.ReadFile(active)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Dir(active))
	if err != nil {
		t.Fatal(err)
	}
	normal := l.execute
	failure := errors.New("setup failed")
	l.execute = func(ctx context.Context, name string, args []string, out, diagnostic io.Writer) error {
		if name != "docker" && reflect.DeepEqual(args, []string{"setup"}) {
			return failure
		}
		return normal(ctx, name, args, out, diagnostic)
	}
	if err := l.Run(context.Background(), []string{"setup"}); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(active)
	remaining, _ := os.ReadDir(filepath.Dir(active))
	if !bytes.Equal(after, original) || len(remaining) != len(entries) {
		t.Fatal("failed setup changed active output")
	}
	ctx, cancel := context.WithCancel(context.Background())
	l.execute = func(ctx context.Context, name string, args []string, out, diagnostic io.Writer) error {
		if name != "docker" && reflect.DeepEqual(args, []string{"setup"}) {
			cancel()
			return nil
		}
		return normal(ctx, name, args, out, diagnostic)
	}
	if err := l.Run(ctx, []string{"setup"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	after, _ = os.ReadFile(active)
	if !bytes.Equal(after, original) {
		t.Fatal("canceled setup became active")
	}
	l.execute = normal
	args := []string{"exec", "example", "--", "printf", "a b"}
	if err := l.Run(context.Background(), args); err != nil {
		t.Fatal(err)
	}
	last := (*calls)[len(*calls)-1]
	if !reflect.DeepEqual(last[1:], args) {
		t.Fatalf("arguments changed: %v", last)
	}
	for _, call := range *calls {
		if call[0] == "docker" && len(call) > 2 && call[2] == "build" {
			if call[len(call)-1] != l.Root {
				t.Fatalf("unexpected context: %v", call)
			}
			for i, arg := range call {
				if arg == "--output" && !strings.HasPrefix(call[i+1], l.Root+string(filepath.Separator)) {
					t.Fatal(call)
				}
			}
		}
	}
}

func TestDevelopmentKeepsOfficialBuildAndUsesLiveCheckout(t *testing.T) {
	l, calls := fixture(t)
	if err := l.Run(context.Background(), []string{"setup"}); err != nil {
		t.Fatal(err)
	}
	active := filepath.Join(l.Root, "dist", "active")
	original, _ := os.ReadFile(active)
	repo := filepath.Join(filepath.Dir(l.Root), "challenges")
	if err := os.Mkdir(repo, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "contract.toml"), []byte("version=1"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := l.Run(context.Background(), []string{"dev"}); err != nil {
		t.Fatal(err)
	}
	last := (*calls)[len(*calls)-1]
	if !reflect.DeepEqual(last[1:], []string{"--repo", repo, "dev", "--web", "http://127.0.0.1:5173", "--nonce", strings.Repeat("a", 64)}) {
		t.Fatal(last)
	}
	after, _ := os.ReadFile(active)
	if !bytes.Equal(after, original) {
		t.Fatal("development changed the official build")
	}
	development, _ := l.active("development")
	if err := l.Run(context.Background(), []string{"--repo", repo, "list"}); err != nil {
		t.Fatal(err)
	}
	if (*calls)[len(*calls)-1][0] != development {
		t.Fatal("source command used the official build")
	}
}

func TestInvalidInputsAndGeneratedSymlinksFailBeforeExecution(t *testing.T) {
	for _, args := range [][]string{{"setup", "extra"}, {"dev", "web"}, {"dev"}} {
		l, calls := fixture(t)
		if err := l.Run(context.Background(), args); err == nil || len(*calls) != 0 {
			t.Fatalf("invalid input executed: %v %v", args, *calls)
		}
	}
	l, calls := fixture(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(l.Root, "dist")); err != nil {
		t.Fatal(err)
	}
	if err := l.Run(context.Background(), []string{"setup"}); err == nil || len(*calls) != 0 {
		t.Fatal("external dist accepted")
	}
	l, calls = fixture(t)
	if err := os.Mkdir(filepath.Join(l.Root, "dist"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(l.Root, "dist", "active"), []byte("build.fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(l.Root, "dist", "build.fixture")); err != nil {
		t.Fatal(err)
	}
	if err := l.Run(context.Background(), []string{"list"}); err == nil || len(*calls) != 0 {
		t.Fatal("external binary accepted")
	}
}

func TestMissingReferencedExecutableDoesNotSelectAnotherBuild(t *testing.T) {
	l, calls := fixture(t)
	if err := l.Run(context.Background(), []string{"setup"}); err != nil {
		t.Fatal(err)
	}
	active, err := l.active("active")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(l.Root, "dist", "development"), []byte("dev.missing\n"), 0600); err != nil {
		t.Fatal(err)
	}
	before := len(*calls)
	if err := l.Run(context.Background(), []string{"--repo", "local", "list"}); err == nil || len(*calls) != before {
		t.Fatal("missing dev executable fell back to official")
	}
	if _, err := os.Stat(active); err != nil {
		t.Fatal(err)
	}
}

type finalFailure struct{}

func (finalFailure) Write(data []byte) (int, error) {
	if bytes.Contains(data, []byte("Setup complete")) {
		return 0, io.ErrClosedPipe
	}
	return len(data), nil
}

func TestOutputFailureAfterActivationKeepsOwnedBuild(t *testing.T) {
	l, _ := fixture(t)
	l.Stdout = finalFailure{}
	if err := l.Run(context.Background(), []string{"setup"}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
	if _, err := l.active("active"); err != nil {
		t.Fatal("activated output was removed", err)
	}
}
