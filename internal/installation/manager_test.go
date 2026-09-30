package installation

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type archiveEntry struct {
	name, data, target string
	kind               byte
}

func bundle(t *testing.T, entries ...archiveEntry) []byte {
	t.Helper()
	var data bytes.Buffer
	compressed := gzip.NewWriter(&data)
	writer := tar.NewWriter(compressed)
	for _, entry := range entries {
		kind := entry.kind
		if kind == 0 {
			kind = tar.TypeReg
		}
		header := &tar.Header{Name: entry.name, Mode: 0644, Typeflag: kind, Linkname: entry.target}
		if kind == tar.TypeReg {
			header.Size = int64(len(entry.data))
		}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(entry.data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := errors.Join(writer.Close(), compressed.Close()); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func manager(t *testing.T, data []byte) *Manager {
	t.Helper()
	dir := t.TempDir()
	archive := filepath.Join(dir, ArchiveName)
	if err := os.WriteFile(archive, data, 0600); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(data)
	return &Manager{DataDir: filepath.Join(dir, "installation"), Distribution: Distribution{
		Archive: archive, Revision: strings.Repeat("1", 40), SHA256: hex.EncodeToString(hash[:]),
	}}
}

func TestInstallActivateAndRepeat(t *testing.T) {
	m := manager(t, bundle(t, archiveEntry{name: "catalog/contract.toml", data: "version=1\n"}, archiveEntry{name: "catalog/files/one.txt", data: "content"}))
	if _, err := m.Resolve(); !errors.Is(err, ErrSetupRequired) {
		t.Fatalf("unset installation: %v", err)
	}
	pending, err := m.Stage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Resolve(); !errors.Is(err, ErrSetupRequired) {
		t.Fatal("staged installation became active before preparation")
	}
	if err := pending.Activate(); err != nil {
		t.Fatal(err)
	}
	if err := pending.Discard(); err != nil {
		t.Fatal(err)
	}
	root, err := m.Resolve()
	if err != nil || root != pending.Root {
		t.Fatalf("installed root: %q %v", root, err)
	}
	if err := os.Remove(m.Distribution.Archive); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "files", "one.txt"), []byte("user edit"), 0644); err != nil {
		t.Fatal(err)
	}
	repeated, err := m.Stage(context.Background())
	if err != nil || repeated.Root != root {
		t.Fatalf("repeat install: %+v %v", repeated, err)
	}
	if err := repeated.Discard(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "files", "one.txt"))
	if err != nil || string(data) != "user edit" {
		t.Fatal("repeat setup replaced existing data")
	}
}

func TestFailedPreparationDiscardsOnlyNewInstall(t *testing.T) {
	m := manager(t, bundle(t, archiveEntry{name: "catalog/contract.toml", data: "version=1\n"}))
	pending, err := m.Stage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := pending.Discard(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(pending.Root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("discard left an unprepared catalog: %v", err)
	}
	if _, err := m.Resolve(); !errors.Is(err, ErrSetupRequired) {
		t.Fatalf("failed preparation activated: %v", err)
	}
}

func TestChecksumAndCancellation(t *testing.T) {
	for _, test := range []string{"checksum", "cancel", "revision"} {
		t.Run(test, func(t *testing.T) {
			m := manager(t, bundle(t, archiveEntry{name: "catalog/contract.toml", data: "version=1\n"}))
			ctx := context.Background()
			if test == "checksum" {
				m.Distribution.SHA256 = strings.Repeat("0", 64)
			} else if test == "revision" {
				m.Distribution.Revision = "../outside"
			} else {
				canceled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = canceled
			}
			if _, err := m.Stage(ctx); err == nil {
				t.Fatal("invalid distribution accepted")
			}
			if _, err := m.Resolve(); !errors.Is(err, ErrSetupRequired) {
				t.Fatalf("failed install published state: %v", err)
			}
		})
	}
}

func TestArchiveBoundary(t *testing.T) {
	for _, entries := range [][]archiveEntry{
		{{name: "../outside", data: "bad"}},
		{{name: "/outside", data: "bad"}},
		{{name: "catalog/../outside", data: "bad"}},
		{{name: "catalog/C:outside", data: "bad"}},
		{{name: "catalog/file", data: "one"}, {name: "catalog/file", data: "two"}},
		{{name: "catalog/escape", kind: tar.TypeSymlink, target: "../../outside"}},
		{{name: "catalog/hard", kind: tar.TypeLink, target: "catalog/file"}},
		{{name: "catalog/a/b", kind: tar.TypeSymlink, target: "../.."}, {name: "catalog/a/b/escape", kind: tar.TypeSymlink, target: "../../outside"}, {name: "catalog/a/b/escape/child", kind: tar.TypeSymlink, target: "../file"}},
	} {
		t.Run(entries[0].name, func(t *testing.T) {
			entries = append(entries, archiveEntry{name: "catalog/contract.toml", data: "version=1\n"})
			m := manager(t, bundle(t, entries...))
			if _, err := m.Stage(context.Background()); err == nil {
				t.Fatal("unsafe archive accepted")
			}
			if _, err := m.Resolve(); !errors.Is(err, ErrSetupRequired) {
				t.Fatal("unsafe archive activated")
			}
		})
	}
}

func TestInternalSymlinkAndReadableModes(t *testing.T) {
	m := manager(t, bundle(t,
		archiveEntry{name: "catalog/contract.toml", data: "version=1\n"},
		archiveEntry{name: "catalog/files/file.txt", data: "content"},
		archiveEntry{name: "catalog/alias.txt", kind: tar.TypeSymlink, target: "files/file.txt"},
	))
	pending, err := m.Stage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer pending.Discard()
	data, err := os.ReadFile(filepath.Join(pending.Root, "alias.txt"))
	if err != nil || string(data) != "content" {
		t.Fatalf("internal symlink: %q %v", data, err)
	}
	for _, path := range []string{pending.Root, filepath.Join(pending.Root, "files"), filepath.Join(pending.Root, "files", "file.txt")} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm()&0004 == 0 || (info.IsDir() && info.Mode().Perm()&0001 == 0) {
			t.Fatalf("toolbox cannot read %s: %v", path, err)
		}
	}
}

func TestManagedRootCannotBeRedirectedOutside(t *testing.T) {
	m := manager(t, bundle(t, archiveEntry{name: "catalog/contract.toml", data: "version=1\n"}))
	pending, err := m.Stage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := pending.Activate(); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "contract.toml"), []byte("version=1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(pending.Root, pending.Root+"-original"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, pending.Root); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Resolve(); err == nil {
		t.Fatal("managed root resolved outside installation")
	}
	if _, err := m.Stage(context.Background()); err == nil {
		t.Fatal("repeat setup reused an outside snapshot")
	}
}

func TestNewCatalogCannotFollowOutsideParent(t *testing.T) {
	m := manager(t, bundle(t, archiveEntry{name: "catalog/contract.toml", data: "version=1\n"}))
	if err := os.MkdirAll(m.DataDir, 0700); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(m.DataDir, "catalogs")); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Stage(context.Background()); err == nil {
		t.Fatal("install followed an outside parent")
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatalf("outside directory changed: %+v %v", entries, err)
	}
}
