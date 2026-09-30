package runtime

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/pwnden/platform/internal/challenge"
)

type EndpointAddress struct {
	Name      string
	URL       string
	Published bool
}

// EndpointAddresses resolves Docker-assigned host ports without assuming a host OS.
func EndpointAddresses(ctx context.Context, c *challenge.Loaded) ([]EndpointAddress, error) {
	if len(c.Endpoints) == 0 {
		return nil, nil
	}
	state, err := ReadState(c)
	if err != nil {
		return nil, err
	}
	cfg, err := Validate(ctx, c, state.Flag, false)
	if err != nil {
		return nil, err
	}
	addresses := make([]EndpointAddress, 0, len(c.Endpoints))
	for _, endpoint := range c.Endpoints {
		address := net.JoinHostPort(endpoint.Service, strconv.Itoa(endpoint.Port))
		published := false
		for _, port := range cfg.Services[endpoint.Service].Ports {
			if port.Target == endpoint.Port && (port.Protocol == "tcp" || port.Protocol == "") {
				published = true
				break
			}
		}
		if published {
			out, stderr, err := command(ctx, c.Dir, []string{"FLAG=" + state.Flag}, "docker",
				composeArgs(c, state.Project, []string{c.Compose}, "port", "--protocol", "tcp",
					endpoint.Service, strconv.Itoa(endpoint.Port))...)
			if err != nil {
				return nil, fmt.Errorf("endpoint %q: %w: %s", endpoint.Name, err, stderr)
			}
			address = strings.TrimSpace(out)
			if err := checkPublishedAddress(address); err != nil {
				return nil, fmt.Errorf("endpoint %q: %w", endpoint.Name, err)
			}
		}
		addresses = append(addresses, EndpointAddress{
			Name: endpoint.Name, URL: endpoint.Protocol + "://" + address, Published: published,
		})
	}
	return addresses, nil
}

func checkPublishedAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err == nil {
		var number int
		number, err = strconv.Atoi(port)
		if err == nil && net.ParseIP(host) != nil && number > 0 && number <= 65535 {
			return nil
		}
	}
	return fmt.Errorf("Docker did not return a published IP and port: %q", address)
}
