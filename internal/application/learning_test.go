package application

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLearningCatalogClosureAndScopedConceptReads(t *testing.T) {
	s, c := fixture(t, false)
	path := filepath.Join(c.RepoRoot, "contract.toml")
	data, _ := os.ReadFile(path)
	os.WriteFile(path, []byte(strings.Replace(string(data), "version=1", "version=7", 1)), 0600)
	path = filepath.Join(c.Dir, "challenge.toml")
	data, _ = os.ReadFile(path)
	metadata := strings.Replace(string(data), "schema=1", "schema=7\ndifficulty=1", 1) + "\n[player]\ntools=['files','terminal']\ncli=[]\n[learning]\nrequires=['ownership']\nteaches=['export']\n[content]\ndescription='README.md'\n"
	os.WriteFile(path, []byte(metadata), 0600)
	os.WriteFile(filepath.Join(c.Dir, "README.md"), []byte("Player briefing."), 0600)
	directory := filepath.Join(c.RepoRoot, "knowledge")
	os.Mkdir(directory, 0700)
	registry := ""
	for _, id := range []string{"base", "ownership", "export", "private"} {
		requires := "[]"
		if id == "ownership" {
			requires = "['base']"
		}
		registry += "[[concepts]]\nid='" + id + "'\ntitle='" + id + "'\nrequires=" + requires + "\nrelated=[]\n"
		os.WriteFile(filepath.Join(directory, id+".md"), []byte("# "+id+"\n\nExplanation."), 0600)
	}
	os.WriteFile(filepath.Join(directory, "catalog.toml"), []byte(registry), 0600)
	ctx := context.Background()
	items, err := s.List(ctx)
	if err != nil || len(items) != 1 || items[0].Learning == nil || len(items[0].Learning.Requires) != 2 || items[0].Learning.Requires[0].ID != "ownership" || items[0].Learning.Requires[1].ID != "base" {
		t.Fatalf("learning closure: %+v %v", items, err)
	}
	detail, err := s.Detail(ctx, c.Slug)
	if err != nil || detail.Learning.Teaches[0].ID != "export" {
		t.Fatalf("detail learning: %+v %v", detail, err)
	}
	for _, id := range []string{"base", "ownership", "export"} {
		body, err := s.Guidance(ctx, c.Slug, "concept-"+id)
		if err != nil || body != "# "+id+"\n\nExplanation." {
			t.Fatalf("concept %s: %q %v", id, body, err)
		}
	}
	for _, id := range []string{"concept-private", "concept-missing", "concept-../../private"} {
		_, err := s.Guidance(ctx, c.Slug, id)
		requireCode(t, err, NotFound)
	}
	out := filepath.Join(t.TempDir(), "private.md")
	os.WriteFile(out, []byte("private"), 0600)
	os.Remove(filepath.Join(directory, "base.md"))
	if err := os.Symlink(out, filepath.Join(directory, "base.md")); err != nil {
		t.Skip(err)
	}
	if _, err := s.Guidance(ctx, c.Slug, "concept-base"); err == nil {
		t.Fatal("outside concept document accepted")
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	_, err = s.Guidance(ctx, c.Slug, "concept-base")
	requireCode(t, err, Canceled)
}
