package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/pwnden/platform/internal/runtime"
)

type Catalog interface {
	List(context.Context) ([]Problem, error)
}

type Player interface {
	Execute(context.Context, string, []string) (Execution, error)
	Submit(context.Context, string, string) (Submission, error)
}

type Problem struct {
	Slug     string
	Title    string
	Category string
	Kind     Kind
}

type Execution struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type Submission struct {
	Slug     string
	Accepted bool
}

func (s *Service) List(ctx context.Context) ([]Problem, error) {
	if err := ctx.Err(); err != nil {
		return nil, operationError(ctx, "list", "", InvalidArgument, err)
	}
	challenges, err := loadProblems(s.repo)
	if err != nil {
		return nil, loadError(ctx, "list", "", err)
	}
	results := make([]Problem, 0, len(challenges))
	for _, c := range challenges {
		results = append(results, problem(c))
	}
	return results, nil
}

func (s *Service) Execute(ctx context.Context, slug string, command []string) (Execution, error) {
	if len(command) == 0 || command[0] == "" {
		return Execution{}, operationError(ctx, "exec", slug, InvalidArgument, errors.New("a toolbox command is required"))
	}
	c, err := s.load(ctx, "exec", slug, false)
	if err != nil {
		return Execution{}, err
	}
	project := ""
	if c.Compose != "" {
		state, err := runtime.ReadState(c)
		if err != nil {
			return Execution{}, operationError(ctx, "exec", slug, ExecutionFailed, err)
		}
		if _, err := runtime.Validate(ctx, c, state.Flag, false); err != nil {
			return Execution{}, operationError(ctx, "exec", slug, ValidationFailed, err)
		}
		project = state.Project
	}
	result, err := runtime.RunCommand(ctx, c, project, c.Solve.Image, command)
	if err != nil {
		return Execution{}, operationError(ctx, "exec", slug, ExecutionFailed, err)
	}
	return Execution{Stdout: result.Stdout, Stderr: result.Stderr, ExitCode: result.ExitCode}, nil
}

func (s *Service) Submit(ctx context.Context, slug, flag string) (Submission, error) {
	c, err := s.load(ctx, "submit", slug, false)
	if err != nil {
		return Submission{}, err
	}
	flag = strings.TrimSpace(flag)
	accepted := false
	if c.Compose == "" {
		digest := sha256.Sum256([]byte(flag))
		accepted = hex.EncodeToString(digest[:]) == strings.ToLower(c.Flag.SHA256)
	} else {
		state, err := runtime.ReadState(c)
		if err != nil {
			return Submission{}, operationError(ctx, "submit", slug, ExecutionFailed, err)
		}
		accepted = flag == state.Flag
	}
	return Submission{Slug: c.Slug, Accepted: accepted}, nil
}

var (
	_ Catalog = (*Service)(nil)
	_ Player  = (*Service)(nil)
)
