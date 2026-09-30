package installation

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pwnden/platform/internal/challenge"
)

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(data []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(data)
}

func catalogPath(name string) (string, error) {
	name = strings.TrimSuffix(name, "/")
	clean := path.Clean(name)
	if name != clean || strings.ContainsAny(name, "\\:") || (name != "catalog" && !strings.HasPrefix(name, "catalog/")) {
		return "", fmt.Errorf("unsafe problem archive path %q", name)
	}
	return name, nil
}

func extract(ctx context.Context, source io.Reader, destination string) error {
	compressed, err := gzip.NewReader(&contextReader{ctx: ctx, reader: source})
	if err != nil {
		return err
	}
	defer compressed.Close()
	reader := tar.NewReader(compressed)
	seen := make(map[string]bool)
	type link struct{ name, target string }
	var links []link
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header.Typeflag == tar.TypeXGlobalHeader {
			continue // git archive's revision metadata has no filesystem entry.
		}
		name, err := catalogPath(header.Name)
		if err != nil {
			return err
		}
		if seen[name] {
			return fmt.Errorf("duplicate problem archive entry %q", name)
		}
		seen[name] = true
		local := filepath.Join(destination, filepath.FromSlash(name))
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(local, 0755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if name == "catalog" {
				return errors.New("problem archive root must be a directory")
			}
			if err := os.MkdirAll(filepath.Dir(local), 0755); err != nil {
				return err
			}
			file, err := os.OpenFile(local, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(header.Mode)&0777)
			if err != nil {
				return err
			}
			if err := file.Chmod(os.FileMode(header.Mode) & 0777); err != nil {
				return errors.Join(err, file.Close())
			}
			_, copyErr := io.Copy(file, &contextReader{ctx: ctx, reader: reader})
			if err := errors.Join(copyErr, file.Close()); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if name == "catalog" || header.Linkname == "" || path.IsAbs(header.Linkname) || strings.ContainsAny(header.Linkname, "\\:") {
				return fmt.Errorf("unsafe problem archive symlink %q", name)
			}
			if _, err := catalogPath(path.Join(path.Dir(name), header.Linkname)); err != nil {
				return err
			}
			links = append(links, link{name, header.Linkname})
		default:
			return fmt.Errorf("unsupported problem archive entry %q", name)
		}
	}
	// Files are extracted before links, so archive writes cannot follow a link.
	for _, link := range links {
		local := filepath.Join(destination, filepath.FromSlash(link.name))
		parent := filepath.Dir(local)
		for check := parent; check != destination; check = filepath.Dir(check) {
			info, err := os.Lstat(check)
			if err == nil && info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("archive symlink parent %q", link.name)
			}
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		if err := os.MkdirAll(filepath.Dir(local), 0755); err != nil {
			return err
		}
		if err := os.Symlink(filepath.FromSlash(link.target), local); err != nil {
			return err
		}
	}
	root := filepath.Join(destination, "catalog")
	// The private installation parent stays 0700. Mounted problem directories
	// must be readable by the toolbox's container UID even with cap-drop ALL.
	if err := filepath.WalkDir(root, func(local string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.Chmod(local, 0755)
		}
		return nil
	}); err != nil {
		return err
	}
	for _, link := range links {
		if _, err := challenge.Within(root, filepath.Join(destination, filepath.FromSlash(link.name))); err != nil {
			return fmt.Errorf("invalid bundled symlink: %w", err)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err = os.Stat(filepath.Join(root, "contract.toml"))
	return err
}
