package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	dockerContainer "github.com/moby/moby/api/types/container"

	"github.com/pwnden/platform/internal/challenge"
)

const gatewayIPv4 = "com.docker.network.bridge.gateway_mode_ipv4"
const gatewayIPv6 = "com.docker.network.bridge.gateway_mode_ipv6"

// isolatedConfig preserves the resolved exercise configuration. Network policy
// and ingress belong to the consumer, so authors need no platform-specific glue.
func isolatedConfig(raw string, c *challenge.Loaded, project string) ([]byte, error) {
	var cfg Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, err
	}
	if err := checkConfig(c, &cfg, project); err != nil {
		return nil, err
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &document); err != nil {
		return nil, err
	}
	var networks map[string]map[string]any
	if err := json.Unmarshal(document["networks"], &networks); err != nil {
		return nil, err
	}
	for _, network := range networks {
		network["internal"] = true
		network["driver"] = "bridge"
		network["driver_opts"] = map[string]string{gatewayIPv4: "isolated", gatewayIPv6: "isolated"}
	}
	var services map[string]map[string]any
	if err := json.Unmarshal(document["services"], &services); err != nil {
		return nil, err
	}
	for _, service := range services {
		// Endpoints remain declarations. Host access uses a fixed-destination
		// Docker stream rather than opening a network path out of the exercise.
		delete(service, "ports")
		servicePolicy(service)
	}
	var volumes map[string]map[string]any
	if len(document["volumes"]) > 0 {
		if err := json.Unmarshal(document["volumes"], &volumes); err != nil {
			return nil, err
		}
		if len(volumes) > 8 {
			return nil, errors.New("problem has too many temporary volumes")
		}
		for _, volume := range volumes {
			volume["driver"] = "local"
			volume["driver_opts"] = map[string]string{"type": "tmpfs", "device": "tmpfs", "o": "size=256m,nosuid,nodev,nr_inodes=32768,mode=1777"}
		}
		document["volumes"], _ = json.Marshal(volumes)
	}
	document["networks"], _ = json.Marshal(networks)
	document["services"], _ = json.Marshal(services)
	return json.Marshal(document)
}

func startIsolated(ctx context.Context, c *challenge.Loaded, flag string, patched bool) error {
	raw, err := resolvedConfig(ctx, c, flag, patched)
	if err != nil {
		return err
	}
	project := Project(c)
	if patched {
		project += "-patched"
	}
	data, err := isolatedConfig(raw, c, project)
	if err != nil {
		return err
	}
	if err := checkIsolationEngine(ctx); err != nil {
		return err
	}
	// The full resolved configuration uses absolute repository paths. Passing it
	// on stdin keeps generated flags and configuration out of temporary files.
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	base := []string{"compose", "--project-directory", c.Dir, "-p", project, "-f", "-"}
	for _, step := range [][]string{{"pull", "--ignore-buildable"}, {"build"}} {
		_, stderr, err := commandInput(ctx, c.Dir, []string{"FLAG=" + flag}, bytes.NewReader(data), "docker", append(append([]string{}, base...), step...)...)
		if err != nil {
			return fmt.Errorf("prepare services: %w: %s", err, stderr)
		}
	}
	for name, service := range cfg.Services {
		image := service.Image
		if image == "" {
			image = project + "-" + name
		}
		mounts := append([]Mount{}, service.Volumes...)
		for _, entry := range service.Tmpfs {
			target, _, _ := strings.Cut(entry, ":")
			mounts = append(mounts, Mount{Type: "tmpfs", Target: target})
		}
		if err := checkImageStorage(ctx, c, image, mounts); err != nil {
			return err
		}
	}
	err = admitCreation(ctx, resourceCost{int64(len(cfg.Services)) * 2e9, int64(len(cfg.Services)) * (2 << 30), int64(len(cfg.Services)) * 256, int64(len(cfg.Services))}, func(ctx context.Context) error {
		_, stderr, err := commandInput(ctx, c.Dir, []string{"FLAG=" + flag}, bytes.NewReader(data), "docker", append(append([]string{}, base...), "create", "--no-build")...)
		if err != nil {
			return fmt.Errorf("docker compose create: %w: %s", err, stderr)
		}
		return checkServiceMounts(ctx, c, project, cfg.Volumes)
	})
	if err != nil {
		return err
	}
	args := append(base, "start", "--wait")
	_, stderr, err := commandInput(ctx, c.Dir, []string{"FLAG=" + flag}, bytes.NewReader(data), "docker", args...)
	if err != nil {
		return fmt.Errorf("docker compose start: %w: %s", err, stderr)
	}
	return checkLiveNetworks(ctx, c, project)
}

