package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwnden/platform/internal/challenge"
	"github.com/pwnden/platform/internal/testutil"
)

func TestMain(m *testing.M) { testutil.Main(m) }

func fixture(t *testing.T) *challenge.Loaded {
	root := t.TempDir()
	return &challenge.Loaded{RepoRoot: root, Dir: root, Challenge: challenge.Challenge{
		Slug: "test", Compose: "compose.yaml", Solve: challenge.Solve{Network: "default", TimeoutSeconds: 1},
	}}
}

func configJSON(c *challenge.Loaded, patched bool) string {
	project := Project(c)
	if patched {
		project += "-patched"
	}
	return fmt.Sprintf(`{"services":{"app":{}},"networks":{"default":{"name":%q}}}`, project+"_default")
}

func TestStopRejectsSharedVolume(t *testing.T) {
	c := fixture(t)
	var cfg map[string]any
	_ = json.Unmarshal([]byte(configJSON(c, false)), &cfg)
	cfg["volumes"] = map[string]Volume{"data": {Name: "unrelated-volume"}}
	data, _ := json.Marshal(cfg)
	calls := testutil.Docker(t, testutil.Reply{Match: []string{"config"}, Out: string(data)}, testutil.Reply{Match: []string{"down"}})
	err := Stop(context.Background(), c)
	if err == nil || !strings.Contains(err.Error(), "project-scoped") {
		t.Fatalf("shared volume stop accepted: %v", err)
	}
	for _, args := range calls() {
		for _, arg := range args {
			if arg == "down" {
				t.Fatal("destructive down reached")
			}
		}
	}
}

func TestStartupRejectsImplicitHostAccess(t *testing.T) {
	for _, patched := range []bool{false, true} {
		for _, kind := range []string{"api socket", "cache import", "cache export"} {
			t.Run(fmt.Sprintf("%s/patched=%t", kind, patched), func(t *testing.T) {
				c := fixture(t)
				c.Patched = &challenge.Patched{Compose: "patched.yaml"}
				if err := os.WriteFile(filepath.Join(c.Dir, "Dockerfile"), []byte("FROM scratch\n"), 0600); err != nil {
					t.Fatal(err)
				}
				var cfg map[string]any
				if err := json.Unmarshal([]byte(configJSON(c, patched)), &cfg); err != nil {
					t.Fatal(err)
				}
				service := cfg["services"].(map[string]any)["app"].(map[string]any)
				if kind == "api socket" {
					service["use_api_socket"] = true
				} else {
					field, attribute := "cache_from", "src"
					if kind == "cache export" {
						field, attribute = "cache_to", "dest"
					}
					service["build"] = map[string]any{
						"context": c.Dir, "dockerfile": "Dockerfile",
						field: []string{"type=local," + attribute + "=" + filepath.Join(t.TempDir(), "cache")},
					}
				}
				data, err := json.Marshal(cfg)
				if err != nil {
					t.Fatal(err)
				}
				calls := testutil.Docker(t, testutil.Reply{Match: []string{"config"}, Out: string(data)})
				if patched {
					_, err = StartPatched(context.Background(), c, "flag")
				} else {
					err = Run(context.Background(), c)
				}
				if err == nil {
					t.Fatal("host access accepted")
				}
				for _, args := range calls() {
					for _, arg := range args {
						if arg == "up" {
							t.Fatalf("host access reached startup: %v", args)
						}
					}
				}
			})
		}
	}
}

func TestStopAllowsRemovedStartupInputs(t *testing.T) {
	c := fixture(t)
	missing := filepath.Join(c.RepoRoot, "removed")
	config := fmt.Sprintf(`{"services":{"app":{"volumes":[{"type":"bind","source":%q}],"build":{"context":%q,"dockerfile":"Dockerfile"}}},"networks":{"default":{"name":%q}}}`, missing, missing, Project(c)+"_default")
	calls := testutil.Docker(t, testutil.Reply{Match: []string{"config"}, Out: config}, testutil.Reply{Match: []string{"down"}})
	if err := Stop(context.Background(), c); err != nil {
		t.Fatalf("missing startup inputs prevented cleanup: %v", err)
	}
	if len(calls()) != 2 {
		t.Fatal("compose down skipped")
	}
}

func TestStopRejectsHostProvider(t *testing.T) {
	c := fixture(t)
	config := strings.Replace(configJSON(c, false), `"app":{}`, `"app":{"provider":{"type":"host-provider"}}`, 1)
	calls := testutil.Docker(t, testutil.Reply{Match: []string{"config"}, Out: config}, testutil.Reply{Match: []string{"down"}})
	if err := Stop(context.Background(), c); err == nil {
		t.Fatal("host provider teardown accepted")
	}
	if len(calls()) != 1 {
		t.Fatal("provider reached destructive down")
	}
}

