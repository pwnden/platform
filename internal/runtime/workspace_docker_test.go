package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBoundedWorkspaceDocker(t *testing.T) {
	if os.Getenv("PWNDEN_TEST_CHALLENGES") == "" {
		t.Skip("set PWNDEN_TEST_CHALLENGES for actual Docker storage checks")
	}
	c := fixture(t)
	c.Compose = ""
	c.Solve.Writable, c.Solve.Image, c.Solve.TimeoutSeconds = true, "pwnden-cli:basic-20261003", 60
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	original := filepath.Join(c.Dir, "input.txt")
	if err := os.WriteFile(original, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := Stop(context.Background(), c); err != nil {
			t.Error(err)
		}
	}()
	var wg sync.WaitGroup
	failures := make(chan error, 2)
	for range 2 {
		wg.Go(func() { _, err := ensureWorkspace(ctx, c, Project(c)); failures <- err })
	}
	wg.Wait()
	for range 2 {
		if err := <-failures; err != nil {
			t.Fatal(err)
		}
	}
	for _, source := range []string{
		`test "$(cat input.txt)" = original && printf changed > input.txt && printf shared > result.txt`,
		`test "$(cat input.txt)" = changed && test "$(cat result.txt)" = shared && test "$(id -u)" = 10001`,
		`python3 -c 'import os; s=os.statvfs("."); assert s.f_blocks*s.f_frsize == 256*1024**2; assert s.f_files == 32768'`,
	} {
		result, err := RunCommand(ctx, c, "", c.Solve.Image, []string{"/bin/sh", "-c", source})
		if err != nil || result.ExitCode != 0 {
			t.Fatalf("shared workspace: %+v %v", result, err)
		}
	}
	bytes, err := os.ReadFile(original)
	if err != nil || string(bytes) != "original" {
		t.Fatalf("original changed: %q %v", bytes, err)
	}
	// A second problem has its own copy even when commands run in the same daemon.
	other := *c
	other.Slug = "other"
	other.Dir = filepath.Join(c.RepoRoot, "other")
	if err := os.Mkdir(other.Dir, 0700); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := Stop(context.Background(), &other); err != nil {
			t.Error(err)
		}
	}()
	separate, err := RunCommand(ctx, &other, "", other.Solve.Image, []string{"sh", "-c", "test ! -e result.txt && printf separate > other.txt"})
	if err != nil || separate.ExitCode != 0 {
		t.Fatalf("workspace separation: %+v %v", separate, err)
	}
	if err := Stop(ctx, &other); err != nil {
		t.Fatal(err)
	}
	result, err := RunCommand(ctx, c, "", c.Solve.Image, []string{"python3", "-c", `import errno
try:
 with open("fill","wb", buffering=0) as f:
  for _ in range(260): f.write(b"x"*1024**2)
except OSError as e:
 assert e.errno == errno.ENOSPC, e
 print("quota enforced")
else: raise AssertionError("unbounded workspace")
`})
	if err != nil || result.ExitCode != 0 || !strings.Contains(result.Stdout, "quota enforced") {
		t.Fatalf("quota: %+v %v", result, err)
	}
	// A PTY gets the same volume, and can see files written by non-PTY commands.
	terminal, err := OpenTerminal(ctx, c, "", 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := terminal.Write([]byte("test \"$(cat result.txt)\" = shared; exit $?\n")); err != nil {
		terminal.Close()
		t.Fatal(err)
	}
	go func() {
		buffer := make([]byte, 4096)
		for {
			if _, err := terminal.Read(buffer); err != nil {
				return
			}
		}
	}()
	code, err := terminal.Wait(ctx)
	closeErr := terminal.Close()
	if err != nil || closeErr != nil || code != 0 {
		t.Fatalf("PTY shared files: %d %v %v", code, err, closeErr)
	}
	if err := Stop(ctx, c); err != nil {
		t.Fatal(err)
	}
	exists, err := namedContainer(ctx, Project(c)+"-workspace")
	if err != nil || exists {
		t.Fatalf("keeper remains: %v %v", exists, err)
	}
	// Starting again seeds the unchanged repository, rather than prior user data.
	result, err = RunCommand(ctx, c, "", c.Solve.Image, []string{"/bin/sh", "-c", `test "$(cat input.txt)" = original && test ! -e result.txt`})
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("fresh workspace: %+v %v", result, err)
	}
}
