package application

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwnden/platform/internal/challenge"
	"github.com/pwnden/platform/internal/installation"
	"github.com/pwnden/platform/internal/runtime"
	"github.com/pwnden/platform/internal/testutil"
)

func setupFixture(t *testing.T, service bool) (*SetupService, *challenge.Loaded) {
	t.Helper()
	_, original := fixture(t, service)
	dir := t.TempDir()
	archive := filepath.Join(dir, installation.ArchiveName)
	file, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	compressed := gzip.NewWriter(file)
	writer := tar.NewWriter(compressed)
	err = filepath.Walk(original.RepoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(original.RepoRoot, path)
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = "catalog"
		if rel != "." {
			header.Name += "/" + filepath.ToSlash(rel)
		}
		if err := writer.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(writer, input)
		return errors.Join(copyErr, input.Close())
	})
	if err := errors.Join(err, writer.Close(), compressed.Close(), file.Close()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(data)
	manager := &installation.Manager{DataDir: filepath.Join(dir, "data"), Distribution: installation.Distribution{
		Archive: archive, Revision: strings.Repeat("1", 40), SHA256: hex.EncodeToString(hash[:]),
	}}
	root := filepath.Join(manager.DataDir, "catalogs", manager.Distribution.Revision+"-"+manager.Distribution.SHA256, "catalog")
	c := *original
	c.RepoRoot, c.Dir = root, filepath.Join(root, "challenges", c.Slug)
	return &SetupService{manager: manager}, &c
}

func setupReplies(c *challenge.Loaded) []testutil.Reply {
	return []testutil.Reply{
		{Match: []string{"info"}, Out: "linux\n"},
		{Match: []string{"version", "{{.Server.Version}}"}, Out: "29.4.1"},
		{Match: []string{"compose", "version"}, Out: "5.1.3\n"},
		{Match: []string{"config"}, Out: config(c)},
		{Match: []string{"pull", "--ignore-buildable"}},
		{Match: []string{"build"}},
		{Match: []string{"image", "inspect"}},
	}
}

func TestSetupPreparesAndActivates(t *testing.T) {
	for _, service := range []bool{false, true} {
		t.Run(fmt.Sprintf("service=%t", service), func(t *testing.T) {
			s, c := setupFixture(t, service)
			calls := testutil.Docker(t, setupReplies(c)...)
			for attempt := 0; attempt < 2; attempt++ {
				result, err := s.Setup(context.Background())
				if err != nil || result.ProblemCount != 1 || result.ContractVersion != 1 {
					t.Fatalf("setup: %+v %v", result, err)
				}
				root, err := s.manager.Resolve()
				if err != nil || root != c.RepoRoot {
					t.Fatalf("setup root: %q %v", root, err)
				}
			}
			for _, call := range calls() {
				for _, arg := range call {
					if arg == "up" || arg == "run" {
						t.Fatalf("setup started a problem: %v", call)
					}
				}
			}
		})
	}
}

func TestPrepareLiveCheckoutPreservesFilesAndDoesNotStartServices(t *testing.T) {
	for _, service := range []bool{false, true} {
		t.Run(fmt.Sprintf("service=%t", service), func(t *testing.T) {
			s, c := fixture(t, service)
			calls := testutil.Docker(t, setupReplies(c)...)
			result, err := s.Prepare(context.Background())
			if err != nil || result.ProblemCount != 1 {
				t.Fatalf("prepare: %+v %v", result, err)
			}
			for _, call := range calls() {
				for _, arg := range call {
					if arg == "up" || arg == "run" {
						t.Fatal("prepare started services", call)
					}
				}
			}
			if _, err := os.Stat(c.Dir); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSetupFailurePublishesNoInstallation(t *testing.T) {
	s, c := setupFixture(t, false)
	testutil.Docker(t,
		testutil.Reply{Match: []string{"info"}, Out: "linux"},
		testutil.Reply{Match: []string{"version", "{{.Server.Version}}"}, Out: "29.4.1"},
		testutil.Reply{Match: []string{"compose", "version"}},
		testutil.Reply{Match: []string{"image", "inspect"}, Code: 1},
		testutil.Reply{Match: []string{"pull"}, Code: 125, Err: "offline"},
	)
	result, err := s.Setup(context.Background())
	requireCode(t, err, SetupFailed)
	if result.ProblemCount != 0 {
		t.Fatal("failure returned success data")
	}
	if _, err := s.manager.Resolve(); !errors.Is(err, installation.ErrSetupRequired) {
		t.Fatal("failed preparation became active")
	}
	if _, err := os.Stat(c.RepoRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed preparation left a new catalog: %v", err)
	}
}

func TestSetupPreservesAnActivePreviousDistribution(t *testing.T) {
	s, c := setupFixture(t, true)
	testutil.Docker(t, setupReplies(c)...)
	if _, err := s.Setup(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := runtime.SaveState(c, runtime.State{Project: runtime.Project(c), Flag: "pwnden{active}"}); err != nil {
		t.Fatal(err)
	}
	s.manager.Distribution.Revision = strings.Repeat("2", 40)
	_, err := s.Setup(context.Background())
	requireCode(t, err, AlreadyRunning)
	root, err := s.manager.Resolve()
	if err != nil || root != c.RepoRoot {
		t.Fatalf("active distribution changed: %q %v", root, err)
	}
	state, err := runtime.ReadState(c)
	if err != nil || state.Flag != "pwnden{active}" {
		t.Fatalf("active state lost: %+v %v", state, err)
	}
}
