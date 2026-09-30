package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pwnden/platform/internal/cli"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "pwnden:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	commands, err := cli.New(cli.DefaultCommands()...)
	if err != nil {
		return err
	}
	signalCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	ctx, cancel := context.WithTimeout(signalCtx, 15*time.Minute)
	defer cancel()
	return commands.Execute(ctx, args, os.Stdout, os.Stderr)
}
