package verify

import (
	"context"
	"fmt"
	"testing"

	"github.com/pwnden/platform/internal/challenge"
	"github.com/pwnden/platform/internal/runtime"
	"github.com/pwnden/platform/internal/testutil"
)

func TestMain(m *testing.M) { testutil.Main(m) }

func TestPatchedAttackOutcomes(t *testing.T) {
	for _, test := range []struct {
		name, output string
		code, delay  int
		pass         bool
	}{
		{"explicit rejection", "", 3, 0, true},
		{"successful no flag", "", 0, 0, true},
		{"successful different output", "attack denied", 0, 0, true},
		{"solver internal failure", "", 1, 0, false},
		{"flag with failure exit", "pwnden{test}", 1, 0, false},
		{"flag with rejection exit", "pwnden{test}", 3, 0, false},
		{"flag with successful exit", "pwnden{test}", 0, 0, false},
		{"Docker failure", "", 125, 0, false},
		{"command not executable", "", 126, 0, false},
		{"command missing", "", 127, 0, false},
		{"killed command", "", 137, 0, false},
		{"timeout", "", 0, 1500, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			c := &challenge.Loaded{RepoRoot: root, Dir: root, Contract: challenge.Contract{AttackRejectedExit: 3}, Challenge: challenge.Challenge{Slug: "test", Compose: "compose.yaml", Solve: challenge.Solve{Image: "image", Command: []string{"attack"}, Network: "default", TimeoutSeconds: 1}, Patched: &challenge.Patched{Compose: "patched.yaml", Check: []string{"check"}}}}
			project := runtime.Project(c)
			config := func(project string) string {
				return fmt.Sprintf(`{"services":{"app":{}},"networks":{"default":{"name":%q}}}`, project+"_default")
			}
			testutil.Docker(t, testutil.WithIsolatedNetwork(project, "base-network", testutil.WithIsolatedNetwork(project+"-patched", "patched-network",
				testutil.Reply{Match: []string{project + "-patched", "config"}, Out: config(project + "-patched")},
				testutil.Reply{Match: []string{project, "config"}, Out: config(project)},
				testutil.Reply{Match: []string{"up"}}, testutil.Reply{Match: []string{"down"}},
				testutil.Reply{Match: []string{"image", "inspect"}, Out: "image-id"},
				testutil.Reply{Match: []string{"network", "label=com.docker.compose.project=" + project + "-patched"}, Out: "patched-network"},
				testutil.Reply{Match: []string{"network", "ls"}, Out: "base-network"},
				testutil.Reply{Match: []string{"run", "patched-network", "attack"}, Out: test.output, Code: test.code, DelayMS: test.delay},
				testutil.Reply{Match: []string{"run", "base-network", "attack"}, Out: "pwnden{test}"},
				testutil.Reply{Match: []string{"run", "check"}}, testutil.Reply{Match: []string{"rm"}},
			)...)...)
			if err := runtime.SaveState(c, runtime.State{Project: project, Flag: "pwnden{test}"}); err != nil {
				t.Fatal(err)
			}
			result, err := Challenge(context.Background(), c)
			if test.pass {
				if err != nil || !result.Patched {
					t.Fatalf("valid patch rejected: %v %+v", err, result)
				}
			} else if err == nil {
				t.Fatalf("invalid patch accepted: %+v", result)
			}
		})
	}
}
