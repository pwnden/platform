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
		matchArgs := os.Args[1:]
		// A Docker start has no image/command flags. Associate the preceding
		// create request so fixtures can select the command being executed.
		if len(matchArgs) > 0 && matchArgs[0] == "start" {
			data, _ := os.ReadFile(os.Getenv("PWNDEN_TEST_LOG"))
			for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
				var previous []string
				if json.Unmarshal([]byte(line), &previous) == nil && len(previous) > 0 && previous[0] == "create" {
					for i, arg := range previous {
						if arg == "--name" && i+1 < len(previous) && previous[i+1] == os.Args[len(os.Args)-1] {
							matchArgs = append(append([]string{}, os.Args[1:]...), previous[1:]...)
						}
					}
				}
			}
		}
		for _, reply := range replies {
			i := 0
			for _, arg := range matchArgs {
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
	container, _ := json.Marshal([]any{map[string]any{
		"Config": map[string]any{"Labels": map[string]string{"com.docker.compose.project": project, "pwnden.runtime-policy": "docker-v2"}},
		"HostConfig": map[string]any{"ReadonlyRootfs": true, "CapDrop": []string{"ALL"}, "SecurityOpt": []string{"no-new-privileges"},
			"CgroupnsMode": "private", "NanoCpus": 2e9, "Memory": 2 << 30, "MemorySwap": 2 << 30, "PidsLimit": 256,
			"Tmpfs":     map[string]string{"/tmp": "rw,exec,nosuid,nodev,size=128m"},
			"LogConfig": map[string]any{"Type": "local", "Config": map[string]string{"max-size": "10m", "max-file": "3"}}},
	}})
	return append([]Reply{
		{Match: []string{"version", "{{.Server.Version}}"}, Out: "29.4.1"},
		{Match: []string{"network", "ls", "label=com.docker.compose.project=" + project}, Out: id},
		{Match: []string{"network", "inspect", id}, Out: string(data)},
		{Match: []string{"container", "ls", "label=com.docker.compose.project=" + project}, Out: project + "-service"},
		{Match: []string{"container", "inspect", project + "-service"}, Out: string(container)},
	}, replies...)
}

func Docker(t *testing.T, replies ...Reply) func() [][]string {
	t.Helper()
	for i := range replies {
		if strings.Join(replies[i].Match, " ") == "image inspect" {
			replies[i].Match = append(replies[i].Match, "{{.Id}}")
		}
		for j, arg := range replies[i].Match {
			if j == 0 && arg == "run" {
				replies[i].Match[j] = "start"
			}
			if arg == "up" {
				replies[i].Match[j] = "create"
			}
		}
	}
	replies = append(replies,
		Reply{Match: []string{"info", `{"ID":{{json .ID}},"NCPU":{{.NCPU}},"MemTotal":{{.MemTotal}}}`}, Out: `{"ID":"test-daemon","NCPU":16,"MemTotal":17179869184}`},
		Reply{Match: []string{"image", "inspect", "{{json .Config.Volumes}}"}, Out: "null"},
		Reply{Match: []string{"container", "ls"}},
		Reply{Match: []string{"volume", "ls"}},
		Reply{Match: []string{"create"}},
		Reply{Match: []string{"start"}},
		Reply{Match: []string{"compose", "pull"}},
		Reply{Match: []string{"compose", "build"}},
	)
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
