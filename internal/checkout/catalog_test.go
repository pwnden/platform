package checkout

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const originalCatalog = "408e8655de9939729c85f39c3d264d1327271864\n"
const nextCatalog = "584915e0f96ae4263885806af9845e159ecb1bab\n"

func catalogFixture(t *testing.T) (*Launcher, *[][]string) {
	t.Helper()
	l, calls := fixture(t)
	if err := os.WriteFile(filepath.Join(l.Root, "catalog.lock"), []byte(originalCatalog), 0644); err != nil {
		t.Fatal(err)
	}
	normal := l.execute
	l.execute = func(ctx context.Context, name string, args []string, out, diagnostic io.Writer) error {
		if slices.Contains(args, "catalog-export") {
			*calls = append(*calls, append([]string{name}, args...))
			output := args[slices.Index(args, "--output")+1]
			if err := os.Mkdir(output, 0700); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(output, "catalog.lock"), []byte(nextCatalog), 0644)
		}
		return normal(ctx, name, args, out, diagnostic)
	}
	return l, calls
}

func TestCatalogUpdatePinsPublishedCommitWithoutHostGitOrActivation(t *testing.T) {
	for _, revision := range []string{"", strings.TrimSpace(nextCatalog)} {
		t.Run(revision, func(t *testing.T) {
			l, calls := catalogFixture(t)
			out := &bytes.Buffer{}
			l.Stdout = out
			args := []string{"catalog", "update"}
			ref := "refs/heads/main"
			if revision != "" {
				args = append(args, "--revision", revision)
				ref = revision
			}
			if err := l.Run(context.Background(), args); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(l.Root, "catalog.lock")
			content, err := os.ReadFile(path)
			if err != nil || string(content) != nextCatalog || !strings.Contains(out.String(), "Catalog pinned") {
				t.Fatal(string(content), err, out.String())
			}
			for _, call := range *calls {
				if call[0] != "docker" {
					t.Fatal("host tool invoked", call)
				}
			}
			build := (*calls)[len(*calls)-1]
			if !slices.Contains(build, "--no-cache") || !slices.Contains(build, "PWNDEN_CATALOG_REF="+ref) || slices.Contains(build, "--mount") || build[len(build)-1] != l.Root {
				t.Fatal(build)
			}
			entries, _ := os.ReadDir(filepath.Join(l.Root, "dist"))
			if len(entries) != 0 {
				t.Fatal("catalog update left outputs or activated a build", entries)
			}
			before, _ := os.Stat(path)
			out.Reset()
			if err := l.Run(context.Background(), args); err != nil {
				t.Fatal(err)
			}
			after, _ := os.Stat(path)
			if !os.SameFile(before, after) || !strings.Contains(out.String(), "already pinned") {
				t.Fatal("no-op replaced the lock")
			}
		})
	}
}

func TestCatalogUpdateFailurePreservesLockAndCleansOutput(t *testing.T) {
	for _, failure := range []string{"acquisition", "cancel", "malformed", "mismatch", "manual edit"} {
		t.Run(failure, func(t *testing.T) {
			l, _ := catalogFixture(t)
			normal := l.execute
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			path := filepath.Join(l.Root, "catalog.lock")
			want := originalCatalog
			l.execute = func(ctx context.Context, name string, args []string, out, diagnostic io.Writer) error {
				if !slices.Contains(args, "catalog-export") {
					return normal(ctx, name, args, out, diagnostic)
				}
				if failure == "acquisition" {
					return errors.New("commit unavailable")
				}
				if err := normal(ctx, name, args, out, diagnostic); err != nil {
					return err
				}
				switch failure {
				case "cancel":
					cancel()
				case "malformed":
					output := args[slices.Index(args, "--output")+1]
					return os.WriteFile(filepath.Join(output, "catalog.lock"), []byte("unexpected"), 0600)
				case "manual edit":
					want = strings.Repeat("b", 40) + "\n"
					return os.WriteFile(path, []byte(want), 0644)
				}
				return nil
			}
			args := []string{"catalog", "update"}
			if failure == "mismatch" {
				args = append(args, "--revision", strings.Repeat("c", 40))
			}
			if err := l.Run(ctx, args); err == nil {
				t.Fatal("failed acquisition updated the catalog")
			}
			content, _ := os.ReadFile(path)
			entries, _ := os.ReadDir(filepath.Join(l.Root, "dist"))
			if string(content) != want || len(entries) != 0 {
				t.Fatal("failure changed lock or retained temporary output", string(content), entries)
			}
		})
	}
}

func TestCatalogInputHelpAndPathBoundaries(t *testing.T) {
	for _, args := range [][]string{{"catalog"}, {"catalog", "unknown"}, {"catalog", "update", "main"}, {"catalog", "update", "--revision=short"}, {"catalog", "update", "--revision="}, {"catalog", "update", "--revision=" + strings.Repeat("A", 40)}} {
		l, calls := catalogFixture(t)
		if err := l.Run(context.Background(), args); err == nil || len(*calls) != 0 {
			t.Fatal("invalid arguments executed", args, err)
		}
	}
	for _, args := range [][]string{{"catalog", "--help"}, {"catalog", "update", "--help"}, {"--help"}} {
		l, calls := catalogFixture(t)
		diagnostic := &bytes.Buffer{}
		l.Stderr = diagnostic
		if err := l.Run(context.Background(), args); err != nil || len(*calls) != 0 || !strings.Contains(diagnostic.String(), "catalog update") {
			t.Fatal("help acquired catalog", err, *calls, diagnostic.String())
		}
	}
	l, calls := catalogFixture(t)
	path := filepath.Join(l.Root, "catalog.lock")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.lock")
	os.WriteFile(outside, []byte(originalCatalog), 0644)
	if err := os.Symlink(outside, path); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := l.Run(context.Background(), []string{"catalog", "update"}); err == nil || len(*calls) != 0 {
		t.Fatal("outside lock accepted")
	}
	content, _ := os.ReadFile(outside)
	if string(content) != originalCatalog {
		t.Fatal("outside lock changed")
	}
}
