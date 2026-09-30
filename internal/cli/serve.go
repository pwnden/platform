package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/pwnden/platform/internal/application"
	"github.com/pwnden/platform/internal/httpapi"
)

func serve(ctx context.Context, invocation Invocation) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(invocation.Stderr)
	flags.Usage = func() { fmt.Fprintln(invocation.Stderr, "usage: pwnden serve") }
	if err := flags.Parse(invocation.Args); err != nil {
		return err
	}
	if flags.NArg() != 0 || invocation.Repo != "" {
		flags.Usage()
		return errors.New("serve uses the managed installation and accepts no repository or positional arguments")
	}
	service, err := application.Open()
	if err != nil {
		return err
	}
	if _, err := service.List(ctx); err != nil {
		return err
	}
	return httpapi.Serve(ctx, service, invocation.Stdout, invocation.Stderr)
}
