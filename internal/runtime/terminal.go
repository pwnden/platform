package runtime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/containerd/errdefs"
	"github.com/docker/cli/cli/connhelper"
	dockerContainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/client"
	"github.com/pwnden/platform/internal/challenge"
)

// Terminal owns one toolbox container and its daemon TTY, never a host shell.
type Terminal struct {
	engine   *client.Client
	stream   client.ContainerAttachResult
	id       string
	ctx      context.Context
	cancel   context.CancelFunc
	wait     client.ContainerWaitResult
	once     sync.Once
	closeErr error
}

func TerminalSize(cols, rows int) bool { return cols >= 2 && cols <= 500 && rows >= 1 && rows <= 200 }

func terminalEngine() (*client.Client, error) {
	// The official CLI helper keeps context, DOCKER_HOST, TLS, SSH and named-pipe
	// selection identical to Compose. Only the daemon allocates the terminal.
	helper, err := connhelper.GetCommandConnectionHelper("docker", "system", "dial-stdio")
	if err != nil {
		return nil, err
	}
	return client.New(client.WithHost(helper.Host), client.WithDialContext(helper.Dialer), client.WithAPIVersionFromEnv())
}

func OpenTerminal(ctx context.Context, c *challenge.Loaded, project string, cols, rows int) (_ *Terminal, err error) {
	if !TerminalSize(cols, rows) {
		return nil, errors.New("invalid terminal dimensions")
	}
	source, err := challenge.Within(c.RepoRoot, c.Dir)
	if err != nil {
		return nil, err
	}
	if err := prepareToolImage(ctx, c, c.Solve.Image); err != nil {
		return nil, err
	}
	network := "none"
	if c.Compose != "" {
		if project != Project(c) {
			return nil, errors.New("invalid terminal project")
		}
		network, err = networkID(ctx, c, project)
		if err != nil {
			return nil, err
		}
	}
	engine, err := terminalEngine()
	if err != nil {
		return nil, err
	}
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		engine.Close()
		return nil, err
	}
	name := "pwnden-terminal-" + hex.EncodeToString(id)
	owned := false
	defer func() {
		if err == nil {
			return
		}
		if owned {
			cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_, failure := engine.ContainerRemove(cleanup, name, client.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
			if failure != nil && !errdefs.IsNotFound(failure) {
				err = errors.Join(err, ErrCleanupFailed, failure)
			}
		}
		engine.Close()
	}()
	config := &dockerContainer.Config{
		Image: c.Solve.Image, Entrypoint: []string{"/bin/sh"}, Cmd: []string{"-i"},
		Tty: true, OpenStdin: true, StdinOnce: true, AttachStdin: true, AttachStdout: true, AttachStderr: true,
		WorkingDir: "/challenge", Env: []string{"TERM=xterm-256color"},
		Labels: map[string]string{"pwnden.kind": "terminal", "pwnden.problem": c.Slug, "pwnden.project": Project(c)},
	}
	if c.Solve.Writable && os.Geteuid() >= 0 && os.Getegid() >= 0 {
		config.User = fmt.Sprintf("%d:%d", os.Geteuid(), os.Getegid())
	}
	// Claim the random name before the request: an interrupted response may have
	// created the container even when its ID was never delivered.
	owned = true
	created, err := engine.ContainerCreate(ctx, client.ContainerCreateOptions{Name: name, Config: config, HostConfig: &dockerContainer.HostConfig{
		NetworkMode: dockerContainer.NetworkMode(network), CapDrop: []string{"ALL"}, SecurityOpt: []string{"no-new-privileges"},
		Mounts:      []mount.Mount{{Type: mount.TypeBind, Source: source, Target: "/challenge", ReadOnly: !c.Solve.Writable}},
		ConsoleSize: [2]uint{uint(rows), uint(cols)},
	}})
	if err != nil {
		return nil, err
	}
	stream, err := engine.ContainerAttach(ctx, created.ID, client.ContainerAttachOptions{Stream: true, Stdin: true, Stdout: true, Stderr: true})
	if err != nil {
		return nil, err
	}
	if _, err = engine.ContainerStart(ctx, created.ID, client.ContainerStartOptions{}); err != nil {
		stream.Close()
		return nil, err
	}
	sessionCtx, cancel := context.WithCancel(ctx)
	t := &Terminal{engine: engine, stream: stream, id: created.ID, ctx: sessionCtx, cancel: cancel}
	t.wait = engine.ContainerWait(sessionCtx, created.ID, client.ContainerWaitOptions{})
	context.AfterFunc(sessionCtx, func() { t.Close() })
	return t, nil
}

func (t *Terminal) Read(p []byte) (int, error)  { return t.stream.Reader.Read(p) }
func (t *Terminal) Write(p []byte) (int, error) { return t.stream.Conn.Write(p) }
func (t *Terminal) Resize(ctx context.Context, cols, rows int) error {
	if !TerminalSize(cols, rows) {
		return errors.New("invalid terminal dimensions")
	}
	_, err := t.engine.ContainerResize(ctx, t.id, client.ContainerResizeOptions{Width: uint(cols), Height: uint(rows)})
	return err
}
func (t *Terminal) Wait(ctx context.Context) (int, error) {
	select {
	case result := <-t.wait.Result:
		if result.Error != nil {
			return 0, errors.New(result.Error.Message)
		}
		return int(result.StatusCode), nil
	case err := <-t.wait.Error:
		return 0, err
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}
func (t *Terminal) Close() error {
	t.once.Do(func() {
		t.cancel()
		t.stream.Close()
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, err := t.engine.ContainerRemove(cleanup, t.id, client.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
		if err != nil && !errdefs.IsNotFound(err) {
			t.closeErr = errors.Join(ErrCleanupFailed, err)
		}
		t.engine.Close()
	})
	return t.closeErr
}

var _ io.ReadWriteCloser = (*Terminal)(nil)
