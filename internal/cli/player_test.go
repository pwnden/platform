package cli

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/pwnden/platform/internal/testutil"
)

func TestPlayerCommands(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	repo := fileRepository(t)
	testutil.Docker(t,
		testutil.Reply{Match: []string{"image", "inspect"}},
		testutil.Reply{Match: []string{"run"}, Out: "locked\n", Err: "diagnostic\n", Code: 1},
	)
	r, err := New(DefaultCommands()...)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		args               []string
		output, diagnostic string
		code               int
	}{
		{[]string{"list"}, "example\t\texample\n", "", 0},
		{[]string{"submit", "example", "pwnden{test}"}, "Correct.\n", "", 0},
		{[]string{"submit", "example", "wrong"}, "Incorrect.\n", "", 1},
		{[]string{"exec", "example", "--", "python3", "files/checker.py", "wrong"}, "locked\n", "diagnostic\n", 1},
	} {
		var out, stderr bytes.Buffer
		args := append([]string{"--repo", repo}, test.args...)
		err := r.Execute(context.Background(), args, &out, &stderr)
		var exit *ExitError
		if test.code == 0 && err != nil || test.code != 0 && (!errors.As(err, &exit) || exit.Code != test.code) {
			t.Fatalf("player exit: %v", err)
		}
		if out.String() != test.output || stderr.String() != test.diagnostic {
			t.Fatalf("player output: %q %q", out.String(), stderr.String())
		}
	}
	for _, args := range [][]string{{"list", "extra"}, {"submit", "example"}, {"exec", "example", "command"}} {
		if err := r.Execute(context.Background(), append([]string{"--repo", repo}, args...), &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
			t.Fatalf("invalid player args accepted: %v", args)
		}
	}
}
