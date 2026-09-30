// Package cli adapts registered commands to the platform application interface.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"regexp"
)

type Invocation struct {
	Repo   string
	Args   []string
	Stdout io.Writer
	Stderr io.Writer
}

// Command owns its argument parsing, help, and response formatting.
type Command struct {
	Name    string
	Summary string
	Run     func(context.Context, Invocation) error
}

type Registry struct {
	commands []Command
}

var commandName = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

func New(commands ...Command) (*Registry, error) {
	seen := make(map[string]bool, len(commands))
	for _, command := range commands {
		if !commandName.MatchString(command.Name) || command.Run == nil {
			return nil, fmt.Errorf("invalid command registration %q", command.Name)
		}
		if seen[command.Name] {
			return nil, fmt.Errorf("duplicate command registration %q", command.Name)
		}
		seen[command.Name] = true
	}
	return &Registry{commands: append([]Command(nil), commands...)}, nil
}

// Execute parses global options only; the selected command parses its own args.
func (r *Registry) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("pwnden", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", "", "path to challenges repository")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "usage: pwnden [--repo PATH] <command> [arguments]")
		fmt.Fprintln(stderr, "\nCommands:")
		for _, command := range r.commands {
			fmt.Fprintf(stderr, "  %-10s %s\n", command.Name, command.Summary)
		}
		fmt.Fprintln(stderr, "\nGlobal options:")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	remaining := flags.Args()
	if len(remaining) == 0 {
		flags.Usage()
		return errors.New("a command is required")
	}
	for _, command := range r.commands {
		if command.Name == remaining[0] {
			err := command.Run(ctx, Invocation{Repo: *repo, Args: remaining[1:], Stdout: stdout, Stderr: stderr})
			if err == flag.ErrHelp {
				return nil
			}
			return err
		}
	}
	return fmt.Errorf("unknown command %q", remaining[0])
}
