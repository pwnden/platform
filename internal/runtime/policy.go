package runtime

import (
	"fmt"
	"strings"

	dockerContainer "github.com/moby/moby/api/types/container"
)

const (
	outputLimit          = 8 * 1024 * 1024
	tmpOptions           = "rw,exec,nosuid,nodev,size=128m"
	servicePolicyVersion = "docker-v2"
)

func toolUser(writable bool) string {
	return "10001:10001"
}

func toolEnv() []string {
	return []string{"HOME=/home/pwnden", "XDG_CACHE_HOME=/tmp/.cache", "POCL_CACHE_DIR=/tmp/pocl-cache", "POCL_MAX_PTHREAD_COUNT=2", "OMP_NUM_THREADS=2"}
}

func toolHomeOptions(writable bool) string {
	user := strings.Split(toolUser(writable), ":")
	return "rw,nosuid,nodev,uid=" + user[0] + ",gid=" + user[1] + ",size=64m"
}

func limitedHostConfig(network string, connector bool) *dockerContainer.HostConfig {
	pidLimit, memory, cpus := int64(256), int64(2<<30), containerCPUs()
	if connector {
		pidLimit, memory, cpus = 64, 128<<20, 5e8
	}
	host := &dockerContainer.HostConfig{
		NetworkMode: dockerContainer.NetworkMode(network), ReadonlyRootfs: true,
		CgroupnsMode: "private", CapDrop: []string{"ALL"}, SecurityOpt: []string{"no-new-privileges"},
		Resources: dockerContainer.Resources{NanoCPUs: cpus, Memory: memory, MemorySwap: memory, PidsLimit: &pidLimit},
		LogConfig: dockerContainer.LogConfig{Type: "local", Config: map[string]string{"max-size": "10m", "max-file": "3"}},
	}
	if !connector {
		host.Tmpfs = map[string]string{"/tmp": tmpOptions}
	}
	return host
}

func toolOptions(writable bool) []string {
	options := []string{"--user", toolUser(writable), "--read-only", "--cgroupns", "private",
		"--cpus", fmt.Sprint(containerCPUs() / 1e9), "--memory", "2g", "--memory-swap", "2g", "--pids-limit", "256",
		"--tmpfs", "/tmp:" + tmpOptions, "--tmpfs", "/home/pwnden:" + toolHomeOptions(writable),
		"--log-driver", "local", "--log-opt", "max-size=10m", "--log-opt", "max-file=3"}
	for _, env := range toolEnv() {
		options = append(options, "--env", env)
	}
	return options
}

func servicePolicy(service map[string]any) {
	for _, key := range []string{"cpu_period", "cpu_quota", "cpu_rt_period", "cpu_rt_runtime", "mem_reservation", "mem_swappiness", "oom_score_adj"} {
		delete(service, key)
	}
	if deploy, ok := service["deploy"].(map[string]any); ok {
		delete(deploy, "resources")
	}
	service["cpus"], service["mem_limit"], service["memswap_limit"] = containerCPUs()/1e9, "2g", "2g"
	service["pids_limit"], service["oom_kill_disable"], service["read_only"] = 256, false, true
	service["shm_size"] = "64m"
	service["cgroup"], service["cap_drop"], service["security_opt"] = "private", []string{"ALL"}, []string{"no-new-privileges:true"}
	service["logging"] = map[string]any{"driver": "local", "options": map[string]string{"max-size": "10m", "max-file": "3"}}
	labels, _ := service["labels"].(map[string]any)
	if labels == nil {
		labels = map[string]any{}
	}
	labels["pwnden.runtime-policy"] = servicePolicyVersion
	labels[managedLabel] = "true"
	service["labels"] = labels
	volumes := make([]any, 0)
	for _, volume := range sliceField(service["volumes"]) {
		mount, _ := volume.(map[string]any)
		if mount["type"] == "volume" {
			mount["volume"] = map[string]any{"nocopy": true}
		}
		if mount["type"] == "tmpfs" {
			mount["tmpfs"] = map[string]any{"size": workspaceSize, "mode": 01777}
		}
		if mount["target"] != "/tmp" && mount["target"] != "/tmp/" {
			volumes = append(volumes, volume)
		}
	}
	service["volumes"] = volumes
	tmpfs := make([]any, 0)
	for _, mount := range sliceField(service["tmpfs"]) {
		entry, _ := mount.(string)
		if entry != "/tmp" && entry != "/tmp/" && !strings.HasPrefix(entry, "/tmp:") && !strings.HasPrefix(entry, "/tmp/:") {
			target, _, _ := strings.Cut(entry, ":")
			tmpfs = append(tmpfs, target+":rw,exec,nosuid,nodev,size=256m,nr_inodes=32768")
		}
	}
	service["tmpfs"] = append(tmpfs, "/tmp:"+tmpOptions)
}

func checkServicePolicy(host *dockerContainer.HostConfig) error {
	if host == nil || !host.ReadonlyRootfs || host.Privileged || len(host.CapAdd) != 0 ||
		len(host.CapDrop) != 1 || host.CapDrop[0] != "ALL" || host.CgroupnsMode != "private" ||
		containerCPUs() == 0 || host.NanoCPUs != containerCPUs() || host.Memory != 2<<30 || host.MemorySwap != host.Memory ||
		host.PidsLimit == nil || *host.PidsLimit != 256 || host.Tmpfs["/tmp"] != tmpOptions ||
		host.LogConfig.Type != "local" || host.LogConfig.Config["max-size"] != "10m" || host.LogConfig.Config["max-file"] != "3" {
		return fmt.Errorf("problem resource policy is outdated; stop and restart its environment")
	}
	if len(host.SecurityOpt) != 1 || (host.SecurityOpt[0] != "no-new-privileges" && host.SecurityOpt[0] != "no-new-privileges:true") {
		return fmt.Errorf("problem security policy is outdated; stop and restart its environment")
	}
	return nil
}

func sliceField(value any) []any {
	items, _ := value.([]any)
	return items
}

func securityOptionAllowed(option string) bool {
	switch option {
	case "no-new-privileges", "no-new-privileges:true", "no-new-privileges:false", "no-new-privileges=true", "no-new-privileges=false":
		return true
	}
	return false
}
