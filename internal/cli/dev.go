package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/pwnden/platform/internal/application"
	"github.com/pwnden/platform/internal/httpapi"
)

func dev(ctx context.Context, invocation Invocation) error {
	flags := flag.NewFlagSet("dev", flag.ContinueOnError)
	flags.SetOutput(invocation.Stderr)
	flags.Usage = func() { fmt.Fprintln(invocation.Stderr, "usage: pwnden dev") }
	if err := flags.Parse(invocation.Args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("dev accepts no positional arguments")
	}
	if invocation.Repo == "" {
		return errors.New("dev requires a platform checkout; run its ./pwnden dev launcher")
	}
	service, err := application.New(invocation.Repo)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(invocation.Stdout, "Preparing local problems and execution environment..."); err != nil {
		return err
	}
	prepareCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	result, err := service.Prepare(prepareCtx)
	cancel()
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(invocation.Stdout, "Development ready: %d local problems.\n", result.ProblemCount); err != nil {
		return err
	}
	return httpapi.Serve(ctx, service, invocation.Stdout, invocation.Stderr)
}
