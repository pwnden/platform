// Package the platform executable and a fixed problem snapshot for end users.
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "package:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("package", flag.ContinueOnError)
	repo := flags.String("challenges", "../challenges", "problem checkout to package")
	output := flags.String("out", "dist", "directory for release packages")
	target := flags.String("target", runtime.GOOS+"/"+runtime.GOARCH, "target GOOS/GOARCH")
	official := flags.Bool("official", false, "fetch the official problem revision from catalog.lock")
	unpacked := flags.Bool("unpacked", false, "write the executable and catalog directly to the output directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	parts := strings.Split(*target, "/")
	if len(parts) != 2 || (parts[0] != "linux" && parts[0] != "darwin" && parts[0] != "windows") || (parts[1] != "amd64" && parts[1] != "arm64") {
		return fmt.Errorf("unsupported target %q", *target)
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if *official {
		conflict := false
		flags.Visit(func(f *flag.Flag) { conflict = conflict || f.Name == "challenges" })
		if conflict {
			return errors.New("--official selects catalog.lock and cannot be combined with --challenges")
		}
	}
	if err := os.MkdirAll(*output, 0755); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(*output, ".package-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	problemRepo := *repo
	if *official {
		data, err := os.ReadFile("catalog.lock")
		if err != nil {
			return fmt.Errorf("read official problem revision: %w", err)
		}
		problemRepo, err = fetchOfficial(stage, "https://github.com/pwnden/challenges.git", strings.TrimSpace(string(data)))
		if err != nil {
			return err
		}
	}
	revision, err := exec.Command("git", "-C", problemRepo, "rev-parse", "--verify", "HEAD^{commit}").Output()
	if err != nil {
		return fmt.Errorf("read problem revision: %w", err)
	}
	commit := strings.TrimSpace(string(revision))
	if len(commit) != 40 {
		return errors.New("expected a full problem commit hash")
	}
	if _, err := hex.DecodeString(commit); err != nil {
		return err
	}
	archive := filepath.Join(stage, "catalog.tar.gz")
	if err := snapshot(problemRepo, commit, archive); err != nil {
		return err
	}
	digest, err := checksum(archive)
	if err != nil {
		return err
	}
	binary := "pwnden"
	if parts[0] == "windows" {
		binary += ".exe"
	}
	ldflags := "-s -w -X github.com/pwnden/platform/internal/installation.catalogRevision=" + commit + " -X github.com/pwnden/platform/internal/installation.catalogSHA256=" + digest
	build := exec.Command("go", "build", "-trimpath", "-ldflags", ldflags, "-o", filepath.Join(stage, binary), "./cmd/pwnden")
	build.Env = targetEnvironment(parts[0], parts[1])
	build.Stdout, build.Stderr = os.Stdout, os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("build platform: %w", err)
	}
	metadata, err := json.MarshalIndent(struct {
		Revision string `json:"catalog_revision"`
		SHA256   string `json:"catalog_sha256"`
		Target   string `json:"target"`
	}{commit, digest, *target}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(stage, "distribution.json"), append(metadata, '\n'), 0644); err != nil {
		return err
	}
	if *unpacked {
		for _, name := range []string{binary, "catalog.tar.gz", "distribution.json"} {
			if err := os.Rename(filepath.Join(stage, name), filepath.Join(*output, name)); err != nil {
				return err
			}
		}
		fmt.Printf("%s (problem revision %s)\n", *output, commit)
		return nil
	}
	name := "pwnden-" + parts[0] + "-" + parts[1]
	suffix := ".tar.gz"
	if parts[0] == "windows" {
		suffix = ".zip"
	}
	artifact := filepath.Join(*output, name+suffix)
	if err := packageFiles(stage, artifact, name, []string{binary, "catalog.tar.gz", "distribution.json"}, parts[0] == "windows"); err != nil {
		return err
	}
	packageDigest, err := checksum(artifact)
	if err != nil {
		return err
	}
	if err := os.WriteFile(artifact+".sha256", []byte(packageDigest+"  "+filepath.Base(artifact)+"\n"), 0644); err != nil {
		return err
	}
	fmt.Printf("%s (problem revision %s)\n", artifact, commit)
	return nil
}

func targetEnvironment(goos, goarch string) []string {
	var env []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if key != "GOOS" && key != "GOARCH" && key != "CGO_ENABLED" {
			env = append(env, entry)
		}
	}
	return append(env, "GOOS="+goos, "GOARCH="+goarch, "CGO_ENABLED=0")
}

// fetchOfficial exports only the pinned commit. No problem code is executed.
func fetchOfficial(stage, origin, revision string) (string, error) {
	if _, err := hex.DecodeString(revision); err != nil || len(revision) != 40 {
		return "", errors.New("catalog.lock must contain a full problem commit hash")
	}
	repo := filepath.Join(stage, "official.git")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	for _, args := range [][]string{
		{"init", "--quiet", "--bare", repo},
		{"-C", repo, "fetch", "--quiet", "--depth=1", "--no-tags", "--", origin, revision},
		{"-C", repo, "update-ref", "HEAD", revision},
	} {
		cmd := exec.CommandContext(ctx, "git", args...)
		if diagnostic, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("acquire official problems at %s; the pinned commit must be published: %w: %s", revision, err, diagnostic)
		}
	}
	return repo, nil
}

func snapshot(repo, revision, output string) error {
	file, err := os.Create(output)
	if err != nil {
		return err
	}
	compressed := gzip.NewWriter(file)
	cmd := exec.Command("git", "-C", repo, "archive", "--format=tar", "--prefix=catalog/", revision)
	cmd.Stdout, cmd.Stderr = compressed, os.Stderr
	return errors.Join(cmd.Run(), compressed.Close(), file.Close())
}

func checksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func packageFiles(source, output, prefix string, names []string, windows bool) (err error) {
	file, err := os.Create(output)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	if windows {
		writer := zip.NewWriter(file)
		defer func() { err = errors.Join(err, writer.Close()) }()
		for _, name := range names {
			info, err := os.Stat(filepath.Join(source, name))
			if err != nil {
				return err
			}
			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}
			header.Name, header.Method = prefix+"/"+name, zip.Deflate
			entry, err := writer.CreateHeader(header)
			if err != nil {
				return err
			}
			if err := copyFile(entry, filepath.Join(source, name)); err != nil {
				return err
			}
		}
		return nil
	}
	compressed := gzip.NewWriter(file)
	defer func() { err = errors.Join(err, compressed.Close()) }()
	writer := tar.NewWriter(compressed)
	defer func() { err = errors.Join(err, writer.Close()) }()
	for _, name := range names {
		info, err := os.Stat(filepath.Join(source, name))
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = prefix + "/" + name
		if err := writer.WriteHeader(header); err != nil {
			return err
		}
		if err := copyFile(writer, filepath.Join(source, name)); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(destination io.Writer, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(destination, file)
	return err
}
