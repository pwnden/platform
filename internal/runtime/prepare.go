package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/pwnden/platform/internal/challenge"
)

func CheckEngine(ctx context.Context) error {
	out, stderr, err := command(ctx, "", nil, "docker", "info", "--format", "{{.OSType}}")
	if err != nil {
		return fmt.Errorf("Docker must be installed and running: %w: %s", err, stderr)
	}
	if strings.TrimSpace(out) != "linux" {
		return fmt.Errorf("Docker must use Linux containers; reported %q", strings.TrimSpace(out))
	}
	if _, stderr, err := command(ctx, "", nil, "docker", "compose", "version", "--short"); err != nil {
		return fmt.Errorf("Docker Compose plugin is required: %w: %s", err, stderr)
	}
	return nil
}

// Prepare validates policy and builds/pulls dependencies without starting services.
func Prepare(ctx context.Context, c *challenge.Loaded) error {
	if c.Compose != "" {
		for _, patched := range []bool{false, true} {
			if patched && c.Patched == nil {
				continue
			}
			if _, err := Validate(ctx, c, "pwnden{setup}", patched); err != nil {
				return err
			}
			project := Project(c)
			files := []string{c.Compose}
			if patched {
				project += "-patched"
				files = append(files, c.Patched.Compose)
			}
			for _, step := range [][]string{{"pull", "--ignore-buildable"}, {"build"}} {
				if _, stderr, err := command(ctx, c.Dir, []string{"FLAG=pwnden{setup}"}, "docker", composeArgs(c, project, files, step...)...); err != nil {
					return fmt.Errorf("prepare %s: %w: %s", c.Slug, err, stderr)
				}
			}
		}
	}
	if err := prepareToolImage(ctx, c, c.Solve.Image); err != nil {
		return err
	}
	if c.Patched != nil && c.Patched.Image != "" && c.Patched.Image != c.Solve.Image {
		return prepareToolImage(ctx, c, c.Patched.Image)
	}
	return nil
}

func prepareToolImage(ctx context.Context, c *challenge.Loaded, image string) error {
	if _, _, err := command(ctx, c.Dir, nil, "docker", "image", "inspect", "--format", "{{.Id}}", "--", image); err != nil {
		if _, stderr, err := command(ctx, c.Dir, nil, "docker", "pull", "--", image); err != nil {
			return fmt.Errorf("prepare toolbox image: %w: %s", err, stderr)
		}
	}
	return nil
}
