package checkout

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// web keeps host SDKs optional. Only repository sources are mounted, read-only;
// installed dependencies and Vite's cache live in a disposable Docker volume.
func (l *Launcher) web(ctx context.Context, directory string) (target, nonce string, stop func() error, err error) {
	root := filepath.Join(l.Root, "web")
	if err := plainPath(root, false); err != nil {
		return "", "", nil, err
	}
	manifests := filepath.Join(directory, "frontend")
	if err := copyWebManifests(root, manifests); err != nil {
		return "", "", nil, err
	}
	// A checkout-specific image tag avoids replacing another workspace's tools.
	name := "pwnden-web-" + filepath.Base(directory)
	identity := sha256.Sum256([]byte(l.Root))
	image := "pwnden-web-dev:" + hex.EncodeToString(identity[:8])
	if err := l.execute(ctx, "docker", []string{"buildx", "build", "--load", "--file", filepath.Join(l.Root, "Dockerfile.dev"), "--tag", image, manifests}, l.Stdout, l.Stderr); err != nil {
		return "", "", nil, err
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", "", nil, err
	}
	nonce = hex.EncodeToString(secret)
	cleanup := func() error {
		cleanup, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		return l.execute(cleanup, "docker", []string{"rm", "--force", "--volumes", name}, io.Discard, l.Stderr)
	}
	// Docker mount arguments are CSV, including paths containing commas.
	var mount strings.Builder
	writer := csv.NewWriter(&mount)
	if err := writer.Write([]string{"type=bind", "source=" + root, "target=/web", "readonly"}); err != nil {
		return "", "", nil, err
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", "", nil, err
	}
	args := []string{"run", "--detach", "--init", "--name", name,
		"--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--pids-limit", "256",
		"--publish", "127.0.0.1::5173", "--mount", strings.TrimSpace(mount.String()),
		"--env", "PWNDEN_STYLE_NONCE=" + nonce}
	// Hide host node_modules at every workspace level; their native modules may
	// target another OS, and development caches must stay writable in Docker.
	if err := filepath.WalkDir(manifests, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Name() == "package.json" {
			relative, err := filepath.Rel(manifests, filepath.Dir(path))
			if err != nil {
				return err
			}
			// Nested volumes need an existing mount point under a read-only bind.
			// These ignored directories contain no host-installed dependencies.
			mountpoint := filepath.Join(root, relative, "node_modules")
			if err := os.Mkdir(mountpoint, 0755); err != nil && !errors.Is(err, os.ErrExist) {
				return err
			}
			args = append(args, "--volume", "/web/"+filepath.ToSlash(filepath.Join(relative, "node_modules")))
		}
		return nil
	}); err != nil {
		return "", "", nil, err
	}
	uid, gid := os.Getuid(), os.Getgid()
	if uid < 0 || gid < 0 {
		uid, gid = 1000, 1000
	}
	args = append(args, "--user", fmt.Sprintf("%d:%d", uid, gid), image)
	if err := l.execute(ctx, "docker", args, io.Discard, l.Stderr); err != nil {
		// A failed create/start may still have allocated an owned container.
		return "", "", nil, errors.Join(err, cleanup())
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, cleanup())
		}
	}()
	var published strings.Builder
	if err = l.execute(ctx, "docker", []string{"port", name, "5173/tcp"}, &published, l.Stderr); err != nil {
		return "", "", nil, err
	}
	authority := strings.TrimSpace(published.String())
	host, port, splitErr := net.SplitHostPort(authority)
	if splitErr != nil || host != "127.0.0.1" || port == "" {
		return "", "", nil, errors.New("development frontend must publish on IPv4 loopback")
	}
	target = "http://" + authority
	ready, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: time.Second}
	defer client.CloseIdleConnections()
	for {
		request, requestErr := http.NewRequestWithContext(ready, "GET", target+"/@vite/client", nil)
		if requestErr != nil {
			return "", "", nil, requestErr
		}
		response, requestErr := client.Do(request)
		if requestErr == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return target, nonce, cleanup, nil
			}
		}
		select {
		case <-ready.Done():
			logs, finish := context.WithTimeout(context.Background(), 5*time.Second)
			_ = l.execute(logs, "docker", []string{"logs", "--tail", "30", name}, l.Stderr, l.Stderr)
			finish()
			return "", "", nil, fmt.Errorf("development frontend did not become ready: %w", ready.Err())
		case <-time.After(150 * time.Millisecond):
		}
	}
}

func copyWebManifests(root, destination string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "node_modules" || entry.Name() == "dist" || strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.Name() != "package.json" && relative != "pnpm-lock.yaml" && relative != "pnpm-workspace.yaml" {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("frontend manifests must be files inside the checkout")
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		output := filepath.Join(destination, relative)
		if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
			return err
		}
		return os.WriteFile(output, content, 0644)
	})
}
