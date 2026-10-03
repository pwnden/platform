package runtime

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/vt"
	"github.com/pwnden/platform/internal/challenge"
)

// Opt in with a local challenge checkout and Docker. OpenTerminal creates a
// fresh toolbox; cleanup removes only that toolbox, leaving player sessions alone.
func TestTerminalFilenameCompletionDocker(t *testing.T) {
	repo := os.Getenv("PWNDEN_TEST_CHALLENGES")
	if repo == "" {
		t.Skip("set PWNDEN_TEST_CHALLENGES to run the actual Docker PTY check")
	}
	c, err := challenge.Load(repo, "rotor-lock")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	terminal, err := OpenTerminal(ctx, c, "", 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		source := terminal.source
		if err := terminal.Close(); err != nil {
			t.Error(err)
		}
		if _, err := os.Stat(source); !os.IsNotExist(err) {
			t.Errorf("readonly player copy remains: %v", err)
		}
	}()
	output := make(chan string, 32)
	go func() {
		defer close(output)
		buf := make([]byte, 4096)
		for {
			n, err := terminal.Read(buf)
			if n > 0 {
				select {
				case output <- string(buf[:n]):
				case <-ctx.Done():
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()
	cases := []struct{ name, input, want string }{
		{"directory prompt", "", "\x1b[36m/workspace\x1b[0m "},
		{"disable input echo", "stty -echo; printf '\\137\\137READY__\\n'\r", "__READY__\r\n"},
		{"prompt follows working directory", "cd /tmp\r", "\x1b[36m/tmp\x1b[0m "},
		{"return to problem directory", "cd /workspace\r", "\x1b[36m/workspace\x1b[0m "},
		{"JSON without final newline", "printf '{\"error\": \"staff_access_required\"}'\r", "{\"error\": \"staff_access_required\"}\r\n\x1b[36m/workspace\x1b[0m "},
		{"text without final newline", "printf '\\137\\137PARTIAL__'\r", "__PARTIAL__\r\n\x1b[36m/workspace\x1b[0m "},
		{"already terminated output", "printf '\\137\\137TERMINATED__\\n'\r", "__TERMINATED__\r\n\r\n\x1b[36m/workspace\x1b[0m "},
		{"empty output", "true\r", "\r\n\x1b[36m/workspace\x1b[0m "},
		{"private files absent", "test ! -e README.md && test ! -e AUTHORING.md && test ! -e challenge.toml && test ! -e hints && test ! -e solve && printf '\\137\\137PRIVATE__:absent\\n'\r", "__PRIVATE__:absent\r\n"},
		{"relative filename ignoring case", "printf '\\137\\137RELATIVE__:%s\\n' files/CHECK\t\r", "__RELATIVE__:files/checker.py\r\n"},
		{"absolute filename ignoring case", "printf '\\137\\137ABSOLUTE__:%s\\n' /workspace/files/CHECK\t\r", "__ABSOLUTE__:/workspace/files/checker.py\r\n"},
		{"directory completion", "printf '\\137\\137DIRECTORY__:%s\\n' /workspace/fil\t\r", "__DIRECTORY__:/workspace/files/\r\n"},
		{"completion candidates", "mkdir -p /tmp/pwnden-completion; touch /tmp/pwnden-completion/choice-{alpha,beta,gamma}; printf '\\137\\137CANDIDATES__:ok\\n'\r", "__CANDIDATES__:ok\r\n"},
		{"first candidate", "printf '\\137\\137FIRST__:%s\\n' /tmp/pwnden-completion/choice-\t\r", "__FIRST__:/tmp/pwnden-completion/choice-alpha\r\n"},
		{"next candidate", "printf '\\137\\137SECOND__:%s\\n' /tmp/pwnden-completion/choice-\t\t\r", "__SECOND__:/tmp/pwnden-completion/choice-beta\r\n"},
		{"completion wraparound", "printf '\\137\\137WRAP__:%s\\n' /tmp/pwnden-completion/choice-" + strings.Repeat("\t", 5) + "\r", "__WRAP__:/tmp/pwnden-completion/choice-alpha\r\n"},
		{"Shift+Tab previous candidate", "printf '\\137\\137PREVIOUS__:%s\\n' /tmp/pwnden-completion/choice-\t\t\x1b[Z\r", "__PREVIOUS__:/tmp/pwnden-completion/choice-alpha\r\n"},
		{"Shift+Tab first candidate", "printf '\\137\\137REVERSE__:%s\\n' /tmp/pwnden-completion/choice-\x1b[Z\r", "__REVERSE__:/tmp/pwnden-completion/choice-gamma\r\n"},
		{"no match keeps input", "printf '\\137\\137MISSING__:%s\\n' /tmp/pwnden-completion/missing-\t\r", "__MISSING__:/tmp/pwnden-completion/missing-\r\n"},
		{"UTF-8 erase", "printf '\\137\\137ERASE__:%s\\n' '가나\x7f\x7fok'\r", "__ERASE__:ok\r\n"},
		{"command history", "\x1b[A\r", "__ERASE__:ok\r\n"},
	}
	promptScreens := map[string]string{
		"JSON without final newline": "{\"error\": \"staff_access_required\"}\n/workspace",
		"text without final newline": "__PARTIAL__\n/workspace",
		"already terminated output":  "__TERMINATED__\n\n/workspace",
		"empty output":               "\n/workspace",
	}
	for _, check := range cases {
		t.Run(check.name, func(t *testing.T) {
			if _, err := io.WriteString(terminal, check.input); err != nil {
				t.Fatal(err)
			}
			deadline := time.NewTimer(5 * time.Second)
			defer deadline.Stop()
			var received strings.Builder
			screen := vt.NewEmulator(80, 24)
			defer screen.Close()
			matches := func() bool {
				if want, ok := promptScreens[check.name]; ok {
					return strings.Contains(screen.String(), want)
				}
				return strings.Contains(received.String(), check.want)
			}
			for !matches() {
				select {
				case chunk, ok := <-output:
					if !ok {
						t.Fatal("toolbox output ended before the result")
					}
					received.WriteString(chunk)
					screen.WriteString(chunk)
				case <-deadline.C:
					t.Fatalf("expected %q in PTY output %q", check.want, received.String())
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			}
			if check.input == "" && strings.Contains(received.String(), c.Slug) {
				t.Fatal("directory prompt includes the problem name")
			}
		})
		if t.Failed() {
			break
		}
	}
}
