package runtime

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net"
	"strconv"

	"github.com/pwnden/platform/internal/challenge"
)

type EndpointAddress struct {
	Name      string
	URL       string
	Published bool
	Proxied   bool
	Instance  string
}

// EndpointAddresses describes private services. The local server supplies HTTP
// ingress through Docker streams, without publishing any problem container port.
func EndpointAddresses(ctx context.Context, c *challenge.Loaded) ([]EndpointAddress, error) {
	if len(c.Endpoints) == 0 {
		return nil, nil
	}
	state, err := ReadState(c)
	if err != nil {
		return nil, err
	}
	_, err = Validate(ctx, c, state.Flag, false)
	if err != nil {
		return nil, err
	}
	if err := checkLiveNetworks(ctx, c, state.Project); err != nil {
		return nil, err
	}
	instance := fmt.Sprintf("%x", sha256.Sum256([]byte(state.Project+"\x00"+state.Flag)))
	addresses := make([]EndpointAddress, 0, len(c.Endpoints))
	for _, endpoint := range c.Endpoints {
		address := net.JoinHostPort(endpoint.Service, strconv.Itoa(endpoint.Port))
		addresses = append(addresses, EndpointAddress{
			Name: endpoint.Name, URL: endpoint.Protocol + "://" + address,
			Proxied: endpoint.Protocol == "http", Instance: instance,
		})
	}
	return addresses, nil
}
