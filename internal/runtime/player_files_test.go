package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPlayerFilesExposeOnlyDeclarations(t *testing.T) {
	c := fixture(t)
	for name, content := range map[string]string{
		"files/input.txt": "exercise", "files/unlisted.txt": "hidden",
		"AUTHORING.md": "author", "README.md": "briefing",
		"solve/solve.py": "answer", "hints/1.md": "hint", "challenge.toml": "metadata",
	} {
		path := filepath.Join(c.Dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	c.Files = []string{"files/input.txt"}
	dir, err := playerFiles(c)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	data, err := os.ReadFile(filepath.Join(dir, "files/input.txt"))
	if err != nil || string(data) != "exercise" {
		t.Fatalf("distributed file: %q %v", data, err)
	}
	for _, name := range []string{"files/unlisted.txt", "AUTHORING.md", "README.md", "solve", "hints", "challenge.toml"} {
		if _, err := os.Lstat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("private path %s exposed: %v", name, err)
		}
	}
	c.Files = nil
	empty, err := playerFiles(c)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(empty)
	entries, err := os.ReadDir(empty)
	if err != nil || len(entries) != 0 {
		t.Fatalf("web-only workspace not empty: %v %v", entries, err)
	}
}

func TestPlayerFilesRejectDirectoriesAndEscapes(t *testing.T) {
	c := fixture(t)
	if err := os.Mkdir(filepath.Join(c.Dir, "files"), 0755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(out, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(out, filepath.Join(c.Dir, "escape")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{".", "files", "escape", "../secret", out} {
		c.Files = []string{name}
		if dir, err := playerFiles(c); err == nil {
			os.RemoveAll(dir)
			t.Fatalf("unsafe declaration accepted: %s", name)
		}
	}
}