func TestStopAttemptsBothProjectsAndRetainsState(t *testing.T) {
	c := fixture(t)
	c.Patched = &challenge.Patched{Compose: "patched.yaml"}
	calls := testutil.Docker(t,
		testutil.Reply{Match: []string{Project(c) + "-patched", "config"}, Err: "patched config failure", Code: 1},
		testutil.Reply{Match: []string{Project(c) + "-patched", "down"}, Err: "patched down failure", Code: 1},
		testutil.Reply{Match: []string{Project(c), "config"}, Out: configJSON(c, false)},
		testutil.Reply{Match: []string{Project(c), "down"}},
	)
	if err := SaveState(c, State{Project: Project(c), Flag: "flag"}); err != nil {
		t.Fatal(err)
	}
	if err := Stop(context.Background(), c); err == nil {
		t.Fatal("cleanup error discarded")
	}
	baseDown := false
	for _, args := range calls() {
		if strings.Contains(strings.Join(args, " "), "-p "+Project(c)+" -f") && args[len(args)-3] == "down" {
			baseDown = true
		}
	}
	if !baseDown {
		t.Fatal("base cleanup skipped")
	}
	if _, err := ReadState(c); err != nil {
		t.Fatalf("state lost after failed cleanup: %v", err)
	}
}

func TestRunReportsCleanupFailure(t *testing.T) {
	c := fixture(t)
	testutil.Docker(t,
		testutil.Reply{Match: []string{"version"}, Out: "29.4.1"},
		testutil.Reply{Match: []string{"config"}, Out: configJSON(c, false)},
		testutil.Reply{Match: []string{"up"}, Err: "startup failure", Code: 1},
		testutil.Reply{Match: []string{"down"}, Err: "cleanup failure", Code: 1},
	)
	err := Run(context.Background(), c)
	if err == nil || !strings.Contains(err.Error(), "startup failure") || !strings.Contains(err.Error(), "cleanup failure") {
		t.Fatalf("incomplete error: %v", err)
	}
}

func TestToolCleanupFailureIsExecutionFailure(t *testing.T) {
	c := fixture(t)
	c.Compose = ""
	testutil.Docker(t,
		testutil.Reply{Match: []string{"image", "inspect"}},
		testutil.Reply{Match: []string{"run"}, Err: "launch failure", Code: 125},
		testutil.Reply{Match: []string{"rm"}, Err: "removal failure", Code: 1},
		testutil.Reply{Match: []string{"container", "ls"}, Out: "leftover-container"},
	)
	_, err := RunTool(context.Background(), c, "", "image", []string{"solve"})
	if err == nil || !strings.Contains(err.Error(), "launch failure") || !strings.Contains(err.Error(), "removal failure") {
		t.Fatalf("cleanup error lost: %v", err)
	}
}

func TestToolAlreadyRemovedIsSuccessfulCleanup(t *testing.T) {
	c := fixture(t)
	c.Compose = ""
	testutil.Docker(t,
		testutil.Reply{Match: []string{"image", "inspect"}},
		testutil.Reply{Match: []string{"run"}, Err: "launch failure", Code: 125},
		testutil.Reply{Match: []string{"rm"}, Err: "no such container", Code: 1},
		testutil.Reply{Match: []string{"container", "ls"}},
	)
	_, err := RunTool(context.Background(), c, "", "image", []string{"solve"})
	if err == nil || strings.Contains(err.Error(), "no such container") {
		t.Fatalf("absent container treated as cleanup failure: %v", err)
	}
}

func TestToolImageArgumentsAndWritableUser(t *testing.T) {
	c := fixture(t)
	c.Compose = ""
	c.Solve.Writable = true
	calls := testutil.Docker(t,
		testutil.Reply{Match: []string{"image", "inspect"}, Code: 1},
		testutil.Reply{Match: []string{"pull"}},
		testutil.Reply{Match: []string{"run"}},
	)
	if _, err := RunTool(context.Background(), c, "", "image", []string{"solve"}); err != nil {
		t.Fatal(err)
	}
	for _, args := range calls() {
		for i, arg := range args {
			if arg == "image" && i > 0 && args[i-1] != "--" {
				t.Fatalf("image parsed as options: %v", args)
			}
		}
		if args[0] == "run" && os.Geteuid() >= 0 && os.Getegid() >= 0 && !strings.Contains(strings.Join(args, " "), fmt.Sprintf("--user %d:%d", os.Geteuid(), os.Getegid())) {
			t.Fatalf("caller ownership missing: %v", args)
		}
	}
}
