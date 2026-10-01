package checkout

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var catalogRevision = regexp.MustCompile(`^[a-f0-9]{40}$`)

func (l *Launcher) catalog(ctx context.Context, args []string) (err error) {
	usage := func() {
		fmt.Fprintln(l.Stderr, "usage: pwnden catalog update [--revision COMMIT]\n\nPin challenges/main by default, or a published full 40-character commit.\nUpdates catalog.lock; run pwnden setup to install the selected catalog.")
	}
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		usage()
		return nil
	}
	if len(args) == 0 || args[0] != "update" {
		usage()
		return errors.New("catalog requires the update subcommand")
	}
	flags := flag.NewFlagSet("catalog update", flag.ContinueOnError)
	flags.SetOutput(l.Stderr)
	flags.Usage = usage
	revision := flags.String("revision", "", "published full commit; default challenges/main")
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	explicit := false
	flags.Visit(func(f *flag.Flag) { explicit = explicit || f.Name == "revision" })
	if flags.NArg() != 0 || (explicit && !catalogRevision.MatchString(*revision)) {
		usage()
		return errors.New("catalog update accepts only --revision with a full lowercase commit hash")
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	path := filepath.Join(l.Root, "catalog.lock")
	before, mode, err := readCatalogLock(path)
	if err != nil {
		return err
	}
	if err := l.checkDocker(ctx); err != nil {
		return err
	}
	dist := filepath.Join(l.Root, "dist")
	if err := os.MkdirAll(dist, 0755); err != nil {
		return err
	}
	directory, err := os.MkdirTemp(dist, ".catalog-*")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(directory)) }()
	ref := "refs/heads/main"
	if *revision != "" {
		ref = *revision
	}
	if _, err := fmt.Fprintln(l.Stdout, "Checking published problems..."); err != nil {
		return err
	}
	output := filepath.Join(directory, "payload")
	// A moving main branch must be fetched on every update, including retries.
	build := []string{"buildx", "build", "--file", filepath.Join(l.Root, "Dockerfile.bootstrap"),
		"--target", "catalog-export", "--no-cache", "--build-arg", "PWNDEN_CATALOG_REF=" + ref,
		"--output", output, l.Root}
	if err := l.execute(ctx, "docker", build, l.Stderr, l.Stderr); err != nil {
		return fmt.Errorf("catalog acquisition failed; the selected commit must be published: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := plainPath(output, false); err != nil {
		return err
	}
	selected, _, err := readCatalogLock(filepath.Join(output, "catalog.lock"))
	if err != nil {
		return err
	}
	commit := strings.TrimSpace(string(selected))
	if *revision != "" && commit != *revision {
		return errors.New("published catalog did not match the requested commit")
	}
	current, _, err := readCatalogLock(path)
	if err != nil {
		return err
	}
	if !bytes.Equal(current, before) {
		return errors.New("catalog.lock changed during acquisition; retry with the current file")
	}
	if strings.TrimSpace(string(before)) == commit {
		_, err := fmt.Fprintf(l.Stdout, "Catalog already pinned to %s.\n", commit)
		return err
	}
	file, err := os.CreateTemp(l.Root, ".catalog-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	writeErr := file.Chmod(mode)
	if writeErr == nil {
		_, writeErr = fmt.Fprintln(file, commit)
	}
	if err := errors.Join(writeErr, file.Close(), ctx.Err()); err != nil {
		return err
	}
	if err := os.Rename(file.Name(), path); err != nil {
		return err
	}
	_, err = fmt.Fprintf(l.Stdout, "Catalog pinned to %s. Commit catalog.lock, then run ./pwnden setup.\n", commit)
	return err
}

func readCatalogLock(path string) ([]byte, os.FileMode, error) {
	if err := plainPath(path, false); err != nil {
		return nil, 0, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, 0, err
	}
	if !info.Mode().IsRegular() {
		return nil, 0, errors.New("catalog.lock must be a regular file")
	}
	content, err := io.ReadAll(io.LimitReader(file, 65))
	if err != nil {
		return nil, 0, err
	}
	if len(content) > 64 || !catalogRevision.MatchString(strings.TrimSpace(string(content))) {
		return nil, 0, errors.New("catalog.lock must contain one full lowercase commit hash")
	}
	return content, info.Mode().Perm(), nil
}
