package runtime

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pwnden/platform/internal/challenge"
)

// Materialize only explicitly distributed files. Never mount the author tree
// in a player container, including through directory entries or symlinks.
func playerFiles(c *challenge.Loaded) (string, error) {
	dir, err := os.MkdirTemp("", "pwnden-player-files-*")
	if err != nil {
		return "", err
	}
	var size int64
	entries := 0
	for _, name := range c.Files {
		clean := filepath.Clean(name)
		if !filepath.IsLocal(name) || clean == "." {
			err = fmt.Errorf("invalid player file path %q", name)
			break
		}
		var source string
		source, err = challenge.Resolve(c.RepoRoot, c.Dir, name)
		if err != nil {
			break
		}
		var info os.FileInfo
		info, err = os.Stat(source)
		if err != nil {
			break
		}
		if !info.Mode().IsRegular() {
			err = fmt.Errorf("player distribution must be a regular file: %s", name)
			break
		}
		size += info.Size()
		entries += len(strings.Split(clean, string(filepath.Separator)))
		if size > workspaceSize || entries > workspaceInodes {
			err = errors.New("player files exceed the temporary workspace quota")
			break
		}
		target := filepath.Join(dir, clean)
		if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			break
		}
		var input, output *os.File
		input, err = os.Open(source)
		if err != nil {
			break
		}
		output, err = os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644|info.Mode().Perm()&0111)
		if err == nil {
			err = output.Chmod(0644 | info.Mode().Perm()&0111)
			if err == nil {
				_, err = io.CopyN(output, input, info.Size())
			}
			err = errors.Join(err, output.Close())
		}
		err = errors.Join(err, input.Close())
		if err != nil {
			break
		}
	}
	if err == nil {
		// Docker's fixed nonroot user must be able to traverse the copy even
		// when the host process uses a restrictive umask.
		err = filepath.WalkDir(dir, func(path string, entry os.DirEntry, failure error) error {
			if failure != nil {
				return failure
			}
			if entry.IsDir() {
				return os.Chmod(path, 0755)
			}
			return nil
		})
	}
	if err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}
