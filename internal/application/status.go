package application

import (
	"context"

	"github.com/pwnden/platform/internal/runtime"
)

type Observer interface {
	Status(context.Context, string) (RunStatus, error)
}

type RunStatus struct {
	Slug      string
	Kind      Kind
	State     string
	Endpoints []Endpoint
}

func (s *Service) Status(ctx context.Context, slug string) (RunStatus, error) {
	// Observation remains usable after a distribution file was removed.
	c, err := s.load(ctx, "status", slug, true)
	if err != nil {
		return RunStatus{}, err
	}
	if err := ctx.Err(); err != nil {
		return RunStatus{}, operationError(ctx, "status", slug, ExecutionFailed, err)
	}
	state, err := runtime.RunStatus(ctx, c)
	if err != nil {
		return RunStatus{}, operationError(ctx, "status", slug, ExecutionFailed, err)
	}
	result := RunStatus{Slug: slug, Kind: kind(c), State: state, Endpoints: make([]Endpoint, 0)}
	if state == "running" {
		addresses, err := runtime.EndpointAddresses(ctx, c)
		if err != nil {
			return RunStatus{}, operationError(ctx, "status", slug, ExecutionFailed, err)
		}
		for _, address := range addresses {
			result.Endpoints = append(result.Endpoints, Endpoint{Name: address.Name, URL: address.URL, Published: address.Published})
		}
	}
	return result, nil
}

var _ Observer = (*Service)(nil)
