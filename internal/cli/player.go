package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/pwnden/platform/internal/application"
)

// ExitError preserves a completed user command's exit code without an error banner.
type ExitError struct{ Code int }

func (e *ExitError) Error() string { return fmt.Sprintf("command exited %d", e.Code) }

func playerInputs(name, usage string, invocation Invocation) ([]string, *application.Service, error) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(invocation.Stderr)
	flags.Usage = func() { fmt.Fprintln(invocation.Stderr, "usage: pwnden "+usage) }
	if err := flags.Parse(invocation.Args); err != nil {
		return nil, nil, err
	}
	args := flags.Args()
	valid := (name == "list" && len(args) == 0) || (name == "submit" && len(args) == 2) || (name == "exec" && len(args) >= 3 && args[1] == "--")
	if !valid {
		flags.Usage()
		return nil, nil, fmt.Errorf("invalid %s arguments", name)
	}
	service, err := problemService(invocation.Repo)
	return args, service, err
}

func list(ctx context.Context, invocation Invocation) error {
	_, service, err := playerInputs("list", "list", invocation)
	if err != nil {
		return err
	}
	problems, err := service.List(ctx)
	if err != nil {
		return err
	}
	for _, problem := range problems {
		if _, err := fmt.Fprintf(invocation.Stdout, "%s\t%s\t%s\n", problem.Slug, problem.Category, problem.Title); err != nil {
			return err
		}
	}
	return nil
}

func execute(ctx context.Context, invocation Invocation) error {
	args, service, err := playerInputs("exec", "exec <problem> -- <command> [arguments]", invocation)
	if err != nil {
		return err
	}
	release, err := service.LockWorkspace()
	if err != nil {
		return err
	}
	defer release()
	result, err := service.Execute(ctx, args[0], args[2:])
	if err != nil {
		return err
	}
	if _, err := fmt.Fprint(invocation.Stdout, result.Stdout); err != nil {
		return err
	}
	if _, err := fmt.Fprint(invocation.Stderr, result.Stderr); err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &ExitError{Code: result.ExitCode}
	}
	return nil
}

func submit(ctx context.Context, invocation Invocation) error {
	args, service, err := playerInputs("submit", "submit <problem> <flag>", invocation)
	if err != nil {
		return err
	}
	result, err := service.Submit(ctx, args[0], args[1])
	if err != nil {
		return err
	}
	if result.Accepted {
		_, err := fmt.Fprintln(invocation.Stdout, "Correct.")
		return err
	}
	if _, err := fmt.Fprintln(invocation.Stdout, "Incorrect."); err != nil {
		return err
	}
	return &ExitError{Code: 1}
}
