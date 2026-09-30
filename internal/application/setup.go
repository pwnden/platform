package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pwnden/platform/internal/challenge"
	"github.com/pwnden/platform/internal/installation"
	"github.com/pwnden/platform/internal/runtime"
)

type Bootstrapper interface {
	Setup(context.Context) (SetupInfo, error)
}

type Preparer interface {
	Prepare(context.Context) (SetupInfo, error)
}

type SetupInfo struct {
	ProblemCount    int
	ContractVersion int
}

type SetupService struct {
	manager *installation.Manager
}

func NewSetup() (*SetupService, error) {
	manager, err := installation.Default()
	if err != nil {
		return nil, operationError(context.Background(), "configure", "", SetupFailed, err)
	}
	return &SetupService{manager: manager}, nil
}

// Open uses the managed problem distribution created by setup.
func Open() (*Service, error) {
	manager, err := installation.Default()
	if err != nil {
		return nil, operationError(context.Background(), "configure", "", SetupFailed, err)
	}
	repo, err := manager.Resolve()
	if err != nil {
		code := SetupFailed
		if errors.Is(err, installation.ErrSetupRequired) {
			code = SetupRequired
		}
		return nil, operationError(context.Background(), "configure", "", code, err)
	}
	return New(repo)
}

func (s *SetupService) Setup(ctx context.Context) (result SetupInfo, err error) {
	defer func() {
		if err != nil {
			result = SetupInfo{}
			code := SetupFailed
			var version *challenge.VersionError
			if errors.As(err, &version) {
				code = IncompatibleContract
			}
			err = operationError(ctx, "setup", "", code, err)
		}
	}()
	if err := ctx.Err(); err != nil {
		return SetupInfo{}, err
	}
	if err := runtime.CheckEngine(ctx); err != nil {
		return SetupInfo{}, err
	}
	previous, err := s.manager.Resolve()
	if err != nil && !errors.Is(err, installation.ErrSetupRequired) {
		return SetupInfo{}, err
	}
	pending, err := s.manager.Stage(ctx)
	if err != nil {
		return SetupInfo{}, err
	}
	defer func() { err = errors.Join(err, pending.Discard()) }()
	if previous != "" && previous != pending.Root {
		problems, err := loadProblems(previous)
		if err != nil {
			return SetupInfo{}, err
		}
		for _, c := range problems {
			if c.Compose == "" {
				continue
			}
			if _, err := runtime.ReadState(c); err == nil {
				return SetupInfo{}, fmt.Errorf("stop %s before changing the installed problems: %w", c.Slug, runtime.ErrAlreadyRunning)
			} else if !errors.Is(err, os.ErrNotExist) {
				return SetupInfo{}, err
			}
		}
	}
	problems, err := loadProblems(pending.Root)
	if err != nil {
		return SetupInfo{}, err
	}
	for _, c := range problems {
		if err := runtime.Prepare(ctx, c); err != nil {
			return SetupInfo{}, err
		}
	}
	if err := ctx.Err(); err != nil {
		return SetupInfo{}, err
	}
	if err := pending.Activate(); err != nil {
		return SetupInfo{}, err
	}
	return SetupInfo{ProblemCount: len(problems), ContractVersion: problems[0].Contract.Version}, nil
}

// Prepare checks the same execution policy and tools as managed setup, using
// the live author checkout. It neither starts services nor changes installation.
func (s *Service) Prepare(ctx context.Context) (result SetupInfo, err error) {
	defer func() {
		if err != nil {
			result = SetupInfo{}
			code := SetupFailed
			var version *challenge.VersionError
			if errors.As(err, &version) {
				code = IncompatibleContract
			}
			err = operationError(ctx, "prepare", "", code, err)
		}
	}()
	if err := runtime.CheckEngine(ctx); err != nil {
		return SetupInfo{}, err
	}
	problems, err := loadProblems(s.repo)
	if err != nil {
		return SetupInfo{}, err
	}
	for _, c := range problems {
		if err := runtime.Prepare(ctx, c); err != nil {
			return SetupInfo{}, err
		}
	}
	if err := ctx.Err(); err != nil {
		return SetupInfo{}, err
	}
	return SetupInfo{ProblemCount: len(problems), ContractVersion: problems[0].Contract.Version}, nil
}

func loadProblems(root string) ([]*challenge.Loaded, error) {
	entries, err := os.ReadDir(filepath.Join(root, "challenges"))
	if err != nil {
		return nil, err
	}
	var problems []*challenge.Loaded
	for _, entry := range entries {
		if !entry.IsDir() && entry.Type()&os.ModeSymlink == 0 {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, "challenges", entry.Name(), "challenge.toml")); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		c, err := challenge.Load(root, entry.Name())
		if err != nil {
			return nil, err
		}
		problems = append(problems, c)
	}
	if len(problems) == 0 {
		return nil, errors.New("problem repository contains no problems")
	}
	return problems, nil
}

var _ Bootstrapper = (*SetupService)(nil)
var _ Preparer = (*Service)(nil)
