package challenge

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFileChallenge(t *testing.T) {
	root := t.TempDir()
	writeContract(t, root, "version = 1\nsolve_network = 'default'\nsolve_timeout_seconds = 60\nattack_rejected_exit = 3\n")
	dir := filepath.Join(root, "challenges", "sample")
	if err := os.MkdirAll(filepath.Join(dir, "files"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "files", "data"), []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	metadata := `schema = 1
slug = "sample"
title = "Sample"
category = "rev"
files = ["files/data"]
[flag]
mode = "sha256"
sha256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
[solve]
image = "python:3.13.15-slim"
command = ["python3", "solve.py"]
`
	if err := os.WriteFile(filepath.Join(dir, "challenge.toml"), []byte(metadata), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(root, "sample")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Solve.TimeoutSeconds != 60 || loaded.Solve.Network != "default" {
		t.Fatalf("unexpected defaults: %+v", loaded.Solve)
	}
	dataFile := filepath.Join(dir, "files", "data")
	if err := os.Remove(dataFile); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root, "sample"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("startup accepted missing distribution: %v", err)
	}
	if _, err := LoadForStop(root, "sample"); err != nil {
		t.Fatalf("missing distribution blocked cleanup: %v", err)
	}
	if err := os.WriteFile(dataFile, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(t.TempDir(), filepath.Join(dir, "files", "outside")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	bad := strings.Replace(metadata, "files/data", "files/outside", 1)
	if err := os.WriteFile(filepath.Join(dir, "challenge.toml"), []byte(bad), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root, "sample"); err == nil || !strings.Contains(err.Error(), "outside challenges repository") {
		t.Fatalf("outside symlink accepted: %v", err)
	}
	if _, err := LoadForStop(root, "sample"); err == nil || !strings.Contains(err.Error(), "outside challenges repository") {
		t.Fatalf("cleanup accepted outside symlink: %v", err)
	}
	bad = strings.Replace(metadata, "files/data", "../../../outside-missing", 1)
	if err := os.WriteFile(filepath.Join(dir, "challenge.toml"), []byte(bad), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadForStop(root, "sample"); err == nil || !strings.Contains(err.Error(), "outside challenges repository") {
		t.Fatalf("cleanup accepted missing outside path: %v", err)
	}
}

func writeContract(t *testing.T, root, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "contract.toml"), []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadReadsExecutionFieldsWithoutAuthorFormatValidation(t *testing.T) {
	root := t.TempDir()
	writeContract(t, root, "version = 1\nsolve_network = 'isolated'\nsolve_timeout_seconds = 17\nattack_rejected_exit = 4\n")
	dir := filepath.Join(root, "challenges", "sample")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "challenge.toml"), []byte("schema = 1\nslug = 'sample'\nunknown = 1\ntitle = 123\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(root, "sample")
	if err != nil {
		t.Fatalf("consumer performed author format validation: %v", err)
	}
	if c.Solve.TimeoutSeconds != 17 || c.Solve.Network != "isolated" || c.Contract.AttackRejectedExit != 4 {
		t.Fatalf("owner defaults not read: %+v", c)
	}
}

func TestLoadRejectsUnsupportedVersionBeforeExecutionFieldDecoding(t *testing.T) {
	for _, repository := range []bool{false, true} {
		t.Run(fmt.Sprint(repository), func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "challenges", "sample")
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			definition := "version = 1\nsolve_network = 'default'\nsolve_timeout_seconds = 60\nattack_rejected_exit = 3\n"
			metadata := "schema = 4\n[solve]\nimage = 123\n"
			if repository {
				definition = "version = 4\nsolve_network = 123\n"
				metadata = "schema = 4\n"
			}
			writeContract(t, root, definition)
			if err := os.WriteFile(filepath.Join(dir, "challenge.toml"), []byte(metadata), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(root, "sample"); err == nil || !strings.Contains(err.Error(), "contract version 4") {
				t.Fatalf("unsupported contract not rejected first: %v", err)
			}
		})
	}
}
