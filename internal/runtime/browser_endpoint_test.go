package runtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/moby/moby/client"
	"github.com/pwnden/platform/internal/challenge"
	"github.com/pwnden/platform/internal/testutil"
)

func TestBrowserEndpointChecksLiveIdentityAndIsolationWithoutCompose(t *testing.T) {
	for _, test := range []struct{ name string }{{"running"}, {"restarted"}, {"stopped"}, {"unhealthy"}, {"external network"}, {"foreign service network"}, {"missing solve network"}, {"undeclared endpoint"}, {"run changed during observation"}, {"old policy"}, {"removed memory limit"}} {
		t.Run(test.name, func(t *testing.T) {
			c := fixture(t)
			c.Endpoints = []challenge.Endpoint{{Name: "web", Service: "app", Port: 8000, Protocol: "http"}}
			calls := testutil.Docker(t)
			if err := os.WriteFile(os.Getenv("PWNDEN_TEST_LOG"), nil, 0600); err != nil {
				t.Fatal(err)
			}
			state := State{Project: Project(c), Flag: "pwnden{first}"}
			if err := SaveState(c, state); err != nil {
				t.Fatal(err)
			}
			instance := endpointInstance(state)
			if test.name == "restarted" {
				instance = "previous-run"
			}
			network := map[string]any{"Id": "network-id", "Name": state.Project + "_" + c.Solve.Network, "Driver": "bridge", "Internal": true,
				"Options": map[string]string{gatewayIPv4: "isolated", gatewayIPv6: "isolated"},
				"Labels":  map[string]string{"com.docker.compose.project": state.Project, "com.docker.compose.network": c.Solve.Network}}
			if test.name == "external network" {
				network["Internal"] = false
			}
			attached := "network-id"
			if test.name == "foreign service network" {
				attached = "another-problem-network"
			}
			serviceState := map[string]any{"Running": test.name != "stopped"}
			if test.name == "unhealthy" {
				serviceState["Health"] = map[string]string{"Status": "unhealthy"}
			}
			labels := map[string]string{"com.docker.compose.project": state.Project, "com.docker.compose.service": "app", "pwnden.runtime-policy": servicePolicyVersion}
			host := limitedHostConfig("network-id", false)
			if test.name == "old policy" {
				delete(labels, "pwnden.runtime-policy")
			}
			if test.name == "removed memory limit" {
				host.Memory = 0
			}
			service := map[string]any{"Id": "container-id", "Config": map[string]any{"Labels": labels}, "HostConfig": host,
				"State": serviceState, "NetworkSettings": map[string]any{"Networks": map[string]any{"default": map[string]string{"NetworkID": attached, "IPAddress": "172.31.0.2"}}}}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				path := strings.TrimPrefix(r.URL.Path, "/v1.56")
				var value any
				switch path {
				case "/networks":
					value = []any{network}
					if test.name == "missing solve network" {
						value = []any{}
					}
				case "/networks/network-id":
					value = network
				case "/containers/json":
					value = []any{map[string]string{"Id": "container-id"}}
				case "/containers/container-id/json":
					value = service
					if test.name == "run changed during observation" {
						if err := SaveState(c, State{Project: state.Project, Flag: "pwnden{next}"}); err != nil {
							t.Error(err)
						}
					}
				default:
					t.Errorf("unexpected Engine request %s", path)
					w.WriteHeader(404)
					return
				}
				_ = json.NewEncoder(w).Encode(value)
			}))
			defer server.Close()
			engine, err := client.New(client.WithHost(server.URL), client.WithAPIVersion("1.56"))
			if err != nil {
				t.Fatal(err)
			}
			defer engine.Close()
			name := "web"
			if test.name == "undeclared endpoint" {
				name = "private"
			}
			result, err := resolveEndpoint(context.Background(), c, engine, name, instance)
			if test.name == "running" {
				if err != nil || result.address != "172.31.0.2" || result.port != 8000 {
					t.Fatalf("resolve: %+v %v", result, err)
				}
			} else if err == nil {
				t.Fatal("unavailable or unsafe endpoint accepted")
			}
			if len(calls()) != 0 {
				t.Fatal("browsing launched Docker CLI", calls())
			}
		})
	}
}
