package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestRejectsDeviceNamespaceHookAndSecurityOverrides(t *testing.T) {
	c := fixture(t)
	for _, declaration := range []string{
		`"gpus":"all"`, `"device_cgroup_rules":["a *:* rwm"]`,
		`"post_start":[{"privileged":true}]`, `"pre_start":[{"privileged":true}]`, `"pre_stop":[{"privileged":true}]`,
		`"userns_mode":"host"`, `"cgroup":"host"`, `"cgroup_parent":"/host"`, `"runtime":"custom"`,
		`"uts":"host"`,
		`"pid":"service:other"`, `"ipc":"container:other"`,
		`"security_opt":["seccomp:unconfined"]`, `"security_opt":["apparmor:unconfined"]`, `"security_opt":["systempaths:unconfined"]`,
	} {
		t.Run(declaration, func(t *testing.T) {
			raw := strings.Replace(configJSON(c, false), `"app":{}`, `"app":{`+declaration+`}`, 1)
			if _, err := isolatedConfig(raw, c, Project(c)); err == nil || (!strings.Contains(err.Error(), "privilege") && !strings.Contains(err.Error(), "security profile")) {
				t.Fatal("unsafe declaration accepted", declaration)
			}
		})
	}
}

func TestAuthorLimitsCannotOverrideRuntimePolicy(t *testing.T) {
	c := fixture(t)
	declaration := `"cpus":99,"mem_limit":"64g","memswap_limit":-1,"pids_limit":-1,"cpu_quota":-1,"oom_kill_disable":true,"read_only":false,"security_opt":["no-new-privileges:false"],"deploy":{"resources":{"limits":{"cpus":"99","memory":"64g"}}},"logging":{"driver":"syslog"},"tmpfs":["/tmp:size=1g","/run:size=1m"],"volumes":[{"type":"tmpfs","target":"/tmp"}],`
	raw := strings.Replace(configJSON(c, false), `"app":{}`, `"app":{`+strings.TrimSuffix(declaration, ",")+`}`, 1)
	data, err := isolatedConfig(raw, c, Project(c))
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	_ = json.Unmarshal(data, &cfg)
	app := cfg["services"].(map[string]any)["app"].(map[string]any)
	if app["cpus"] != float64(2) || app["mem_limit"] != "2g" || app["memswap_limit"] != "2g" || app["pids_limit"] != float64(256) || app["read_only"] != true || app["oom_kill_disable"] != false || app["cpu_quota"] != nil || app["deploy"].(map[string]any)["resources"] != nil {
		t.Fatal("author bypassed resource limits", app)
	}
	if app["logging"].(map[string]any)["driver"] != "local" || len(app["volumes"].([]any)) != 0 || len(app["tmpfs"].([]any)) != 2 || app["tmpfs"].([]any)[1] != "/tmp:"+tmpOptions {
		t.Fatal("author bypassed disk policy", app)
	}
}

func TestCommandBoundsOutputAndPreservesInput(t *testing.T) {
	for _, stream := range []string{"stdout", "stderr"} {
		t.Run(stream, func(t *testing.T) {
			out, stderr, err := command(context.Background(), "", nil, "python3", "-c", "import sys; sys."+stream+".buffer.write(b'x'*(9*1024*1024))")
			if !errors.Is(err, errOutputLimit) || len(out) > outputLimit || len(stderr) > outputLimit {
				t.Fatalf("output was not bounded: %d %d %v", len(out), len(stderr), err)
			}
		})
	}
	out, stderr, err := commandInput(context.Background(), "", nil, strings.NewReader("input"), "python3", "-c", "import sys; sys.stdout.write(sys.stdin.read()); sys.stderr.write('error')")
	if err != nil || out != "input" || stderr != "error" {
		t.Fatalf("stdin/stdout/stderr changed: %q %q %v", out, stderr, err)
	}
}

func TestRejectsOldAndModifiedLivePolicy(t *testing.T) {
	if err := checkServicePolicy(limitedHostConfig("none", false)); err != nil {
		t.Fatal("current service policy rejected", err)
	}
	for _, change := range []string{"memory", "cpu", "pids", "root", "security", "tmpfs", "logs"} {
		host := limitedHostConfig("none", false)
		switch change {
		case "memory":
			host.Memory = 0
		case "cpu":
			host.NanoCPUs = 0
		case "pids":
			host.PidsLimit = nil
		case "root":
			host.ReadonlyRootfs = false
		case "security":
			host.SecurityOpt = []string{"seccomp:unconfined"}
		case "tmpfs":
			host.Tmpfs["/tmp"] = "size=1g"
		case "logs":
			host.LogConfig.Type = "json-file"
		}
		if err := checkServicePolicy(host); err == nil {
			t.Fatal("changed live policy accepted", change)
		}
	}
}
