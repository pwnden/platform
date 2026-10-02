package application

import (
	"context"
	"fmt"
	"testing"

	"github.com/pwnden/platform/internal/runtime"
	"github.com/pwnden/platform/internal/testutil"
)

func TestCatalogAndSubmissions(t *testing.T) {
	for _, service := range []bool{false, true} {
		t.Run(fmt.Sprintf("service=%t", service), func(t *testing.T) {
			s, c := fixture(t, service)
			problems, err := s.List(context.Background())
			if err != nil || len(problems) != 1 || problems[0].Slug != c.Slug || problems[0].Kind != kind(c) {
				t.Fatalf("catalog: %+v %v", problems, err)
			}
			if service {
				testutil.Docker(t)
				_, err := s.Submit(context.Background(), c.Slug, "pwnden{test}")
				requireCode(t, err, NotRunning)
				if err := runtime.SaveState(c, runtime.State{Project: runtime.Project(c), Flag: "pwnden{test}"}); err != nil {
					t.Fatal(err)
				}
			}
			for _, candidate := range []string{"pwnden{test}", " \tpwnden{test}\n", "wrong"} {
				result, err := s.Submit(context.Background(), c.Slug, candidate)
				if err != nil || result.Accepted != (candidate != "wrong") {
					t.Fatalf("submission: %+v %v", result, err)
				}
			}
		})
	}
}

func TestPlayerToolExecution(t *testing.T) {
	for _, code := range []int{0, 1, 125} {
		t.Run(fmt.Sprintf("exit=%d", code), func(t *testing.T) {
			s, c := fixture(t, false)
			calls := testutil.Docker(t,
				testutil.Reply{Match: []string{"image", "inspect"}},
				testutil.Reply{Match: []string{"run"}, Out: "command output", Err: "command diagnostic", Code: code},
				testutil.Reply{Match: []string{"rm"}},
			)
			result, err := s.Execute(context.Background(), c.Slug, []string{"python3", "files/checker.py", "candidate"})
			if code == 125 {
				requireCode(t, err, ExecutionFailed)
				if result.Stdout != "" {
					t.Fatal("execution failure exposed partial output")
				}
			} else if err != nil || result.ExitCode != code || result.Stdout != "command output" || result.Stderr != "command diagnostic" {
				t.Fatalf("execution: %+v %v", result, err)
			}
			for _, call := range calls() {
				if len(call) > 0 && call[0] == "create" {
					if call[len(call)-3] != "python3" || call[len(call)-2] != "files/checker.py" || call[len(call)-1] != "candidate" {
						t.Fatalf("command args changed: %v", call)
					}
					foundNone := false
					for i, arg := range call {
						if arg == "--network" && call[i+1] == "none" {
							foundNone = true
						}
					}
					if !foundNone {
						t.Fatal("file toolbox acquired network access")
					}
				}
			}
		})
	}
	_, err := New("missing")
	if err != nil {
		t.Fatal(err)
	}
	s, c := fixture(t, false)
	_, err = s.Execute(context.Background(), c.Slug, nil)
	requireCode(t, err, InvalidArgument)
}
