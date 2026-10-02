package runtime

import (
	"context"
	"errors"
	"net"
	"strconv"

	"github.com/moby/moby/client"
	"github.com/pwnden/platform/internal/challenge"
)

type resolvedEndpoint struct {
	project, network, address, target string
	port                              int
}

// CheckEndpoint observes the running endpoint through the Engine API. Compose
// policy is validated at preparation; browsing checks live isolation and run
// identity on every request, including requests on reused HTTP connections.
func CheckEndpoint(ctx context.Context, c *challenge.Loaded, name, instance, target string) error {
	engine, err := terminalEngine()
	if err != nil {
		return err
	}
	defer engine.Close()
	resolved, err := resolveEndpoint(ctx, c, engine, name, instance)
	if err == nil && resolved.target != target {
		return errors.New("declared HTTP endpoint changed")
	}
	return err
}

func resolveEndpoint(ctx context.Context, c *challenge.Loaded, engine *client.Client, name, instance string) (resolvedEndpoint, error) {
	state, err := ReadState(c)
	if err != nil {
		return resolvedEndpoint{}, err
	}
	var endpoint challenge.Endpoint
	for _, candidate := range c.Endpoints {
		if candidate.Name == name && candidate.Protocol == "http" {
			endpoint = candidate
		}
	}
	if endpoint.Name == "" || instance != endpointInstance(state) {
		return resolvedEndpoint{}, errors.New("HTTP endpoint is unavailable or belongs to an earlier run")
	}
	networks, err := engine.NetworkList(ctx, client.NetworkListOptions{Filters: client.Filters{
		"label": {"com.docker.compose.project=" + state.Project: true},
	}})
	if err != nil {
		return resolvedEndpoint{}, err
	}
	allowed := make(map[string]bool)
	networkID := ""
	for _, item := range networks.Items {
		inspected, err := engine.NetworkInspect(ctx, item.ID, client.NetworkInspectOptions{})
		if err != nil {
			return resolvedEndpoint{}, err
		}
		n := inspected.Network
		if err := checkIsolatedNetwork(liveNetwork{ID: n.ID, Name: n.Name, Driver: n.Driver, Internal: n.Internal, Options: n.Options, Labels: n.Labels}, state.Project); err != nil {
			return resolvedEndpoint{}, err
		}
		allowed[n.ID] = true
		if n.Labels["com.docker.compose.network"] == c.Solve.Network {
			if networkID != "" {
				return resolvedEndpoint{}, errors.New("multiple solve networks")
			}
			networkID = n.ID
		}
	}
	if networkID == "" {
		return resolvedEndpoint{}, errors.New("problem has no isolated solve network")
	}
	containers, err := engine.ContainerList(ctx, client.ContainerListOptions{All: true, Filters: client.Filters{
		"label": {"com.docker.compose.project=" + state.Project: true, "com.docker.compose.service=" + endpoint.Service: true, "com.docker.compose.oneoff=False": true},
	}})
	if err != nil {
		return resolvedEndpoint{}, err
	}
	if len(containers.Items) != 1 {
		return resolvedEndpoint{}, errors.New("endpoint requires one running service container")
	}
	inspected, err := engine.ContainerInspect(ctx, containers.Items[0].ID, client.ContainerInspectOptions{})
	if err != nil {
		return resolvedEndpoint{}, err
	}
	service := inspected.Container
	if service.Config == nil || service.State == nil || !service.State.Running || service.NetworkSettings == nil ||
		(service.State.Health != nil && service.State.Health.Status != "healthy") ||
		service.Config.Labels["com.docker.compose.project"] != state.Project || service.Config.Labels["com.docker.compose.service"] != endpoint.Service {
		return resolvedEndpoint{}, errors.New("endpoint service does not belong to this running problem")
	}
	address := ""
	for _, settings := range service.NetworkSettings.Networks {
		if settings == nil || !allowed[settings.NetworkID] {
			return resolvedEndpoint{}, errors.New("endpoint service is connected to another network")
		}
		if settings.NetworkID == networkID {
			address = settings.IPAddress.String()
		}
	}
	if net.ParseIP(address) == nil {
		return resolvedEndpoint{}, errors.New("endpoint service is not connected to the solve network")
	}
	current, err := ReadState(c)
	if err != nil || endpointInstance(current) != instance {
		return resolvedEndpoint{}, errors.New("problem run changed during endpoint observation")
	}
	return resolvedEndpoint{project: state.Project, network: networkID, address: address,
		target: "http://" + net.JoinHostPort(endpoint.Service, strconv.Itoa(endpoint.Port)), port: endpoint.Port}, nil
}
