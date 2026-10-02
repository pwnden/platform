package runtime

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"time"
)

var errOutputLimit = errors.New("command output exceeded 8 MiB per stream")

type boundedOutput struct {
	buffer   bytes.Buffer
	cancel   context.CancelFunc
	overflow bool
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	remaining := outputLimit - b.buffer.Len()
	if len(p) <= remaining {
		return b.buffer.Write(p)
	}
	n, _ := b.buffer.Write(p[:remaining])
	b.overflow = true
	b.cancel()
	return n, errOutputLimit
}

func commandInput(ctx context.Context, dir string, extraEnv []string, input io.Reader, name string, args ...string) (string, string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir, cmd.Env, cmd.Stdin = dir, append(os.Environ(), extraEnv...), input
	// A child that leaves inherited pipes open must not hold the caller forever.
	cmd.WaitDelay = time.Second
	stdout, stderr := boundedOutput{cancel: cancel}, boundedOutput{cancel: cancel}
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if stdout.overflow || stderr.overflow {
		err = errOutputLimit
	}
	return stdout.buffer.String(), stderr.buffer.String(), err
}
