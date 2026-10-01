package application

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pwnden/platform/internal/runtime"
	"github.com/pwnden/platform/internal/testutil"
)

func TestFileStatusAndRecordedServiceObservation(t *testing.T) {
	ctx := context.Background()
	s, c := fixture(t, false)
	t.Setenv("PATH", t.TempDir())
	result, err := s.Status(ctx, c.Slug)
	if err != nil || result.State != "ready" || result.Kind != KindFile || len(result.Endpoints) != 0 {
		t.Fatalf("file: %+v %v", result, err)
	}
	s, c = fixture(t, true)
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("LOCALAPPDATA", cache)
	result, err = s.Status(ctx, c.Slug)
	if err != nil || result.State != "stopped" {
		t.Fatalf("stopped: %+v %v", result, err)
	}
	if err := runtime.SaveState(c, runtime.State{Project: runtime.Project(c), Flag: "pwnden{private}"}); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ output, state string }{
		{`[{"Service":"app","State":"running","Health":"healthy"}]`, "running"},
		{`{"Service":"app","State":"running","Health":""}` + "\n", "running"},
		{`[]`, "unavailable"},
		{`[{"Service":"app","State":"exited","Health":""}]`, "unavailable"},
		{`[{"Service":"app","State":"running","Health":"unhealthy"}]`, "unavailable"},
	} {
		t.Run(test.state+test.output, func(t *testing.T) {
			fake := testutil.Docker(t, testutil.WithIsolatedNetwork(runtime.Project(c), "network-id",
				testutil.Reply{Match: []string{"config"}, Out: config(c)},
				testutil.Reply{Match: []string{"ps"}, Out: test.output},
			)...)
			t.Setenv("XDG_CACHE_HOME", cache)
			t.Setenv("LOCALAPPDATA", cache)
			// Fresh service instances recover the same recorded run.
			fresh, _ := New(c.RepoRoot)
			result, err := fresh.Status(ctx, c.Slug)
			if err != nil || result.State != test.state || (test.state == "running" && len(result.Endpoints) != 1) {
				t.Fatalf("observed: %+v %v", result, err)
			}
			if _, err := runtime.ReadState(c); err != nil {
				t.Fatal("observation changed saved state", err)
			}
			for _, call := range fake() {
				for _, arg := range call {
					if arg == "up" || arg == "down" {
						t.Fatal("observation mutated Docker", call)
					}
				}
			}
		})
	}
	os.Remove(filepath.Join(c.Dir, "file.txt"))
	testutil.Docker(t, testutil.WithIsolatedNetwork(runtime.Project(c), "network-id", testutil.Reply{Match: []string{"config"}, Out: config(c)}, testutil.Reply{Match: []string{"ps"}, Out: `[]`})...)
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("LOCALAPPDATA", cache)
	if result, err := s.Status(ctx, c.Slug); err != nil || result.State != "unavailable" {
		t.Fatalf("missing distribution still permits observation: %+v %v", result, err)
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	_, err = s.Status(ctx, c.Slug)
	requireCode(t, err, Canceled)
}

func TestStatusErrorsPreserveRun(t *testing.T) {
	s, c := fixture(t, true)
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("LOCALAPPDATA", cache)
	if err := runtime.SaveState(c, runtime.State{Project: runtime.Project(c), Flag: "private"}); err != nil {
		t.Fatal(err)
	}
	for _, reply := range []testutil.Reply{
		{Match: []string{"ps"}, Err: "private output", Code: 1},
		{Match: []string{"ps"}, Out: "not JSON"},
	} {
		testutil.Docker(t, testutil.WithIsolatedNetwork(runtime.Project(c), "network-id", testutil.Reply{Match: []string{"config"}, Out: config(c)}, reply)...)
		t.Setenv("XDG_CACHE_HOME", cache)
		t.Setenv("LOCALAPPDATA", cache)
		_, err := s.Status(context.Background(), c.Slug)
		requireCode(t, err, ExecutionFailed)
		if _, err := runtime.ReadState(c); err != nil {
			t.Fatal(err)
		}
	}
}
