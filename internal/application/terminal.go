package application

import (
	"context"
	"errors"
	"io"

	"github.com/pwnden/platform/internal/runtime"
)

type TerminalSession interface {
	io.ReadWriteCloser
	Resize(context.Context, int, int) error
	Wait(context.Context) (int, error)
}

type Terminals interface {
	OpenTerminal(context.Context, string, int, int) (TerminalSession, error)
}

func (s *Service) OpenTerminal(ctx context.Context, slug string, cols, rows int) (TerminalSession, error) {
	if !runtime.TerminalSize(cols, rows) {
		return nil, operationError(ctx, "terminal", slug, InvalidArgument, errors.New("invalid terminal dimensions"))
	}
	c, err := s.load(ctx, "terminal", slug, false)
	if err != nil {
		return nil, err
	}
	project := ""
	if c.Compose != "" {
		state, err := runtime.ReadState(c)
		if err != nil {
			return nil, operationError(ctx, "terminal", slug, ExecutionFailed, err)
		}
		status, err := runtime.RunStatus(ctx, c)
		if err != nil {
			return nil, operationError(ctx, "terminal", slug, ValidationFailed, err)
		}
		if status != "running" {
			return nil, operationError(ctx, "terminal", slug, NotRunning, runtime.ErrNotRunning)
		}
		project = state.Project
	}
	session, err := runtime.OpenTerminal(ctx, c, project, cols, rows)
	if err != nil {
		return nil, operationError(ctx, "terminal", slug, ExecutionFailed, err)
	}
	return session, nil
}

var _ Terminals = (*Service)(nil)
