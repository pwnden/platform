// Package testutil provides a Docker process double without contacting a daemon.
package testutil

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type Reply struct {
	Match         []string
	Out, Err      string
	Code          int
	DelayMS       int
	InputContains []string
}

func Main(m *testing.M) {
	if strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe") == "docker" {
		data, err := os.ReadFile(os.Getenv("PWNDEN_TEST_REPLIES"))
		if err != nil {
			panic(err)
		}
		var replies []Reply
		if err := json.Unmarshal(data, &replies); err != nil {
			panic(err)
		}
		log, err := os.OpenFile(os.Getenv("PWNDEN_TEST_LOG"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}
		_ = json.NewEncoder(log).Encode(os.Args[1:])
		_ = log.Close()
		for _, reply := range replies {
			i := 0
			for _, arg := range os.Args[1:] {
				if i < len(reply.Match) && arg == reply.Match[i] {
					i++
				}
			}
			if i != len(reply.Match) {
				continue
			}
			if len(reply.InputContains) > 0 {
				input, err := io.ReadAll(os.Stdin)
				if err != nil {
					panic(err)
				}
				for _, value := range reply.InputContains {
					if !strings.Contains(string(input), value) {
						fmt.Fprintln(os.Stderr, "expected isolated compose input missing", value)
						os.Exit(125)
					}
				}
			}
			time.Sleep(time.Duration(reply.DelayMS) * time.Millisecond)
			fmt.Fprint(os.Stdout, reply.Out)
			fmt.Fprint(os.Stderr, reply.Err)
			os.Exit(reply.Code)
		}
		fmt.Fprintln(os.Stderr, "unexpected fake Docker call", os.Args[1:])
		os.Exit(125)
	}
	os.Exit(m.Run())
}

// WithIsolatedNetwork supplies a real-shaped daemon response for a scoped,
// isolated network. Tests for unsafe networks supply their own explicit reply.
func WithIsolatedNetwork(project, id string, replies ...Reply) []Reply {
	network := map[string]any{
		"ID": id, "Name": project + "_default", "Driver": "bridge", "Internal": true,
		"Labels":  map[string]string{"com.docker.compose.project": project, "com.docker.compose.network": "default"},
		"Options": map[string]string{"com.docker.network.bridge.gateway_mode_ipv4": "isolated", "com.docker.network.bridge.gateway_mode_ipv6": "isolated"},
	}
	data, _ := json.Marshal([]any{network})
	return append([]Reply{
		{Match: []string{"version", "{{.Server.Version}}"}, Out: "29.4.1"},
		{Match: []string{"network", "ls", "label=com.docker.compose.project=" + project}, Out: id},
		{Match: []string{"network", "inspect", id}, Out: string(data)},
	}, replies...)
}

func Docker(t *testing.T, replies ...Reply) func() [][]string {
	t.Helper()
	dir := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	name := "docker"
	if strings.HasSuffix(exe, ".exe") {
		name += ".exe"
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0700); err != nil {
		t.Fatal(err)
	}
	data, err = json.Marshal(replies)
	if err != nil {
		t.Fatal(err)
	}
	rules := filepath.Join(dir, "replies.json")
	if err := os.WriteFile(rules, data, 0600); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(dir, "calls.jsonl")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("PWNDEN_TEST_REPLIES", rules)
	t.Setenv("PWNDEN_TEST_LOG", log)
	t.Setenv("XDG_CACHE_HOME", dir)
	t.Setenv("LOCALAPPDATA", dir)
	return func() [][]string {
		file, err := os.Open(log)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		var calls [][]string
		decoder := json.NewDecoder(file)
		for decoder.More() {
			var args []string
			if err := decoder.Decode(&args); err != nil {
				t.Fatal(err)
			}
			calls = append(calls, args)
		}
		return calls
	}
}
