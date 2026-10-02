package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/pwnden/platform/internal/challenge"
	"github.com/pwnden/platform/internal/testutil"
)

func TestPreparationUsesExecutionPolicyAndStartsNoServices(t *testing.T) {
	for _, unsafe := range []bool{false, true} {
		t.Run(strings.TrimSpace(map[bool]string{false: "valid", true: "host access"}[unsafe]), func(t *testing.T) {
			c := fixture(t)
			c.Patched = &challenge.Patched{Compose: "patched.yaml"}
			c.Solve.Image = "tool-image"
			base, patched := configJSON(c, false), configJSON(c, true)
			if unsafe {
				base = strings.Replace(base, `"app":{}`, `"app":{"use_api_socket":true}`, 1)
			}
			calls := testutil.Docker(t,
				testutil.Reply{Match: []string{Project(c) + "-patched", "config"}, Out: patched},
				testutil.Reply{Match: []string{Project(c), "config"}, Out: base},
				testutil.Reply{Match: []string{"pull", "--ignore-buildable"}},
				testutil.Reply{Match: []string{"build"}},
				testutil.Reply{Match: []string{"image", "inspect"}},
			)
			err := Prepare(context.Background(), c)
			if (err != nil) != unsafe {
				t.Fatalf("prepare: %v", err)
			}
			for _, call := range calls() {
				for _, arg := range call {
					if arg == "create" || arg == "start" || (unsafe && (arg == "pull" || arg == "build")) {
						t.Fatalf("unexpected preparation action: %v", call)
					}
				}
			}
		})
	}
}
