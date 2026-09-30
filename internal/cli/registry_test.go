package cli

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestCommandRegistration(t *testing.T) {
	handler := func(context.Context, Invocation) error { return nil }
	for _, commands := range [][]Command{
		{{Name: "", Run: handler}},
		{{Name: "-invalid", Run: handler}},
		{{Name: "two words", Run: handler}},
		{{Name: "missing-handler"}},
		{{Name: "run", Run: handler}, {Name: "run", Run: handler}},
	} {
		if _, err := New(commands...); err == nil {
			t.Fatalf("invalid registry accepted: %+v", commands)
		}
	}
}

func TestAdditionalCommandOwnsArgumentsAndOptions(t *testing.T) {
	// This extra command accepts no slug, its own flag, and multiple positional args.
	// The dispatcher needs no changes to accommodate it.
	called := false
	command := Command{Name: "example", Summary: "exercise an additional command", Run: func(ctx context.Context, invocation Invocation) error {
		called = true
		if invocation.Repo != "repository with spaces" || ctx.Err() != nil {
			t.Fatalf("invocation lost: %+v %v", invocation, ctx.Err())
		}
		flags := flag.NewFlagSet("example", flag.ContinueOnError)
		label := flags.String("label", "", "output label")
		if err := flags.Parse(invocation.Args); err != nil {
			return err
		}
		_, err := fmt.Fprintf(invocation.Stdout, "%s:%s", *label, strings.Join(flags.Args(), ","))
		return err
	}}
	commands := append(DefaultCommands(), command)
	r, err := New(commands...)
	if err != nil {
		t.Fatal(err)
	}
	// Registration copies the slice so later caller edits cannot change dispatch.
	commands[len(commands)-1].Name = "changed"
	var out, stderr bytes.Buffer
	err = r.Execute(context.Background(), []string{"--repo", "repository with spaces", "example", "--label", "ok", "first", "second"}, &out, &stderr)
	if err != nil || !called || out.String() != "ok:first,second" {
		t.Fatalf("extension dispatch: %q %v", out.String(), err)
	}
}

func TestHelpAndDispatchErrors(t *testing.T) {
	r, err := New(DefaultCommands()...)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		args []string
		pass bool
		text string
	}{
		{[]string{"--help"}, true, "Commands:"},
		{[]string{"validate", "--help"}, true, "validate <slug>"},
		{nil, false, "a command is required"},
		{[]string{"unknown", "missing"}, false, "unknown command"},
		{[]string{"validate"}, false, "exactly one problem slug"},
		{[]string{"validate", "first", "second"}, false, "exactly one problem slug"},
		{[]string{"--unknown"}, false, "flag provided but not defined"},
		{[]string{"validate", "--unknown", "slug"}, false, "flag provided but not defined"},
		{[]string{"validate", "slug"}, false, "repository path is required"},
	} {
		t.Run(strings.Join(test.args, "/"), func(t *testing.T) {
			var out, stderr bytes.Buffer
			err := r.Execute(context.Background(), test.args, &out, &stderr)
			if (err == nil) != test.pass {
				t.Fatalf("unexpected result: %v", err)
			}
			text := stderr.String()
			if err != nil {
				text += err.Error()
			}
			if !strings.Contains(text, test.text) || out.Len() != 0 {
				t.Fatalf("response: stdout=%q stderr=%q error=%v", out.String(), stderr.String(), err)
			}
		})
	}
}

func TestCommandErrorAndContextPreserved(t *testing.T) {
	cause := errors.New("handler failed")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, err := New(Command{Name: "example", Run: func(got context.Context, _ Invocation) error {
		if got != ctx {
			t.Fatal("caller context replaced")
		}
		return errors.Join(cause, got.Err(), flag.ErrHelp)
	}})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Execute(ctx, []string{"example"}, io.Discard, io.Discard)
	if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
		t.Fatalf("error causes lost: %v", err)
	}
}
