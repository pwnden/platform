package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/pwnden/platform/internal/challenge"
)

// Details and Files are read capabilities; downloaded streams belong to callers.
type Details interface {
	Detail(context.Context, string) (ProblemDetail, error)
}

type Files interface {
	Download(context.Context, string, string) (Download, error)
}

type ProblemFile struct {
	ID   string
	Name string
	Size int64
}

type ProblemDetail struct {
	Problem
	Description string
	Files       []ProblemFile
}

type Download struct {
	Name    string
	Size    int64
	Content io.ReadSeekCloser
}

type distributionFile struct {
	ProblemFile
	path string
}

func problem(c *challenge.Loaded) Problem {
	title, _ := c.Title.(string)
	category, _ := c.Category.(string)
	if title == "" {
		title = c.Slug
	}
	return Problem{Slug: c.Slug, Title: title, Category: category, Kind: kind(c)}
}

// Root keeps even a file replaced by a symlink confined while opening it.
func distribution(ctx context.Context, c *challenge.Loaded, root *os.Root) ([]distributionFile, error) {
	files := make(map[string]distributionFile)
	var walk func(string, string, []os.FileInfo) error
	walk = func(path, name string, ancestors []os.FileInfo) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		real, err := challenge.Within(c.RepoRoot, filepath.Join(c.RepoRoot, path))
		if err != nil {
			return err
		}
		resolved, err := filepath.Rel(c.RepoRoot, real)
		if err != nil {
			return err
		}
		info, err := root.Stat(resolved)
		if err != nil {
			return err
		}
		if info.IsDir() {
			for _, ancestor := range ancestors {
				if os.SameFile(info, ancestor) {
					return errors.New("distribution directory contains a cycle")
				}
			}
			dir, err := root.Open(resolved)
			if err != nil {
				return err
			}
			names, err := dir.Readdirnames(-1)
			dir.Close()
			if err != nil {
				return err
			}
			for _, child := range names {
				if err := walk(filepath.Join(path, child), filepath.ToSlash(filepath.Join(name, child)), append(ancestors, info)); err != nil {
					return err
				}
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return errors.New("distribution entries must be regular files or directories")
		}
		digest := sha256.Sum256([]byte(filepath.ToSlash(path)))
		id := hex.EncodeToString(digest[:])
		files[id] = distributionFile{ProblemFile: ProblemFile{ID: id, Name: name, Size: info.Size()}, path: resolved}
		return nil
	}
	for _, declared := range c.Files {
		// IDs follow declared logical paths. Resolve repository-contained links
		// and open through Root so a replacement cannot escape during access.
		path, err := filepath.Rel(c.RepoRoot, filepath.Join(c.Dir, filepath.FromSlash(declared)))
		if err != nil {
			return nil, err
		}
		if err := walk(path, filepath.ToSlash(filepath.Clean(declared)), nil); err != nil {
			return nil, err
		}
	}
	result := make([]distributionFile, 0, len(files))
	for _, file := range files {
		result = append(result, file)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func (s *Service) Detail(ctx context.Context, slug string) (ProblemDetail, error) {
	c, err := s.load(ctx, "detail", slug, false)
	if err != nil {
		return ProblemDetail{}, err
	}
	root, err := os.OpenRoot(c.RepoRoot)
	if err != nil {
		return ProblemDetail{}, loadError(ctx, "detail", slug, err)
	}
	defer root.Close()
	files, err := distribution(ctx, c, root)
	if err != nil {
		return ProblemDetail{}, loadError(ctx, "detail", slug, err)
	}
	result := ProblemDetail{Problem: problem(c), Files: make([]ProblemFile, 0, len(files))}
	for _, file := range files {
		result.Files = append(result.Files, file.ProblemFile)
	}
	path, _ := filepath.Rel(c.RepoRoot, filepath.Join(c.Dir, "README.md"))
	info, err := root.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return ProblemDetail{}, loadError(ctx, "detail", slug, err)
	}
	if !info.Mode().IsRegular() {
		return ProblemDetail{}, loadError(ctx, "detail", slug, errors.New("description must be a regular file"))
	}
	f, err := root.Open(path)
	if err != nil {
		return ProblemDetail{}, loadError(ctx, "detail", slug, err)
	}
	defer f.Close()
	content, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err == nil && (len(content) > 1<<20 || !utf8.Valid(content)) {
		err = errors.New("description must be UTF-8 and at most 1 MiB")
	}
	if err != nil {
		return ProblemDetail{}, loadError(ctx, "detail", slug, err)
	}
	result.Description = string(content)
	return result, nil
}

func (s *Service) Download(ctx context.Context, slug, id string) (Download, error) {
	if len(id) != 64 || strings.Trim(id, "0123456789abcdef") != "" {
		return Download{}, operationError(ctx, "download", slug, InvalidArgument, errors.New("invalid file ID"))
	}
	c, err := s.load(ctx, "download", slug, false)
	if err != nil {
		return Download{}, err
	}
	root, err := os.OpenRoot(c.RepoRoot)
	if err != nil {
		return Download{}, loadError(ctx, "download", slug, err)
	}
	defer root.Close()
	files, err := distribution(ctx, c, root)
	if err != nil {
		return Download{}, loadError(ctx, "download", slug, err)
	}
	for _, entry := range files {
		if entry.ID != id {
			continue
		}
		f, err := root.Open(entry.path)
		if err != nil {
			return Download{}, loadError(ctx, "download", slug, err)
		}
		info, err := f.Stat()
		if err == nil && !info.Mode().IsRegular() {
			err = errors.New("download must be a regular file")
		}
		if err == nil {
			err = ctx.Err()
		}
		if err != nil {
			f.Close()
			return Download{}, loadError(ctx, "download", slug, err)
		}
		return Download{Name: filepath.Base(entry.Name), Size: info.Size(), Content: f}, nil
	}
	return Download{}, operationError(ctx, "download", slug, NotFound, os.ErrNotExist)
}

var (
	_ Details = (*Service)(nil)
	_ Files   = (*Service)(nil)
)
