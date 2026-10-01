package application

import (
	"context"
	"net"

	"github.com/pwnden/platform/internal/runtime"
)

// EndpointDialer connects only a declared HTTP endpoint of a particular run.
// Consumers provide names and an instance identity, never arbitrary destinations.
type EndpointDialer interface {
	DialEndpoint(context.Context, string, string, string) (net.Conn, error)
}

func (s *Service) DialEndpoint(ctx context.Context, slug, name, instance string) (net.Conn, error) {
	c, err := s.load(ctx, "browser", slug, false)
	if err != nil {
		return nil, err
	}
	connection, err := runtime.DialEndpoint(ctx, c, name, instance)
	if err != nil {
		return nil, operationError(ctx, "browser", slug, ExecutionFailed, err)
	}
	return connection, nil
}

var _ EndpointDialer = (*Service)(nil)
