// Check a packaged player's terminal with real Docker. Credentials arrive on
// stdin, stay in memory and never enter diagnostics or command arguments.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/coder/websocket"
)

type config struct{ Origin, Token, Slug, Root, Mode string }
type terminal struct {
	conn   *websocket.Conn
	ctx    context.Context
	output bytes.Buffer
}

func (s *terminal) control(value any) error {
	data, _ := json.Marshal(value)
	return s.conn.Write(s.ctx, websocket.MessageText, data)
}
func connect(ctx context.Context, c config) (*terminal, error) {
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(c.Origin, "http")+"/api/v1/problems/"+c.Slug+"/terminal", &websocket.DialOptions{Subprotocols: []string{"pwnden.terminal.v2"}, HTTPHeader: http.Header{"Origin": []string{c.Origin}}})
	if err != nil {
		return nil, errors.New("terminal upgrade failed")
	}
	s := &terminal{conn: conn, ctx: ctx}
	if err := s.control(map[string]any{"type": "authenticate", "token": c.Token, "cols": 80, "rows": 24}); err != nil {
		conn.CloseNow()
		return nil, err
	}
	typ, data, err := conn.Read(ctx)
	var ready struct {
		Type   string
		Reused bool
	}
	if err != nil || typ != websocket.MessageText || json.Unmarshal(data, &ready) != nil || ready.Type != "ready" {
		conn.CloseNow()
		return nil, errors.New("terminal was not ready")
	}
	return s, nil
}
func (s *terminal) input(command string) error {
	return s.conn.Write(s.ctx, websocket.MessageBinary, []byte(command))
}
func (s *terminal) until(marker string) error {
	s.output.Reset()
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()
	for !strings.Contains(s.output.String(), marker) {
		typ, data, err := s.conn.Read(ctx)
		if err != nil {
			return errors.New("terminal output ended early")
		}
		if typ != websocket.MessageBinary {
			return errors.New("unexpected terminal control")
		}
		if s.output.Len()+len(data) > 1<<20 {
			return errors.New("terminal output limit exceeded")
		}
		s.output.Write(data)
		if err := s.control(map[string]string{"type": "ack"}); err != nil {
			return err
		}
	}
	return nil
}

// Send the same UTF-8 bytes and control sequences as xterm. Assertions match
// command results rather than input echo, so a shell without Readline fails.
func (s *terminal) editing() error {
	cases := []struct{ name, input, result string }{
		{"UTF-8 erase", "printf '\\137\\137UTF8__:%s\\n' '가나\x7f\x7fok'\r", "__UTF8__:ok\r\n"},
		{"erase at prompt", "한글" + strings.Repeat("\x7f", 20) + "printf '\\137\\137BOUNDARY__:ok\\n'\r", "__BOUNDARY__:ok\r\n"},
		{"path completion", "printf '\\137\\137COMPLETE__:%s\\n' /challenge/sol\t\r", "__COMPLETE__:/challenge/solve/\r\n"},
		{"history seed", "printf '\\137\\137HISTORY__:ok\\n'\r", "__HISTORY__:ok\r\n"},
		{"previous command", "\x1b[A\r", "__HISTORY__:ok\r\n"},
		{"next command", "\x1b[A\x1b[Bprintf '\\137\\137NEXT__:ok\\n'\r", "__NEXT__:ok\r\n"},
		{"left/right and Delete", "printf '\\137\\137CURSOR__:%s\\n' 'axc'\x1b[D\x1b[D\x1b[D\x1b[D\x1b[C\x1b[3~b\r", "__CURSOR__:abc\r\n"},
		{"Home / End", "rintf '\\137\\137HOME__:%s\\n' 'ok'\x1b[Hp\x1b[F\r", "__HOME__:ok\r\n"},
		{"Ctrl+A / Ctrl+E", "rintf '\\137\\137ENDS__:%s\\n' 'ok'\x01p\x05\r", "__ENDS__:ok\r\n"},
		{"Ctrl+D forward erase", "printf '\\137\\137DELETE__:%s\\n' 'axc'\x1b[D\x1b[D\x1b[D\x04b\r", "__DELETE__:abc\r\n"},
		{"Ctrl+L", "\x0cprintf '\\137\\137CLEAR__:ok\\n'\r", "__CLEAR__:ok\r\n"},
		{"Ctrl+U", "discard this\x15printf '\\137\\137KILL__:ok\\n'\r", "__KILL__:ok\r\n"},
		{"Ctrl+W", "printf '\\137\\137WORD__:%s\\n' wrong\x17ok\r", "__WORD__:ok\r\n"},
		{"Ctrl+K / Ctrl+Y", "printf '\\137\\137YANK__:%s\\n' 'ok'\x01\x0b\x19\r", "__YANK__:ok\r\n"},
		{"Ctrl+R", "\x12YANK__\r", "__YANK__:ok\r\n"},
		{"bracketed paste", "\x1b[200~printf '\\137\\137PASTE__:%s\\n' '가나다'\x1b[201~\r", "__PASTE__:가나다\r\n"},
	}
	for _, check := range cases {
		if err := s.input(check.input); err != nil {
			return err
		}
		if err := s.until(check.result); err != nil {
			return fmt.Errorf("%s failed: %w", check.name, err)
		}
	}
	return nil
}

