package runtime

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwnden/platform/internal/challenge"
)

func TestToolboxMountPreservesRepositoryPath(t *testing.T) {
	source := filepath.Join(t.TempDir(), "path with spaces,comma")
	fields, err := csv.NewReader(strings.NewReader(toolboxMount(source, false))).Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 4 || fields[1] != "source="+source || fields[2] != "target=/challenge" || fields[3] != "readonly" {
		t.Fatalf("repository path changed mount options: %v", fields)
	}
}

func TestFileChallengeNeedsNoDocker(t *testing.T) {
	c := &challenge.Loaded{}
	ctx := context.Background()
	if cfg, err := Validate(ctx, c, "", false); err != nil || cfg != nil {
		t.Fatalf("validate file challenge: config=%v, err=%v", cfg, err)
	}
	if err := Run(ctx, c); err != nil {
		t.Fatalf("run file challenge: %v", err)
	}
	if err := Stop(ctx, c); err != nil {
		t.Fatalf("stop file challenge: %v", err)
	}
}

func TestCheckConfigRepositoryBoundary(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "inside")
	if err := os.Mkdir(inside, 0700); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	c := &challenge.Loaded{RepoRoot: root, Challenge: challenge.Challenge{Solve: challenge.Solve{Network: "default"}}}
	project := Project(c)
	config := func(source string) *Config {
		return &Config{
			Services: map[string]Service{"app": {Volumes: []Mount{{Type: "bind", Source: source, ReadOnly: true}}}},
			Networks: map[string]Network{"default": {Name: project + "_default"}},
		}
	}
	if err := checkConfig(c, config(inside), project); err != nil {
		t.Fatalf("inside path rejected: %v", err)
	}
	for _, source := range []string{outside, link} {
		if err := checkConfig(c, config(source), project); err == nil || !strings.Contains(err.Error(), "outside challenges repository") {
			t.Fatalf("outside path %q accepted: %v", source, err)
		}
	}
}

func TestCheckConfigRejectsHostAccess(t *testing.T) {
	c := &challenge.Loaded{RepoRoot: t.TempDir(), Challenge: challenge.Challenge{Solve: challenge.Solve{Network: "default"}}}
	project := Project(c)
	config := &Config{
		Services: map[string]Service{"app": {Privileged: true}},
		Networks: map[string]Network{"default": {Name: project + "_default"}},
	}
	if err := checkConfig(c, config, project); err == nil || !strings.Contains(err.Error(), "host privileges") {
		t.Fatalf("privileged service accepted: %v", err)
	}
	config.Services["app"] = Service{}
	config.Volumes = map[string]Volume{"host-data": {DriverOpts: map[string]any{"device": "/tmp"}}}
	if err := checkConfig(c, config, project); err == nil || !strings.Contains(err.Error(), "driver_opts") {
		t.Fatalf("local-driver host bind accepted: %v", err)
	}
	config.Volumes = nil
	config.Services["app"] = Service{VolumesFrom: []string{"external"}}
	if err := checkConfig(c, config, project); err == nil || !strings.Contains(err.Error(), "host privileges") {
		t.Fatalf("volumes_from accepted: %v", err)
	}
	config.Services["app"] = Service{}
	secret := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(secret, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	config.Secrets = map[string]Resource{"host-secret": {File: secret}}
	if err := checkConfig(c, config, project); err == nil || !strings.Contains(err.Error(), "outside challenges repository") {
		t.Fatalf("outside secret accepted: %v", err)
	}
	config.Secrets = nil
	config.Networks["default"] = Network{Name: "host", External: true}
	if err := checkConfig(c, config, project); err == nil || !strings.Contains(err.Error(), "project-scoped") {
		t.Fatalf("external network accepted: %v", err)
	}
}

func TestLocalBuildCacheRepositoryBoundary(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "challenges", "test")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	c := &challenge.Loaded{RepoRoot: root, Dir: dir}
	for _, attribute := range []string{"src", "dest"} {
		cases := []struct {
			name, entry string
			valid       bool
		}{
			{"inside existing", "type=local," + attribute + "=" + root, true},
			{"inside new", "type=local," + attribute + "=" + filepath.Join(root, ".cache", "new"), true},
			{"relative shared", "type=local," + attribute + "=../../.cache/new", true},
			{"quoted comma", fmt.Sprintf("type=local,\"%s=%s\"", attribute, filepath.Join(root, "spaces and,comma", "new")), true},
			{"outside existing", "type=local," + attribute + "=" + outside, false},
			{"outside new", "type=local," + attribute + "=" + filepath.Join(outside, "new"), false},
			{"relative escape", "type=local," + attribute + "=../../../outside", false},
			{"case-insensitive key", "TYPE=local," + strings.ToUpper(attribute) + "=" + outside, false},
			{"last path wins", "type=local," + attribute + "=" + root + "," + attribute + "=" + outside, false},
			{"last type wins", "type=registry,type=local," + attribute + "=" + outside, false},
			{"missing path", "type=local", false},
			{"broken CSV", "type=local,\"" + attribute + "=unterminated", false},
			{"registry shorthand", "example/image:cache", true},
			{"registry attributes", "type=registry,ref=example/image:cache", true},
		}
		for _, tc := range cases {
			t.Run(attribute+"/"+tc.name, func(t *testing.T) {
				err := checkCachePaths(c, []string{tc.entry}, attribute)
				if (err == nil) != tc.valid {
					t.Fatalf("cache boundary result: %v (valid=%t)", err, tc.valid)
				}
			})
		}
	}
}

func TestLocalBuildCacheSymlinkBoundary(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	for name, target := range map[string]string{"inside": root, "outside": outside} {
		if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
	}
	c := &challenge.Loaded{RepoRoot: root, Dir: root}
	for _, attribute := range []string{"src", "dest"} {
		for _, name := range []string{"inside", "outside"} {
			for _, suffix := range []string{"", "/new/cache"} {
				t.Run(attribute+"/"+name+suffix, func(t *testing.T) {
					err := checkCachePaths(c, []string{"type=local," + attribute + "=" + name + suffix}, attribute)
					if (err == nil) != (name == "inside") {
						t.Fatalf("cache symlink boundary result: %v", err)
					}
				})
			}
		}
	}
}

func TestLocalBuildCacheParentTraversal(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	for _, dir := range []string{filepath.Join(root, "cache"), filepath.Join(root, "deep"), filepath.Join(outside, "deep"), filepath.Join(outside, "cache")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(outside, "deep"), filepath.Join(root, "escape")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := os.Symlink(filepath.Join(root, "deep"), filepath.Join(root, "internal")); err != nil {
		t.Fatal(err)
	}
	c := &challenge.Loaded{RepoRoot: root, Dir: root}
	for _, attribute := range []string{"src", "dest"} {
		for _, path := range []string{"internal/../cache", "internal/../new/cache", "missing/../new/cache"} {
			if err := checkCachePaths(c, []string{"type=local," + attribute + "=" + path}, attribute); err != nil {
				t.Fatalf("internal parent traversal rejected: %v", err)
			}
		}
		for _, path := range []string{
			"escape/../cache", root + string(os.PathSeparator) + "escape/../cache",
			"escape/../new/cache", "missing/../../../outside-cache",
			"missing/../escape/../cache", "missing/../escape/../new/cache",
		} {
			t.Run(attribute+"/"+path, func(t *testing.T) {
				if err := checkCachePaths(c, []string{"type=local," + attribute + "=" + path}, attribute); err == nil {
					t.Fatal("parent traversal through real or future directories escaped the repository")
				}
			})
		}
	}
}
