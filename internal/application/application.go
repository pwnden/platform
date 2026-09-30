// Package application exposes platform operations to command and HTTP adapters.
package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/pwnden/platform/internal/challenge"
	"github.com/pwnden/platform/internal/runtime"
	"github.com/pwnden/platform/internal/verify"
)

// Consumers depend on the capabilities they use. Service implements all three.
type Validator interface {
	Validate(context.Context, string) (Validation, error)
}

type Runner interface {
	Run(context.Context, string) (RunInfo, error)
	Stop(context.Context, string) (StopInfo, error)
}

type Verifier interface {
	Verify(context.Context, string) (Verification, error)
}

type Kind string

const (
	KindFile    Kind = "file"
	KindService Kind = "service"
)

type Validation struct {
	Slug            string
	Kind            Kind
	ContractVersion int
	FileCount       int
	ServiceCount    int
	Patched         bool
}

type Endpoint struct {
	Name      string
	URL       string
	Published bool
}

type RunInfo struct {
	Slug      string
	Kind      Kind
	FileCount int
	Project   string
	Endpoints []Endpoint
}

// Verification is an author operation; Flag contains the recovered solution.
type Verification struct {
	Slug    string
	Flag    string
	Patched bool
}

type StopInfo struct {
	Slug string
}

// Service binds operations to one challenges checkout. Construction performs no IO.
type Service struct {
	repo string
}

func New(repo string) (*Service, error) {
	if repo == "" {
		return nil, &Error{Code: InvalidArgument, Operation: "configure", Cause: errors.New("challenges repository path is required")}
	}
	return &Service{repo: repo}, nil
}

func (s *Service) load(ctx context.Context, operation, slug string, cleanup bool) (*challenge.Loaded, error) {
	if !cleanup && ctx.Err() != nil {
		return nil, operationError(ctx, operation, slug, InvalidArgument, ctx.Err())
	}
	load := challenge.Load
	if cleanup {
		load = challenge.LoadForStop
	}
	c, err := load(s.repo, slug)
	if err != nil {
		return nil, loadError(ctx, operation, slug, err)
	}
	return c, nil
}

func kind(c *challenge.Loaded) Kind {
	if c.Compose == "" {
		return KindFile
	}
	return KindService
}

func (s *Service) Validate(ctx context.Context, slug string) (Validation, error) {
	c, err := s.load(ctx, "validate", slug, false)
	if err != nil {
		return Validation{}, err
	}
	result := Validation{Slug: c.Slug, Kind: kind(c), ContractVersion: c.Contract.Version, FileCount: len(c.Files), Patched: c.Patched != nil}
	if c.Compose != "" {
		cfg, err := runtime.Validate(ctx, c, "pwnden{validation}", false)
		if err != nil {
			return Validation{}, operationError(ctx, "validate", slug, ValidationFailed, err)
		}
		result.ServiceCount = len(cfg.Services)
		if c.Patched != nil {
			if _, err := runtime.Validate(ctx, c, "pwnden{validation}", true); err != nil {
				return Validation{}, operationError(ctx, "validate", slug, ValidationFailed, err)
			}
		}
	}
	return result, nil
}

func (s *Service) Run(ctx context.Context, slug string) (RunInfo, error) {
	c, err := s.load(ctx, "run", slug, false)
	if err != nil {
		return RunInfo{}, err
	}
	if err := runtime.Run(ctx, c); err != nil {
		return RunInfo{}, operationError(ctx, "run", slug, ExecutionFailed, err)
	}
	result := RunInfo{Slug: c.Slug, Kind: kind(c), FileCount: len(c.Files)}
	if c.Compose != "" {
		addresses, err := runtime.EndpointAddresses(ctx, c)
		if err != nil {
			err = errors.Join(err, runtime.Stop(context.WithoutCancel(ctx), c))
			return RunInfo{}, operationError(ctx, "run", slug, ExecutionFailed, err)
		}
		result.Project = runtime.Project(c)
		result.Endpoints = make([]Endpoint, 0, len(addresses))
		for _, address := range addresses {
			result.Endpoints = append(result.Endpoints, Endpoint{Name: address.Name, URL: address.URL, Published: address.Published})
		}
	}
	return result, nil
}

func (s *Service) Stop(ctx context.Context, slug string) (StopInfo, error) {
	c, err := s.load(context.WithoutCancel(ctx), "stop", slug, true)
	if err != nil {
		return StopInfo{}, err
	}
	if err := runtime.Stop(ctx, c); err != nil {
		return StopInfo{}, operationError(context.WithoutCancel(ctx), "stop", slug, CleanupFailed, err)
	}
	return StopInfo{Slug: c.Slug}, nil
}

func (s *Service) Verify(ctx context.Context, slug string) (Verification, error) {
	c, err := s.load(ctx, "verify", slug, false)
	if err != nil {
		return Verification{}, err
	}
	result, err := verify.Challenge(ctx, c)
	if err != nil {
		return Verification{}, operationError(ctx, "verify", slug, VerificationFailed, err)
	}
	return Verification{Slug: c.Slug, Flag: result.Flag, Patched: result.Patched}, nil
}

var (
	_ Validator = (*Service)(nil)
	_ Runner    = (*Service)(nil)
	_ Verifier  = (*Service)(nil)
)

// Error preserves the operation's original cause for errors.Is and errors.As.
type Error struct {
	Code      Code
	Operation string
	Slug      string
	Cause     error
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s %q (%s): %v", e.Operation, e.Slug, e.Code, e.Cause)
}

func (e *Error) Unwrap() error { return e.Cause }
