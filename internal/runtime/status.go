package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pwnden/platform/internal/challenge"
)

// RunStatus observes resources without creating them or changing stored state.
// unavailable means a recorded run needs attention before another start.
func RunStatus(ctx context.Context, c *challenge.Loaded) (string, error) {
	if c.Compose == "" {
		return "ready", ctx.Err()
	}
	state, err := ReadState(c)
	if errors.Is(err, os.ErrNotExist) {
		return "stopped", nil
	}
	if err != nil {
		return "", err
	}
	cfg, err := Validate(ctx, c, state.Flag, false)
	if err != nil {
		return "", err
	}
	if err := checkLiveNetworks(ctx, c, state.Project); err != nil {
		return "", err
	}
	out, stderr, err := command(ctx, c.Dir, []string{"FLAG=" + state.Flag}, "docker",
		composeArgs(c, state.Project, []string{c.Compose}, "ps", "--all", "--format", "json")...)
	if err != nil {
		return "", fmt.Errorf("observe problem containers: %w: %s", err, stderr)
	}
	return containerStatus(out, cfg)
}

type container struct {
	Service string
	State   string
	Health  string
}

func containerStatus(output string, cfg *Config) (string, error) {
	// Compose versions emit either one array or newline-delimited objects.
	d := json.NewDecoder(strings.NewReader(output))
	var containers []container
	for {
		var raw json.RawMessage
		err := d.Decode(&raw)
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if len(raw) > 0 && raw[0] == '[' {
			var batch []container
			if err := json.Unmarshal(raw, &batch); err != nil {
				return "", err
			}
			containers = append(containers, batch...)
		} else {
			var item container
			if err := json.Unmarshal(raw, &item); err != nil || item.Service == "" || item.State == "" {
				return "", errors.New("invalid container status response")
			}
			containers = append(containers, item)
		}
	}
	seen := make(map[string]bool)
	for _, item := range containers {
		if _, declared := cfg.Services[item.Service]; !declared {
			continue
		}
		if item.State != "running" || (item.Health != "" && item.Health != "healthy") {
			return "unavailable", nil
		}
		seen[item.Service] = true
	}
	if len(seen) == 0 || len(seen) != len(cfg.Services) {
		return "unavailable", nil
	}
	return "running", nil
}
