package runtime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/pwnden/platform/internal/testutil"
)

func TestIsolationPreservesExerciseAndRemovesPortPublication(t *testing.T) {
	c := fixture(t)
	var document map[string]any
	_ = json.Unmarshal([]byte(configJSON(c, false)), &document)
	services := document["services"].(map[string]any)
	services["app"] = map[string]any{"image": "author-image", "command": []string{"run", "target"}, "environment": map[string]string{"FLAG": "pwnden{test}"},
		"ports": []any{map[string]any{"host_ip": "0.0.0.0", "target": 8000, "published": "8000", "protocol": "tcp"}}, "healthcheck": map[string]any{"test": []string{"CMD", "check"}}, "networks": map[string]any{"default": nil}}
	networks := document["networks"].(map[string]any)
	networks["database"] = map[string]any{"name": Project(c) + "_database"}
	raw, _ := json.Marshal(document)
	data, err := isolatedConfig(string(raw), c, Project(c))
	if err != nil {
		t.Fatal(err)
	}
	var isolated map[string]any
	if err := json.Unmarshal(data, &isolated); err != nil {
		t.Fatal(err)
	}
	app := isolated["services"].(map[string]any)["app"].(map[string]any)
	if _, ok := app["ports"]; ok {
		t.Fatal("host port publication survived")
	}
	if app["image"] != "author-image" || len(app["command"].([]any)) != 2 || app["environment"].(map[string]any)["FLAG"] != "pwnden{test}" || app["healthcheck"] == nil || app["networks"] == nil {
		t.Fatal("exercise configuration was changed", app)
	}
	for name, value := range isolated["networks"].(map[string]any) {
		network := value.(map[string]any)
		opts := network["driver_opts"].(map[string]any)
		if network["internal"] != true || opts[gatewayIPv4] != "isolated" || opts[gatewayIPv6] != "isolated" {
			t.Fatal("network left unisolated", name, network)
		}
	}
}

func TestIsolationRejectsHostBridgeAndRoutingOptions(t *testing.T) {
	c := fixture(t)
	for _, key := range []string{"com.docker.network.bridge.name", "com.docker.network.bridge.trusted_host_interfaces", gatewayIPv4, gatewayIPv6} {
		raw := strings.Replace(configJSON(c, false), `"default":{`, `"default":{"driver_opts":{"`+key+`":"host"},`, 1)
		if _, err := isolatedConfig(raw, c, Project(c)); err == nil {
			t.Fatal("unsafe bridge option accepted", key)
		}
	}
}

func TestLiveNetworkRejectsIncompleteIsolationAndOwnership(t *testing.T) {
	project := "pwnden-test-12345678"
	for _, change := range []string{"valid", "external", "ipv4", "ipv6", "driver", "owner", "name", "host bridge"} {
		t.Run(change, func(t *testing.T) {
			network := liveNetwork{Name: project + "_default", Driver: "bridge", Internal: true,
				Options: map[string]string{gatewayIPv4: "isolated", gatewayIPv6: "isolated"},
				Labels:  map[string]string{"com.docker.compose.project": project, "com.docker.compose.network": "default"}}
			switch change {
			case "external":
				network.Internal = false
			case "ipv4":
				network.Options[gatewayIPv4] = "nat"
			case "ipv6":
				delete(network.Options, gatewayIPv6)
			case "driver":
				network.Driver = "host"
			case "owner":
				network.Labels["com.docker.compose.project"] = "other-project"
			case "name":
				network.Name = "shared-network"
			case "host bridge":
				network.Options["com.docker.network.bridge.name"] = "docker0"
			}
			if err := checkIsolatedNetwork(network, project); (err == nil) != (change == "valid") {
				t.Fatalf("%s: %v", change, err)
			}
		})
	}
}

func TestStartupSuppliesIsolationOnStdinAndChecksDaemon(t *testing.T) {
	c := fixture(t)
	testutil.Docker(t, testutil.WithIsolatedNetwork(Project(c), "network-id",
		testutil.Reply{Match: []string{"config"}, Out: configJSON(c, false)},
		testutil.Reply{Match: []string{"-f", "-", "up"}, InputContains: []string{`"internal":true`, `"` + gatewayIPv4 + `":"isolated"`, `"` + gatewayIPv6 + `":"isolated"`}},
	)...)
	if err := Run(context.Background(), c); err != nil {
		t.Fatal(err)
	}
}

func TestToolRejectsPreviouslyUnisolatedNetwork(t *testing.T) {
	c := fixture(t)
	calls := testutil.Docker(t,
		testutil.Reply{Match: []string{"image", "inspect"}},
		testutil.Reply{Match: []string{"network", "ls"}, Out: "unsafe-network"},
		testutil.Reply{Match: []string{"network", "inspect"}, Out: `[{"Name":"unsafe","Driver":"bridge","Internal":false}]`},
	)
	if _, err := RunTool(context.Background(), c, Project(c), "image", []string{"solve"}); err == nil {
		t.Fatal("tool joined an unisolated environment")
	}
	for _, call := range calls() {
		if call[0] == "create" || call[0] == "start" {
			t.Fatal("tool container started", call)
		}
	}
}

func TestIsolationRequiresSupportedEngine(t *testing.T) {
	for _, version := range []string{"27.5.1", "28.0.0", "29.4.1", "unavailable"} {
		t.Run(version, func(t *testing.T) {
			testutil.Docker(t,
				testutil.Reply{Match: []string{"version"}, Out: version},
			)
			err := checkIsolationEngine(context.Background())
			supported := strings.HasPrefix(version, "28.") || strings.HasPrefix(version, "29.")
			if (err == nil) != supported {
				t.Fatalf("version %q: %v", version, err)
			}
		})
	}
}
