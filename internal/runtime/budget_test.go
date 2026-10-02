package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/pwnden/platform/internal/testutil"
)

func TestRuntimeBudget(t *testing.T) {
	limits, err := runtimeBudget(4, 4<<30)
	if err != nil || limits.CPUs != 4e9 || limits.Memory != 4<<30 {
		t.Fatalf("daemon clipping: %+v %v", limits, err)
	}
	if err := checkBudget(toolboxCost(), toolboxCost(), limits); err != nil {
		t.Fatal(err)
	}
	for _, cost := range []resourceCost{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}, {0, 0, 0, 1}} {
		if err := checkBudget(limits, cost, limits); err == nil {
			t.Fatal("budget dimension not enforced", cost)
		}
	}
	t.Setenv("PWNDEN_RUNTIME_CPUS", "0")
	if _, err := runtimeBudget(16, 16<<30); err == nil {
		t.Fatal("invalid budget accepted")
	}
}

func TestAdmissionCountsCreatedContainersAndLeavesOtherApps(t *testing.T) {
	replies := []testutil.Reply{
		{Match: []string{"container", "ls", "{{json .}}"}, Out: `{"ID":"managed","Names":"created-tool","Labels":"pwnden.managed=true"}
{"ID":"unrelated","Names":"postgres","Labels":"other.app=true"}`},
		{Match: []string{"container", "inspect", "managed"}, Out: `{"NanoCpus":2000000000,"Memory":2147483648,"PidsLimit":256}`},
	}
	calls := testutil.Docker(t, replies...)
	t.Setenv("PWNDEN_RUNTIME_MEMORY_MIB", "2048")
	created := false
	err := admitCreation(context.Background(), toolboxCost(), func(context.Context) error { created = true; return nil })
	if err == nil || !strings.Contains(err.Error(), "budget exceeded") || created {
		t.Fatalf("creation not refused: %v", err)
	}
	for _, args := range calls() {
		if strings.Contains(strings.Join(args, " "), "inspect unrelated") {
			t.Fatal("unrelated app counted")
		}
	}
}

func TestAdmissionIgnoresConfirmedRemovedContainer(t *testing.T) {
	testutil.Docker(t,
		testutil.Reply{Match: []string{"container", "ls", "{{json .}}"}, Out: `{"ID":"gone","Names":"pwnden-tool","Labels":"pwnden.managed=true"}`},
		testutil.Reply{Match: []string{"container", "inspect", "gone"}, Code: 1},
	)
	if err := admitCreation(context.Background(), toolboxCost(), func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
}
