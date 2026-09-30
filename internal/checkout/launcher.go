// Package checkout owns the source checkout's build and launch lifecycle.
// Host scripts only build this native entry tool and pass through arguments.
package checkout

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type Launcher struct {
	Root           string
	Stdin          io.Reader
	Stdout, Stderr io.Writer
	execute        func(context.Context, string, []string, io.Writer, io.Writer) error
}

var buildName = regexp.MustCompile(`^(build|dev)\.[a-zA-Z0-9]+$`)
var errNoBuild = errors.New("no generated build reference")

func (l *Launcher) Run(ctx context.Context, args []string) error {
	root, err := filepath.Abs(l.Root)
	if err != nil {
		return err
	}
	l.Root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	if l.execute == nil {
		l.execute = l.command
	}
	if err := plainPath(filepath.Join(l.Root, "dist"), true); err != nil {
		return err
	}
	if len(args) > 0 && (args[0] == "setup" || args[0] == "dev") {
		if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
			_, err := fmt.Fprintf(l.Stderr, "usage: pwnden %s\n", args[0])
			return err
		}
		if len(args) != 1 {
			return fmt.Errorf("%s accepts no arguments; usage: pwnden %s", args[0], args[0])
		}
		if args[0] == "setup" {
			return l.setup(ctx)
		}
		return l.dev(ctx)
	}
	if len(args) == 0 || (len(args) == 1 && (args[0] == "--help" || args[0] == "-h")) {
		if _, err := l.active("active"); errors.Is(err, errNoBuild) {
			if _, err := l.active("development"); errors.Is(err, errNoBuild) {
				_, err := fmt.Fprintln(l.Stderr, "usage: pwnden <command> [arguments]\n\nCheckout commands:\n  setup   build and prepare official problems\n  dev     build and serve the local challenges checkout\n\nRun pwnden setup for player commands, or pwnden dev for development.")
				if err == nil && len(args) == 0 {
					err = errors.New("a command is required")
				}
				return err
			}
		}
	}
	// Source commands use the last development build; ordinary commands keep
	// using the official installation independently of development.
	reference := "active"
	if len(args) > 0 && (args[0] == "--repo" || strings.HasPrefix(args[0], "--repo=")) {
		reference = "development"
	}
	binary, err := l.active(reference)
	if errors.Is(err, errNoBuild) {
		other := "development"
		if reference == "development" {
			other = "active"
		}
		binary, err = l.active(other)
	}
	if errors.Is(err, errNoBuild) {
		return errors.New("run ./pwnden setup first, or ./pwnden dev for development")
	}
	if err != nil {
		return err
	}
	return l.execute(ctx, binary, args, l.Stdout, l.Stderr)
}

func (l *Launcher) setup(ctx context.Context) (err error) {
	if err := l.checkDocker(ctx); err != nil {
		return err
	}
	directory, err := l.build(ctx, "build.", "export")
	if err != nil {
		return err
	}
	activated := false
	defer func() {
		if !activated {
			err = errors.Join(err, os.RemoveAll(directory))
		}
	}()
	if err := l.execute(ctx, l.binary(directory), []string{"setup"}, l.Stdout, l.Stderr); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := l.activate("active", directory); err != nil {
		return err
	}
	activated = true
	_, err = fmt.Fprintln(l.Stdout, "Setup complete. Use ./pwnden serve and open the printed URL.")
	return err
}

func (l *Launcher) dev(ctx context.Context) error {
	workspace := filepath.Dir(l.Root)
	repo, err := filepath.EvalSymlinks(filepath.Join(workspace, "challenges"))
	if err != nil {
		return fmt.Errorf("dev requires the local challenges checkout next to platform: %w", err)
	}
	rel, err := filepath.Rel(workspace, repo)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return errors.New("development challenges must stay inside the workspace")
	}
	if _, err := os.Stat(filepath.Join(repo, "contract.toml")); err != nil {
		return fmt.Errorf("local challenges contract: %w", err)
	}
	if err := l.checkDocker(ctx); err != nil {
		return err
	}
	directory, err := l.build(ctx, "dev.", "development-export")
	if err != nil {
		return err
	}
	if err := l.activate("development", directory); err != nil {
		return errors.Join(err, os.RemoveAll(directory))
	}
	return l.execute(ctx, l.binary(directory), []string{"--repo", repo, "dev"}, l.Stdout, l.Stderr)
}

