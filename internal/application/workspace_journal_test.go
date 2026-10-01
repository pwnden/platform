package application

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/pwnden/platform/internal/testutil"
)

func TestWorkspaceOwnershipLockAndCrashRecovery(t *testing.T) {
	s, c := fixture(t, false)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	release, err := s.AcquireWorkspace(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.LockWorkspace(); err == nil {
		t.Fatal("same installation acquired twice")
	}
	other, _ := fixture(t, false)
	otherRelease, err := other.LockWorkspace()
	if err != nil {
		t.Fatal("independent installation locked:", err)
	}
	otherRelease()
	if err := s.TrackWorkspace(c.Slug); err != nil {
		t.Fatal(err)
	}
	if err := s.TrackWorkspace("../outside"); err == nil {
		t.Fatal("invalid problem recorded")
	}
	release()
	id := strings.Repeat("a", 12)
	calls := testutil.Docker(t,
		testutil.Reply{Match: []string{"ps", "-aq"}, Out: id + "\n"},
		testutil.Reply{Match: []string{"rm", "-f", "-v", id}},
	)
	next, _ := New(s.repo)
	release, err = next.AcquireWorkspace(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	seen := calls()
	if len(seen) != 2 || !strings.Contains(strings.Join(seen[0], " "), "label=pwnden.repository=") || !strings.Contains(strings.Join(seen[0], " "), "label=pwnden.problem=example") {
		t.Fatalf("cleanup scope: %v", seen)
	}
	data, err := os.ReadFile(next.journal.path)
	if err != nil || string(data) != "[]" {
		t.Fatalf("recovery journal: %s %v", data, err)
	}
}

func TestWorkspaceRecoveryFailureRetainsOwnership(t *testing.T) {
	s, _ := fixture(t, false)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	release, err := s.AcquireWorkspace(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.TrackWorkspace("example"); err != nil {
		t.Fatal(err)
	}
	release()
	testutil.Docker(t, testutil.Reply{Match: []string{"ps", "-aq"}, Code: 1, Err: "daemon unavailable"})
	next, _ := New(s.repo)
	_, err = next.AcquireWorkspace(context.Background())
	requireCode(t, err, CleanupFailed)
	var slugs []string
	data, err := os.ReadFile(next.journal.path)
	if err != nil || json.Unmarshal(data, &slugs) != nil || len(slugs) != 1 || slugs[0] != "example" {
		t.Fatalf("ownership lost: %s %v", data, err)
	}
	unlock, err := s.LockWorkspace()
	if err != nil {
		t.Fatal("failed recovery kept lock:", err)
	}
	unlock()
}
