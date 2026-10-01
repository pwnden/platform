package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

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
		service["cap_drop"] = []string{"ALL"}
		existing, _ := service["security_opt"].([]any)
		security := make([]any, 0, len(existing)+1)
		for _, option := range existing {
			value, _ := option.(string)
			if !strings.HasPrefix(value, "no-new-privileges") {
				security = append(security, option)
			}
		}
		service["security_opt"] = append(security, "no-new-privileges:true")
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
	args := []string{"compose", "--project-directory", c.Dir, "-p", project, "-f", "-", "up", "-d", "--build", "--wait"}
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir, cmd.Env, cmd.Stdin = c.Dir, append(os.Environ(), "FLAG="+flag), bytes.NewReader(data)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose up: %w: %s", err, stderr.String())
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
	return nil
}

func checkIsolatedNetwork(network liveNetwork, project string) error {
	if network.Driver != "bridge" || !network.Internal || network.Options[gatewayIPv4] != "isolated" ||
		network.Options[gatewayIPv6] != "isolated" || len(network.Options) != 2 || network.Labels["com.docker.compose.project"] != project ||
		network.Name != project+"_"+network.Labels["com.docker.compose.network"] {
		return errors.New("problem network is not isolated; stop and restart its environment")
	}
	return nil
}
