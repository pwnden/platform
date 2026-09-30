package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pwnden/platform/internal/challenge"
	"github.com/pwnden/platform/internal/runtime"
	"github.com/pwnden/platform/internal/verify"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "pwnden:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("pwnden", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	repo := flags.String("repo", "", "path to challenges repository")
	if err := flags.Parse(args); err != nil {
		return err
	}
	rest := flags.Args()
	if *repo == "" || len(rest) != 2 {
		return fmt.Errorf("usage: pwnden --repo PATH <validate|run|verify|stop> <slug>")
	}
	load := challenge.Load
	if rest[0] == "stop" {
		load = challenge.LoadForStop
	}
	c, err := load(*repo, rest[1])
	if err != nil {
		return err
	}
	signalCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	ctx, cancel := context.WithTimeout(signalCtx, 15*time.Minute)
	defer cancel()
	switch rest[0] {
	case "validate":
		if c.Compose == "" {
			fmt.Printf("compatible file challenge %s (contract %d; %d files)\n", c.Slug, c.Contract.Version, len(c.Files))
			return nil
		}
		cfg, err := runtime.Validate(ctx, c, "pwnden{validation}", false)
		if err != nil {
			return err
		}
		if c.Patched != nil {
			if _, err := runtime.Validate(ctx, c, "pwnden{validation}", true); err != nil {
				return err
			}
		}
		fmt.Printf("compatible service challenge %s (contract %d; %d services)\n", c.Slug, c.Contract.Version, len(cfg.Services))
		return nil
	case "run":
		if err := runtime.Run(ctx, c); err != nil {
			return err
		}
		if c.Compose == "" {
			fmt.Printf("file challenge %s: %d files; run verify to check the solution\n", c.Slug, len(c.Files))
		} else {
			addresses, err := runtime.EndpointAddresses(ctx, c)
			if err != nil {
				return errors.Join(err, runtime.Stop(context.Background(), c))
			}
			fmt.Printf("running %s (project %s)\n", c.Slug, runtime.Project(c))
			for _, address := range addresses {
				suffix := ""
				if !address.Published {
					suffix = " (container network)"
				}
				fmt.Printf("  %s: %s%s\n", address.Name, address.URL, suffix)
			}
		}
		return nil
	case "verify":
		result, err := verify.Challenge(ctx, c)
		if err != nil {
			return err
		}
		fmt.Printf("verified %s: %s\n", c.Slug, result.Flag)
		if result.Patched {
			fmt.Println("patched attack failed; functional check passed")
		}
		return nil
	case "stop":
		if err := runtime.Stop(ctx, c); err != nil {
			return err
		}
		fmt.Printf("stopped %s\n", c.Slug)
		return nil
	default:
		return fmt.Errorf("unknown command %q", rest[0])
	}
}
