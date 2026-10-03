package application

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDifficultySurvivesCatalogAndDetailWithLegacyUnrated(t *testing.T) {
	for level := 0; level <= 5; level++ {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			s, c := fixture(t, false)
			if err := os.WriteFile(filepath.Join(c.Dir, "README.md"), []byte("A player brief."), 0600); err != nil {
				t.Fatal(err)
			}
			if level > 0 {
				path := filepath.Join(c.RepoRoot, "contract.toml")
				definition, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(strings.Replace(string(definition), "version=1", "version=5", 1)), 0600); err != nil {
					t.Fatal(err)
				}
				path = filepath.Join(c.Dir, "challenge.toml")
				manifest, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				updated := strings.Replace(string(manifest), "schema=1", fmt.Sprintf("schema=5\ndifficulty=%d", level), 1) + "\n[content]\ndescription='README.md'\n[player]\ntools=['files','terminal']\n"
				if err := os.WriteFile(path, []byte(updated), 0600); err != nil {
					t.Fatal(err)
				}
			}
			items, err := s.List(context.Background())
			if err != nil || len(items) != 1 || items[0].Difficulty != level {
				t.Fatalf("catalog difficulty: %+v %v", items, err)
			}
			detail, err := s.Detail(context.Background(), c.Slug)
			if err != nil || detail.Difficulty != level {
				t.Fatalf("detail difficulty: %+v %v", detail, err)
			}
		})
	}
}

func TestDetailAndDeclaredDownload(t *testing.T) {
	s, c := fixture(t, false)
	ctx := context.Background()
	os.WriteFile(filepath.Join(c.Dir, "README.md"), []byte("# Example\n<script>private()</script>\n"), 0600)
	os.WriteFile(filepath.Join(c.Dir, "solve.txt"), []byte("private solution"), 0600)
	result, err := s.Detail(ctx, c.Slug)
	if err != nil || result.Title != c.Slug || result.Description != "# Example\n<script>private()</script>\n" || len(result.Files) != 1 {
		t.Fatalf("detail: %+v %v", result, err)
	}
	file := result.Files[0]
	if file.Name != "file.txt" || file.Size != 12 || len(file.ID) != 64 {
		t.Fatal(file)
	}
	download, err := s.Download(ctx, c.Slug, file.ID)
	if err != nil {
		t.Fatal(err)
	}
	content, err := io.ReadAll(download.Content)
	download.Content.Close()
	if err != nil || string(content) != "distribution" || download.Name != "file.txt" || download.Size != 12 {
		t.Fatalf("download: %+v %q %v", download, content, err)
	}
	_, err = s.Download(ctx, c.Slug, strings.Repeat("0", 64))
	requireCode(t, err, NotFound)
	_, err = s.Download(ctx, c.Slug, "../../solve.txt")
	requireCode(t, err, InvalidArgument)
	_, err = s.Download(ctx, "../example", file.ID)
	requireCode(t, err, InvalidArgument)
	os.Remove(filepath.Join(c.Dir, "README.md"))
	result, err = s.Detail(ctx, c.Slug)
	if err != nil || result.Description != "" {
		t.Fatalf("optional description: %+v %v", result, err)
	}
	items, err := s.List(ctx)
	if err != nil || len(items) != 1 || items[0].SearchText != "" {
		t.Fatalf("legacy catalog without brief: %+v %v", items, err)
	}
}

func TestCatalogSearchIndexesPublicToolNames(t *testing.T) {
	s, c := fixture(t, false)
	brief := "# 개발용 점검 포트\nUse `nmap --unprivileged -sT` to inspect the service.\n"
	if err := os.WriteFile(filepath.Join(c.Dir, "README.md"), []byte(brief), 0600); err != nil {
		t.Fatal(err)
	}
	items, err := s.List(context.Background())
	if err != nil || len(items) != 1 || items[0].SearchText != brief {
		t.Fatalf("tool search text missing: %+v %v", items, err)
	}
}

