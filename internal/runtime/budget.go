package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/flock"
	dockerContainer "github.com/moby/moby/api/types/container"
)

const managedLabel = "pwnden.managed"
const budgetInfoFormat = `{"ID":{{json .ID}},"NCPU":{{.NCPU}},"MemTotal":{{.MemTotal}}}`

type resourceCost struct {
	CPUs, Memory, PIDs int64
	Containers         int64
}

func (r resourceCost) add(other resourceCost) resourceCost {
	return resourceCost{r.CPUs + other.CPUs, r.Memory + other.Memory, r.PIDs + other.PIDs, r.Containers + other.Containers}
}

// Operators may tighten the per-container ceiling without increasing it.
func containerCPUs() int64 {
	switch os.Getenv("PWNDEN_CONTAINER_CPUS") {
	case "", "2":
		return 2e9
	case "1":
		return 1e9
	default:
		return 0
	}
}

func toolboxCost() resourceCost   { return resourceCost{containerCPUs(), 2 << 30, 256, 1} }
func connectorCost() resourceCost { return resourceCost{5e8, 128 << 20, 64, 1} }

func runtimeBudget(cpus int64, memory int64) (resourceCost, error) {
	if containerCPUs() == 0 {
		return resourceCost{}, errors.New("PWNDEN_CONTAINER_CPUS must be 1 or 2")
	}
	limits := resourceCost{8e9, 8 << 30, 1024, 12}
	for _, setting := range []struct {
		name       string
		target     *int64
		multiplier int64
	}{
		{"PWNDEN_RUNTIME_CPUS", &limits.CPUs, 1e9},
		{"PWNDEN_RUNTIME_MEMORY_MIB", &limits.Memory, 1 << 20},
		{"PWNDEN_RUNTIME_PIDS", &limits.PIDs, 1},
		{"PWNDEN_RUNTIME_CONTAINERS", &limits.Containers, 1},
	} {
		if value := os.Getenv(setting.name); value != "" {
			if strings.Trim(value, "0123456789") != "" {
				return resourceCost{}, fmt.Errorf("%s must be a positive integer", setting.name)
			}
			number, err := strconv.ParseInt(value, 10, 32)
			if err != nil || number <= 0 {
				return resourceCost{}, fmt.Errorf("%s must be a positive integer", setting.name)
			}
			*setting.target = number * setting.multiplier
		}
	}
	// Budgets describe this daemon, including when Docker is remote.
	limits.CPUs = min(limits.CPUs, cpus*1e9)
	limits.Memory = min(limits.Memory, memory)
	return limits, nil
}

func checkBudget(used, requested, limits resourceCost) error {
	total := used.add(requested)
	if total.CPUs > limits.CPUs || total.Memory > limits.Memory || total.PIDs > limits.PIDs || total.Containers > limits.Containers {
		return fmt.Errorf("pwnden runtime budget exceeded: would use %.1f/%.1f CPUs, %d/%d MiB memory, %d/%d PIDs, %d/%d containers; close another environment or adjust PWNDEN_RUNTIME_*", float64(total.CPUs)/1e9, float64(limits.CPUs)/1e9, total.Memory>>20, limits.Memory>>20, total.PIDs, limits.PIDs, total.Containers, limits.Containers)
	}
	return nil
}

// admitCreation holds the shared daemon lock only through creation. Created
// containers reserve their ceilings before their processes are started.
func admitCreation(ctx context.Context, cost resourceCost, create func(context.Context) error) error {
	return withAdmission(ctx, func(ctx context.Context, check func(resourceCost) error) error {
		if err := check(cost); err != nil {
			return err
		}
		return create(ctx)
	})
}

func withAdmission(ctx context.Context, action func(context.Context, func(resourceCost) error) error) error {
	return withRuntimeLock(ctx, true, action)
}

func withRuntimeLock(ctx context.Context, accounting bool, action func(context.Context, func(resourceCost) error) error) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	out, stderr, err := command(ctx, "", nil, "docker", "info", "--format", budgetInfoFormat)
	if err != nil {
		return fmt.Errorf("read runtime budget daemon: %w: %s", err, stderr)
	}
	var info struct {
		ID             string
		NCPU, MemTotal int64
	}
	if err := json.Unmarshal([]byte(out), &info); err != nil {
		return err
	}
	if info.ID == "" || info.NCPU <= 0 || info.MemTotal <= 0 {
		return errors.New("Docker did not report its resource capacity")
	}
	dir := filepath.Join("/tmp", "pwnden-runtime-"+strconv.Itoa(os.Geteuid()))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	digest := sha256.Sum256([]byte(info.ID))
	lock := flock.New(filepath.Join(dir, fmt.Sprintf("%x.lock", digest[:16])), flock.SetPermissions(0600))
	defer lock.Close()
	locked, err := lock.TryLockContext(ctx, 50*time.Millisecond)
	if err != nil {
		return err
	}
	if !locked {
		return errors.New("runtime admission lock unavailable")
	}
	if !accounting {
		return action(ctx, nil)
	}
	limits, err := runtimeBudget(info.NCPU, info.MemTotal)
	if err != nil {
		return err
	}
	used, err := liveResourceCost(ctx)
	if err != nil {
		return err
	}
	return action(ctx, func(cost resourceCost) error { return checkBudget(used, cost, limits) })
}

func managedContainer(labels, name string) bool {
	for _, entry := range strings.Split(labels, ",") {
		key, value, _ := strings.Cut(entry, "=")
		if key == managedLabel || key == "pwnden.runtime-policy" ||
			(key == "pwnden.kind" && (value == "tool" || value == "terminal" || value == "connector")) {
			return true
		}
	}
	return strings.HasPrefix(name, "pwnden-author-tool-") || strings.HasPrefix(name, "pwnden-tool-")
}

func liveResourceCost(ctx context.Context) (resourceCost, error) {
	out, stderr, err := command(ctx, "", nil, "docker", "container", "ls", "--all", "--format", "{{json .}}")
	if err != nil {
		return resourceCost{}, fmt.Errorf("list runtime budget containers: %w: %s", err, stderr)
	}
	var used resourceCost
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		var row struct{ ID, Labels, Names string }
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return used, err
		}
		if !managedContainer(row.Labels, row.Names) {
			continue
		}
		out, stderr, err := command(ctx, "", nil, "docker", "container", "inspect", "--format", "{{json .HostConfig}}", row.ID)
		if err != nil {
			// A completed --rm command can disappear while its snapshot is read.
			remaining, _, lookupErr := command(ctx, "", nil, "docker", "container", "ls", "--all", "--filter", "id="+row.ID, "--format", "{{.ID}}")
			if lookupErr == nil && strings.TrimSpace(remaining) == "" {
				continue
			}
			return used, fmt.Errorf("inspect runtime budget container: %w: %s", err, stderr)
		}
		var host dockerContainer.HostConfig
		if err := json.Unmarshal([]byte(out), &host); err != nil {
			return used, err
		}
		if host.NanoCPUs <= 0 || host.Memory <= 0 || host.PidsLimit == nil || *host.PidsLimit <= 0 {
			return used, fmt.Errorf("managed container %s has no resource ceilings; stop and restart its environment", row.Names)
		}
		used = used.add(resourceCost{host.NanoCPUs, host.Memory, *host.PidsLimit, 1})
	}
	return used, nil
}
