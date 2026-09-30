package main

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestOfficialAcquisitionPinsRevision(t *testing.T) {
	origin := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", origin}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "--quiet")
	git("config", "user.name", "Fixture")
	git("config", "user.email", "fixture@example.invalid")
	git("config", "commit.gpgsign", "false")
	file := filepath.Join(origin, "contract.toml")
	if err := os.WriteFile(file, []byte("version = 1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	git("add", "contract.toml")
	git("-c", "core.hooksPath=/dev/null", "commit", "--quiet", "-m", "fixture")
	revision := git("rev-parse", "HEAD")
	if err := os.WriteFile(file, []byte("version = 2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	git("add", "contract.toml")
	git("-c", "core.hooksPath=/dev/null", "commit", "--quiet", "-m", "move branch")
	stage := t.TempDir()
	repo, err := fetchOfficial(stage, "file://"+filepath.ToSlash(origin), revision)
	if err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(stage, "catalog.tar.gz")
	if err := snapshot(repo, revision, archive); err != nil {
		t.Fatal(err)
	}
	input, err := os.Open(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	compressed, err := gzip.NewReader(input)
	if err != nil {
		t.Fatal(err)
	}
	defer compressed.Close()
	reader := tar.NewReader(compressed)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			t.Fatal("contract missing from official archive")
		}
		if err != nil {
			t.Fatal(err)
		}
		if header.Name == "catalog/contract.toml" {
			data, err := io.ReadAll(reader)
			if err != nil || string(data) != "version = 1\n" {
				t.Fatalf("pinned content: %q %v", data, err)
			}
			break
		}
	}
	if _, err := fetchOfficial(t.TempDir(), "file://"+filepath.ToSlash(origin), strings.Repeat("0", 40)); err == nil {
		t.Fatal("unpublished revision was accepted")
	}
}

func TestOfficialAcquisitionRejectsInvalidIdentity(t *testing.T) {
	for _, revision := range []string{"main", "--help", strings.Repeat("z", 40)} {
		stage := t.TempDir()
		if _, err := fetchOfficial(stage, "unreachable", revision); err == nil {
			t.Fatalf("accepted %q", revision)
		}
		entries, err := os.ReadDir(stage)
		if err != nil || len(entries) != 0 {
			t.Fatalf("invalid identity touched the checkout: %v %v", entries, err)
		}
	}
}

func TestOfficialModeRejectsConflictingCheckout(t *testing.T) {
	if err := run([]string{"--official", "--challenges", "local"}); err == nil {
		t.Fatal("official acquisition accepted a checkout override")
	}
}