func TestDistributionDirectoriesAndRepositoryBoundaries(t *testing.T) {
	s, c := fixture(t, false)
	dir := filepath.Join(c.Dir, "bundle")
	os.MkdirAll(filepath.Join(dir, "nested"), 0700)
	os.WriteFile(filepath.Join(dir, "nested", "one.bin"), []byte{0, 1, 255}, 0600)
	manifest := filepath.Join(c.Dir, "challenge.toml")
	content, _ := os.ReadFile(manifest)
	os.WriteFile(manifest, []byte(strings.Replace(string(content), "files=['file.txt']", "files=['bundle','../shared.txt']", 1)), 0600)
	os.WriteFile(filepath.Join(c.Dir, "..", "shared.txt"), []byte("shared"), 0600)
	ctx := context.Background()
	detail, err := s.Detail(ctx, c.Slug)
	if err != nil || len(detail.Files) != 2 || detail.Files[1].Name != "bundle/nested/one.bin" {
		t.Fatalf("directory/shared files: %+v %v", detail, err)
	}
	for _, file := range detail.Files {
		download, err := s.Download(ctx, c.Slug, file.ID)
		if err != nil {
			t.Fatal(err)
		}
		download.Content.Close()
	}
	// Author-declared directories may contain repository-contained links.
	if err := os.Symlink(filepath.Join(c.Dir, "..", "shared.txt"), filepath.Join(dir, "linked.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	linked, err := s.Detail(ctx, c.Slug)
	if err != nil || len(linked.Files) != 3 {
		t.Fatalf("contained absolute symlink: %+v %v", linked, err)
	}
	outside := filepath.Join(t.TempDir(), "secret")
	os.WriteFile(outside, []byte("private"), 0600)
	if err := os.Symlink(outside, filepath.Join(dir, "escape")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	_, err = s.Detail(ctx, c.Slug)
	requireCode(t, err, InvalidArgument)
	_, err = s.Download(ctx, c.Slug, detail.Files[1].ID)
	requireCode(t, err, InvalidArgument)
	os.Remove(filepath.Join(dir, "escape"))
	if err := os.Symlink(".", filepath.Join(dir, "cycle")); err != nil {
		t.Fatal(err)
	}
	_, err = s.Detail(ctx, c.Slug)
	requireCode(t, err, InvalidArgument)
}

func TestDescriptionAndCatalogCancellation(t *testing.T) {
	s, c := fixture(t, false)
	path := filepath.Join(c.Dir, "README.md")
	for _, content := range [][]byte{{255}, []byte(strings.Repeat("x", (1<<20)+1))} {
		os.WriteFile(path, content, 0600)
		_, err := s.Detail(context.Background(), c.Slug)
		requireCode(t, err, InvalidArgument)
		_, err = s.List(context.Background())
		requireCode(t, err, InvalidArgument)
	}
	os.Remove(path)
	if err := os.Symlink(filepath.Join(t.TempDir(), "missing"), path); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	_, err := s.Detail(context.Background(), c.Slug)
	if err == nil {
		t.Fatal("outside description was accepted")
	}
	if _, err := s.List(context.Background()); err == nil {
		t.Fatal("catalog search accepted an outside description")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = s.Detail(ctx, c.Slug)
	requireCode(t, err, Canceled)
	_, err = s.Download(ctx, c.Slug, strings.Repeat("a", 64))
	requireCode(t, err, Canceled)
	_, err = s.List(ctx)
	requireCode(t, err, Canceled)
}

func TestDeclaredPlayerContentAndSpoilerReads(t *testing.T) {
	s, c := fixture(t, false)
	definition := filepath.Join(c.RepoRoot, "contract.toml")
	data, _ := os.ReadFile(definition)
	os.WriteFile(definition, []byte(strings.Replace(string(data), "version=1", "version=2", 1)), 0600)
	manifest := filepath.Join(c.Dir, "challenge.toml")
	data, _ = os.ReadFile(manifest)
	metadata := strings.Replace(string(data), "schema=1", "schema=2", 1) + "\n[content]\ndescription='brief.md'\nhints=['hint.md']\nwalkthrough='answer.md'\n"
	os.WriteFile(manifest, []byte(metadata), 0600)
	for name, body := range map[string]string{"brief.md": "Find the key.", "hint.md": "Look at the address.", "answer.md": "The complete answer.", "private.md": "Private author notes."} {
		os.WriteFile(filepath.Join(c.Dir, name), []byte(body), 0600)
	}
	ctx := context.Background()
	detail, err := s.Detail(ctx, c.Slug)
	if err != nil || detail.Description != "Find the key." || detail.HintCount != 1 || !detail.Walkthrough {
		t.Fatalf("player detail: %+v %v", detail, err)
	}
	items, err := s.List(ctx)
	if err != nil || len(items) != 1 || items[0].SearchText != "Find the key." {
		t.Fatalf("search indexed something other than the declared public brief: %+v %v", items, err)
	}
	for id, want := range map[string]string{"hint-1": "Look at the address.", "walkthrough": "The complete answer."} {
		got, err := s.Guidance(ctx, c.Slug, id)
		if err != nil || got != want {
			t.Fatalf("%s: %q %v", id, got, err)
		}
	}
	for _, id := range []string{"hint-0", "hint-01", "hint-2", "private.md", "../answer.md"} {
		_, err := s.Guidance(ctx, c.Slug, id)
		requireCode(t, err, NotFound)
	}
	for _, body := range [][]byte{{255}, []byte(strings.Repeat("x", (1<<20)+1))} {
		os.WriteFile(filepath.Join(c.Dir, "answer.md"), body, 0600)
		_, err := s.Guidance(ctx, c.Slug, "walkthrough")
		requireCode(t, err, InvalidArgument)
	}
	// An authored path and a later file replacement both retain root containment.
	outside := filepath.Join(t.TempDir(), "secret.md")
	os.WriteFile(outside, []byte("outside"), 0600)
	os.Remove(filepath.Join(c.Dir, "answer.md"))
	if err := os.Symlink(outside, filepath.Join(c.Dir, "answer.md")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := s.Guidance(ctx, c.Slug, "walkthrough"); err == nil {
		t.Fatal("outside walkthrough accepted")
	}
	os.WriteFile(manifest, []byte(strings.Replace(metadata, "'answer.md'", "'../../../outside.md'", 1)), 0600)
	_, err = s.Guidance(ctx, c.Slug, "walkthrough")
	requireCode(t, err, InvalidArgument)
	// Reading a brief does not read broken spoiler documents.
	if _, err := s.Detail(ctx, c.Slug); err != nil {
		t.Fatal("brief eagerly read a walkthrough", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	_, err = s.Guidance(canceled, c.Slug, "hint-1")
	requireCode(t, err, Canceled)
}