func (l *Launcher) checkDocker(ctx context.Context) error {
	var out strings.Builder
	if err := l.execute(ctx, "docker", []string{"info", "--format", "{{.OSType}}"}, &out, l.Stderr); err != nil {
		return fmt.Errorf("Docker must be installed and running: %w", err)
	}
	if strings.TrimSpace(out.String()) != "linux" {
		return errors.New("Docker must be running Linux containers")
	}
	for _, args := range [][]string{{"compose", "version", "--short"}, {"buildx", "version"}} {
		if err := l.execute(ctx, "docker", args, io.Discard, l.Stderr); err != nil {
			return err
		}
	}
	return nil
}

func (l *Launcher) build(ctx context.Context, prefix, target string) (directory string, err error) {
	dist := filepath.Join(l.Root, "dist")
	if err := os.MkdirAll(dist, 0755); err != nil {
		return "", err
	}
	directory, err = os.MkdirTemp(dist, prefix)
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, os.RemoveAll(directory))
			directory = ""
		}
	}()
	if _, err := fmt.Fprintln(l.Stdout, "Building platform with Docker; unchanged layers use the build cache..."); err != nil {
		return directory, err
	}
	args := []string{"buildx", "build", "--file", filepath.Join(l.Root, "Dockerfile.bootstrap"), "--target", target,
		"--build-arg", "PWNDEN_TARGET=" + runtime.GOOS + "/" + runtime.GOARCH,
		"--output", filepath.Join(directory, "payload"), l.Root}
	if err = l.execute(ctx, "docker", args, l.Stdout, l.Stderr); err != nil {
		return directory, err
	}
	if err = plainPath(filepath.Join(directory, "payload"), false); err != nil {
		return directory, err
	}
	if err = plainPath(l.binary(directory), false); err != nil {
		return directory, err
	}
	return directory, nil
}

func (l *Launcher) binary(directory string) string {
	name := "pwnden"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(directory, "payload", name)
}

func (l *Launcher) active(reference string) (string, error) {
	path := filepath.Join(l.Root, "dist", reference)
	if err := plainPath(path, false); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", errNoBuild
		}
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(string(data))
	if !buildName.MatchString(name) {
		return "", errors.New("invalid generated build reference; run ./pwnden setup or ./pwnden dev")
	}
	directory := filepath.Join(l.Root, "dist", name)
	for _, path := range []string{directory, filepath.Join(directory, "payload"), l.binary(directory)} {
		if err := plainPath(path, false); err != nil {
			return "", err
		}
	}
	return l.binary(directory), nil
}

func (l *Launcher) activate(reference, directory string) error {
	dist := filepath.Join(l.Root, "dist")
	path := filepath.Join(dist, reference)
	if err := plainPath(path, true); err != nil {
		return err
	}
	file, err := os.CreateTemp(dist, ".activate-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, writeErr := fmt.Fprintln(file, filepath.Base(directory))
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func plainPath(path string, missing bool) error {
	info, err := os.Lstat(path)
	if missing && errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("generated paths must stay inside this checkout: %s", filepath.Base(path))
	}
	return nil
}

func (l *Launcher) command(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = l.Stdin, stdout, stderr
	// Allow the server to close sessions and finish independent cleanup.
	cmd.Cancel = func() error {
		// Windows broadcasts console Ctrl+C to attached children. Wait for their
		// cleanup instead of replacing that graceful signal with an immediate kill.
		if runtime.GOOS == "windows" {
			return nil
		}
		return cmd.Process.Signal(os.Interrupt)
	}
	cmd.WaitDelay = 45 * time.Second
	return cmd.Run()
}