func checkIsolationEngine(ctx context.Context) error {
	out, stderr, err := command(ctx, "", nil, "docker", "version", "--format", "{{.Server.Version}}")
	if err != nil {
		return fmt.Errorf("read Docker Engine version: %w: %s", err, stderr)
	}
	major, err := strconv.Atoi(strings.SplitN(strings.TrimSpace(out), ".", 2)[0])
	if err != nil || major < 28 {
		return errors.New("Docker Engine 28 or newer is required for isolated problem networks")
	}
	return nil
}

func checkLiveServicePolicy(ctx context.Context, c *challenge.Loaded, project string) error {
	out, stderr, err := command(ctx, c.Dir, nil, "docker", "container", "ls", "--all", "--filter", "label=com.docker.compose.project="+project, "--format", "{{.ID}}")
	if err != nil {
		return fmt.Errorf("find problem containers: %w: %s", err, stderr)
	}
	ids := strings.Fields(out)
	if len(ids) == 0 {
		return errors.New("problem has no service containers")
	}
	out, stderr, err = command(ctx, c.Dir, nil, "docker", append([]string{"container", "inspect"}, ids...)...)
	if err != nil {
		return fmt.Errorf("inspect problem policy: %w: %s", err, stderr)
	}
	var containers []dockerContainer.InspectResponse
	if err := json.Unmarshal([]byte(out), &containers); err != nil {
		return err
	}
	if len(containers) != len(ids) {
		return errors.New("incomplete problem container inspection")
	}
	for _, container := range containers {
		if container.Config == nil || container.Config.Labels["com.docker.compose.project"] != project {
			return errors.New("problem container ownership changed")
		}
		if container.Config.Labels["pwnden.kind"] == "connector" {
			if container.Name != "/"+project+"-connector" || container.NetworkSettings == nil {
				return errors.New("problem connector ownership changed")
			}
			network := container.NetworkSettings.Networks[project+"_"+c.Solve.Network]
			if network == nil {
				return errors.New("problem connector network changed")
			}
			if err := checkConnector(container, c, project, network.NetworkID); err != nil {
				return err
			}
			continue
		}
		if container.Config.Labels["pwnden.runtime-policy"] != servicePolicyVersion {
			return errors.New("problem resource policy is outdated; stop and restart its environment")
		}
		if err := checkServicePolicy(container.HostConfig); err != nil {
			return err
		}
	}
	return nil
}

type liveNetwork struct {
	ID, Name, Driver string
	Internal         bool
	Options          map[string]string
	Labels           map[string]string
}

func checkLiveNetworks(ctx context.Context, c *challenge.Loaded, project string) error {
	out, stderr, err := command(ctx, c.Dir, nil, "docker", "network", "ls", "--no-trunc", "--filter", "label=com.docker.compose.project="+project, "--format", "{{.ID}}")
	if err != nil {
		return fmt.Errorf("find problem networks: %w: %s", err, stderr)
	}
	ids := strings.Fields(out)
	if len(ids) == 0 {
		return errors.New("problem has no isolated network; stop and restart its environment")
	}
	out, stderr, err = command(ctx, c.Dir, nil, "docker", append([]string{"network", "inspect"}, ids...)...)
	if err != nil {
		return fmt.Errorf("inspect problem networks: %w: %s", err, stderr)
	}
	var networks []liveNetwork
	if err := json.Unmarshal([]byte(out), &networks); err != nil {
		return err
	}
	if len(networks) != len(ids) {
		return errors.New("incomplete problem network inspection")
	}
	for _, network := range networks {
		if err := checkIsolatedNetwork(network, project); err != nil {
			return err
		}
	}
	return checkLiveServicePolicy(ctx, c, project)
}

func checkIsolatedNetwork(network liveNetwork, project string) error {
	if network.Driver != "bridge" || !network.Internal || network.Options[gatewayIPv4] != "isolated" ||
		network.Options[gatewayIPv6] != "isolated" || len(network.Options) != 2 || network.Labels["com.docker.compose.project"] != project ||
		network.Name != project+"_"+network.Labels["com.docker.compose.network"] {
		return errors.New("problem network is not isolated; stop and restart its environment")
	}
	return nil
}
