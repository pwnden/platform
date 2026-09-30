// Build and run a platform source checkout without host development SDKs.
package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/pwnden/platform/internal/checkout"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "checkout root is required")
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	l := checkout.Launcher{Root: os.Args[1], Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr}
	if err := l.Run(ctx, os.Args[2:]); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() >= 0 {
			os.Exit(exit.ExitCode())
		}
		if ctx.Err() != nil {
			os.Exit(130)
		}
		fmt.Fprintln(os.Stderr, "pwnden:", err)
		os.Exit(1)
	}
}
