package runtime

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
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
		if err := checkBudget(limits, cost, limits); !errors.Is(err, ErrResourceLimit) {
			t.Fatal("budget dimension not enforced", cost)
		}
	}
	t.Setenv("PWNDEN_RUNTIME_CPUS", "0")
	if _, err := runtimeBudget(16, 16<<30); err == nil {
		t.Fatal("invalid budget accepted")
	}
}

// A daemon too small to fit two 2-CPU containers side by side (here, 3) should
// drop the unset default to 1 CPU, the same adaptive ceiling the challenges
// author tool applies (tools/policy.py's cpu_limit()) -- otherwise a second
// workspace on a 3-CPU host is refused the moment the first one is still open.
func TestContainerCPUsAdaptsToSmallDaemon(t *testing.T) {
	previous := atomic.SwapInt64(&lastKnownDaemonCPUs, 0)
	t.Cleanup(func() { atomic.StoreInt64(&lastKnownDaemonCPUs, previous) })
	if got := containerCPUs(); got != 2e9 {
		t.Fatalf("default before any daemon is known: got %d, want 2e9", got)
	}
	testutil.Docker(t, testutil.Reply{Match: []string{"info", budgetInfoFormat},
		Out: `{"ID":"small-daemon","NCPU":3,"MemTotal":4294967296}`})
	if err := admitCreation(context.Background(), toolboxCost(), func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if got := containerCPUs(); got != 1e9 {
		t.Fatalf("after a 3-CPU daemon is known: got %d, want 1e9", got)
	}
	t.Setenv("PWNDEN_CONTAINER_CPUS", "2")
	if got := containerCPUs(); got != 2e9 {
		t.Fatalf("explicit PWNDEN_CONTAINER_CPUS=2 must override the adaptive default: got %d", got)
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

func TestAdmissionLeavesAuthorVerificationContainers(t *testing.T) {
	replies := []testutil.Reply{
		{Match: []string{"container", "ls", "{{json .}}"}, Out: `{"ID":"author","Names":"pwnden-author-0123456789ab-sample-app-1","Labels":"pwnden.managed=true"}
{"ID":"project","Names":"service","Labels":"pwnden.managed=true,com.docker.compose.project=pwnden-author-0123456789ab-sample"}
{"ID":"keeper","Names":"keeper","Labels":"pwnden.author-verification=owner,pwnden.kind=workspace"}`},
	}
	calls := testutil.Docker(t, replies...)
	if err := admitCreation(context.Background(), toolboxCost(), func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
	for _, args := range calls() {
		if strings.Contains(strings.Join(args, " "), "inspect") {
			t.Fatalf("author verification container counted: %v", args)
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
