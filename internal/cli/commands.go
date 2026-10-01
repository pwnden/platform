package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/pwnden/platform/internal/application"
)

func DefaultCommands() []Command {
	return []Command{
		{Name: "setup", Summary: "prepare the bundled problems and execution environment", Run: setup},
		{Name: "dev", Summary: "start the development player with Vue HMR", Run: dev, Persistent: true},
		{Name: "serve", Summary: "start the local player web and API server", Run: serve, Persistent: true},
		{Name: "list", Summary: "list available problems", Run: list},
		{Name: "exec", Summary: "run a command inside a problem toolbox", Run: execute},
		{Name: "submit", Summary: "check a flag you found", Run: submit},
		{Name: "validate", Summary: "check compatibility and execution policy", Run: validate},
		{Name: "run", Summary: "start a problem and show its endpoints", Run: run},
		{Name: "verify", Summary: "check the automatic solution and optional patch", Run: verify},
		{Name: "stop", Summary: "remove problem resources and saved state", Run: stop},
	}
}

func challengeInputs(name string, invocation Invocation) (string, *application.Service, error) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(invocation.Stderr)
	flags.Usage = func() {
		fmt.Fprintf(invocation.Stderr, "usage: pwnden [--repo PATH] %s <slug>\n", name)
		flags.PrintDefaults()
	}
	if err := flags.Parse(invocation.Args); err != nil {
		return "", nil, err
	}
	if flags.NArg() != 1 {
		flags.Usage()
		return "", nil, fmt.Errorf("%s requires exactly one problem slug", name)
	}
	service, err := problemService(invocation.Repo)
	return flags.Arg(0), service, err
}

func problemService(repo string) (*application.Service, error) {
	if repo == "" {
		return application.Open()
	}
	return application.New(repo)
}

func setup(ctx context.Context, invocation Invocation) error {
	flags := flag.NewFlagSet("setup", flag.ContinueOnError)
	flags.SetOutput(invocation.Stderr)
	flags.Usage = func() { fmt.Fprintln(invocation.Stderr, "usage: pwnden setup") }
	if err := flags.Parse(invocation.Args); err != nil {
		return err
	}
	if flags.NArg() != 0 || invocation.Repo != "" {
		flags.Usage()
		return errors.New("setup uses the bundled distribution and accepts no repository or positional arguments")
	}
	service, err := application.NewSetup()
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(invocation.Stdout, "Preparing problems and execution environment..."); err != nil {
		return err
	}
	result, err := service.Setup(ctx)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(invocation.Stdout, "Ready: %d problems. Start with pwnden serve.\n", result.ProblemCount)
	return err
}

func validate(ctx context.Context, invocation Invocation) error {
	slug, service, err := challengeInputs("validate", invocation)
	if err != nil {
		return err
	}
	result, err := service.Validate(ctx, slug)
	if err != nil {
		return err
	}
	if result.Kind == application.KindFile {
		_, err = fmt.Fprintf(invocation.Stdout, "compatible file challenge %s (contract %d; %d files)\n", result.Slug, result.ContractVersion, result.FileCount)
	} else {
		_, err = fmt.Fprintf(invocation.Stdout, "compatible service challenge %s (contract %d; %d services)\n", result.Slug, result.ContractVersion, result.ServiceCount)
	}
	return err
}

func run(ctx context.Context, invocation Invocation) (err error) {
	slug, service, err := challengeInputs("run", invocation)
	if err != nil {
		return err
	}
	release, err := service.LockWorkspace()
	if err != nil {
		return err
	}
	defer release()
	result, err := service.Run(ctx, slug)
	if err != nil {
		return err
	}
	if result.Kind == application.KindFile {
		_, err = fmt.Fprintf(invocation.Stdout, "file challenge %s: %d files; use pwnden exec to run toolbox commands\n", result.Slug, result.FileCount)
		return err
	}
	// A failed command must not leave a new run owned by nobody. Existing runs
	// fail above, before this cleanup is installed.
	defer func() {
		if err != nil {
			_, cleanupErr := service.Stop(context.WithoutCancel(ctx), slug)
			err = errors.Join(err, cleanupErr)
		}
	}()
	if _, err := fmt.Fprintf(invocation.Stdout, "running %s (project %s)\n", result.Slug, result.Project); err != nil {
		return err
	}
	for _, endpoint := range result.Endpoints {
		suffix := ""
		if !endpoint.Published {
			suffix = " (container network)"
		}
		if _, err := fmt.Fprintf(invocation.Stdout, "  %s: %s%s\n", endpoint.Name, endpoint.URL, suffix); err != nil {
			return err
		}
	}
	return nil
}

func verify(ctx context.Context, invocation Invocation) error {
	slug, service, err := challengeInputs("verify", invocation)
	if err != nil {
		return err
	}
	release, err := service.LockWorkspace()
	if err != nil {
		return err
	}
	defer release()
	result, err := service.Verify(ctx, slug)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(invocation.Stdout, "verified %s: %s\n", result.Slug, result.Flag); err != nil {
		return err
	}
	if result.Patched {
		_, err = fmt.Fprintln(invocation.Stdout, "patched attack failed; functional check passed")
	}
	return err
}

func stop(ctx context.Context, invocation Invocation) error {
	slug, service, err := challengeInputs("stop", invocation)
	if err != nil {
		return err
	}
	release, err := service.LockWorkspace()
	if err != nil {
		return err
	}
	defer release()
	result, err := service.Stop(ctx, slug)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(invocation.Stdout, "stopped %s\n", result.Slug)
	return err
}
