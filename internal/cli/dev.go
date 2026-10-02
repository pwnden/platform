package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/pwnden/platform/internal/application"
	"github.com/pwnden/platform/internal/httpapi"
)

func dev(ctx context.Context, invocation Invocation) error {
	flags := flag.NewFlagSet("dev", flag.ContinueOnError)
	flags.SetOutput(invocation.Stderr)
	web := flags.String("web", "", "launcher-owned loopback frontend URL")
	nonce := flags.String("nonce", "", "launcher-owned frontend style nonce")
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
	frontend, err := httpapi.DevelopmentFrontend(*web, *nonce)
	if err != nil {
		return err
	}
	service, err := application.New(invocation.Repo)
	if err != nil {
		return err
	}
	store, err := playerStorage(ctx, service, invocation.Repo)
	if err != nil {
		return err
	}
	defer store.Close()
	problems, err := service.List(ctx)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(invocation.Stdout, "Development ready: %d local problems; Vue HMR enabled.\n", len(problems)); err != nil {
		return err
	}
	return httpapi.ServeDevelopment(ctx, service, invocation.Stdout, invocation.Stderr, frontend)
}