func (s *terminal) jobControl() error {
	if err := s.input("sleep 30\r"); err != nil {
		return err
	}
	time.Sleep(200 * time.Millisecond)
	if err := s.input("\x1ajobs; printf '\\137\\137JOBS_END__\\n'\r"); err != nil {
		return err
	}
	if err := s.until("__JOBS_END__\r\n"); err != nil {
		return err
	}
	if !strings.Contains(s.output.String(), "Stopped") || !strings.Contains(s.output.String(), "sleep 30") {
		return errors.New("Ctrl+Z did not suspend the foreground job")
	}
	if err := s.input("fg\r"); err != nil {
		return err
	}
	time.Sleep(200 * time.Millisecond)
	if err := s.input("\x03printf '\\137\\137RESUMED_END__\\n'\r"); err != nil {
		return err
	}
	return s.until("__RESUMED_END__\r\n")
}
func docker(ctx context.Context, args ...string) ([]byte, error) {
	data, err := exec.CommandContext(ctx, "docker", args...).Output()
	if err != nil {
		return nil, errors.New("Docker inspection failed")
	}
	return data, nil
}
func checkGone(ctx context.Context, id string) error {
	for i := 0; i < 100; i++ {
		data, err := docker(ctx, "ps", "-a", "--filter", "id="+id, "--format", "{{.ID}}")
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(data)) == "" {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return errors.New("terminal container survived cleanup")
}
func run(c config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	s, err := connect(ctx, c)
	if err != nil {
		return err
	}
	defer s.conn.CloseNow()
	// Remove input echo so output markers cannot match the sent commands.
	if err := s.input("stty -echo; printf '\\137\\137ECHO_OFF\\137\\137\\n'\n"); err != nil {
		return err
	}
	if err := s.until("__ECHO_OFF__\r\n"); err != nil {
		return err
	}
	if err := s.editing(); err != nil {
		return err
	}
	if err := s.jobControl(); err != nil {
		return err
	}
	if err := s.input("printf '__ID__'; cat /etc/hostname; printf '__ID_END__\\n'\n"); err != nil {
		return err
	}
	if err := s.until("__ID_END__\r\n"); err != nil {
		return err
	}
	output := s.output.String()
	start := strings.Index(output, "__ID__") + len("__ID__")
	end := strings.Index(output, "__ID_END__")
	if start < len("__ID__") || end < start {
		return errors.New("container identity missing")
	}
	id := strings.TrimSpace(output[start:end])
	data, err := docker(ctx, "inspect", id)
	if err != nil {
		return err
	}
	var containers []struct {
		Config     struct{ Labels map[string]string }
		HostConfig struct {
			NetworkMode          string
			Privileged           bool
			CapDrop, SecurityOpt []string
		}
		Mounts []struct {
			Type, Source, Destination string
			RW                        bool
		}
	}
	if err := json.Unmarshal(data, &containers); err != nil || len(containers) != 1 {
		return errors.New("invalid container inspection")
	}
	owned := containers[0]
	if owned.Config.Labels["pwnden.kind"] != "terminal" || owned.Config.Labels["pwnden.problem"] != c.Slug {
		return errors.New("unexpected container ownership")
	}
	if len(owned.Mounts) != 1 || owned.Mounts[0].Type != "bind" || owned.Mounts[0].Destination != "/challenge" || owned.Mounts[0].RW || owned.Mounts[0].Source != filepath.Join(c.Root, "challenges", c.Slug) {
		return errors.New("terminal mount policy differs")
	}
	if owned.HostConfig.Privileged || strings.Join(owned.HostConfig.CapDrop, ",") != "ALL" || !strings.Contains(strings.Join(owned.HostConfig.SecurityOpt, ","), "no-new-privileges") {
		return errors.New("terminal privilege policy differs")
	}
	if c.Slug == "rotor-lock" && owned.HostConfig.NetworkMode != "none" {
		return errors.New("file terminal acquired network")
	}
	if c.Slug == "note-vault" {
		network, err := docker(ctx, "network", "inspect", owned.HostConfig.NetworkMode, "--format", "{{index .Labels \"com.docker.compose.project\"}}")
		if err != nil || strings.TrimSpace(string(network)) != owned.Config.Labels["pwnden.project"] {
			return errors.New("terminal acquired unrelated network")
		}
	}
	if err := s.control(map[string]any{"type": "resize", "cols": 120, "rows": 40}); err != nil {
		return err
	}
	if err := s.input("sleep 0.2; stty size; printf '__SIZE_END__\\n'\n"); err != nil {
		return err
	}
	if err := s.until("__SIZE_END__\r\n"); err != nil {
		return err
	}
	if !strings.Contains(s.output.String(), "40 120") {
		return errors.New("daemon TTY did not resize")
	}
	if err := s.input("sleep 30\n"); err != nil {
		return err
	}
	time.Sleep(200 * time.Millisecond)
	if err := s.conn.Write(ctx, websocket.MessageBinary, []byte{3}); err != nil {
		return err
	}
	if err := s.input("printf '__INTERRUPTED__\\n'\n"); err != nil {
		return err
	}
	if err := s.until("__INTERRUPTED__\r\n"); err != nil {
		return err
	}
	if c.Slug == "rotor-lock" {
		if err := s.input("python3 solve/solve.py; printf '__SOLVED__\\n'\n"); err != nil {
			return err
		}
		if err := s.until("__SOLVED__\r\n"); err != nil {
			return err
		}
		if !strings.Contains(s.output.String(), "pwnden{") {
			return errors.New("file solver failed inside terminal")
		}
	} else {
		if err := s.input("python3 -c \"import urllib.request; print(urllib.request.urlopen('http://app:8000/healthz').status)\"; printf '__NETWORK_END__\\n'\n"); err != nil {
			return err
		}
		if err := s.until("__NETWORK_END__\r\n"); err != nil {
			return err
		}
		if !strings.Contains(s.output.String(), "200") {
			return errors.New("terminal cannot access problem service")
		}
	}
	if c.Mode == "stop" {
		req, _ := http.NewRequestWithContext(ctx, "DELETE", c.Origin+"/api/v1/problems/"+c.Slug+"/run", nil)
		req.Header.Set("Authorization", "Bearer "+c.Token)
		req.Header.Set("Origin", c.Origin)
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			return errors.New("stop request failed")
		}
		response.Body.Close()
		if response.StatusCode != 200 {
			return errors.New("problem stop did not clean terminal")
		}
	} else if c.Mode == "retain" {
		usage, err := docker(ctx, "stats", "--no-stream", "--format", "memory={{.MemUsage}} cpu={{.CPUPerc}}", id)
		if err != nil {
			return err
		}
		fmt.Printf("Retained toolbox %s: %s\n", c.Slug, strings.TrimSpace(string(usage)))
		s.conn.CloseNow()
		return nil
	} else if c.Mode == "disconnect" {
		if err := s.input("cd /tmp; export PWNDEN_KEEP=retained; printf retained > pwnden-kept; sleep 300 & (sleep 0.3; seq 1 5000; printf '\\137\\137DETACHED\\137\\137\\n') & printf '\\137\\137RETAIN_READY\\137\\137\\n'\n"); err != nil {
			return err
		}
		if err := s.until("__RETAIN_READY__\r\n"); err != nil {
			return err
		}
		s.conn.CloseNow()
		time.Sleep(800 * time.Millisecond)
		// Detached output must keep draining even when it exceeds browser queues.
		next, err := connect(ctx, c)
		if err != nil {
			return err
		}
		defer next.conn.CloseNow()
		if err := next.input("printf '__RETAIN__'; printf '%s %s ' \"$PWD\" \"$PWNDEN_KEEP\"; cat pwnden-kept /etc/hostname; jobs -r; history 3; printf '__RETAIN_END__\\n'\n"); err != nil {
			return err
		}
		if err := next.until("__RETAIN_END__\r\n"); err != nil {
			return err
		}
		output := next.output.String()
		for _, check := range []struct{ name, marker string }{{"cwd/env/file/identity", "/tmp retained retained" + id}, {"background job", "sleep 300"}, {"history", "PWNDEN_KEEP=retained"}, {"detached output", "__DETACHED__"}} {
			if !strings.Contains(output, check.marker) {
				return fmt.Errorf("reattachment lost %s", check.name)
			}
		}
		req, _ := http.NewRequestWithContext(ctx, "DELETE", c.Origin+"/api/v1/problems/"+c.Slug+"/terminal", nil)
		req.Header.Set("Authorization", "Bearer "+c.Token)
		req.Header.Set("Origin", c.Origin)
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			return errors.New("terminal cleanup request failed")
		}
		response.Body.Close()
		if response.StatusCode != 200 {
			return errors.New("terminal cleanup failed")
		}
	} else {
		if err := s.input("exit 7\n"); err != nil {
			return err
		}
		for {
			typ, data, err := s.conn.Read(ctx)
			if err != nil {
				return errors.New("shell exit was not reported")
			}
			if typ == websocket.MessageBinary {
				s.control(map[string]string{"type": "ack"})
				continue
			}
			if string(data) != `{"code":7,"type":"exit"}` {
				return errors.New("wrong shell exit code")
			}
			break
		}
	}
	return checkGone(ctx, id)
}
func main() {
	var c config
	if err := json.NewDecoder(io.LimitReader(os.Stdin, 4096)).Decode(&c); err != nil {
		fmt.Fprintln(os.Stderr, "invalid terminal smoke input")
		os.Exit(1)
	}
	if err := run(c); err != nil {
		fmt.Fprintln(os.Stderr, "terminal smoke:", err)
		os.Exit(1)
	}
	fmt.Println("Terminal smoke passed:", c.Slug, c.Mode)
}
