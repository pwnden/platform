package runtime

import (
	"context"
	"testing"

	"github.com/pwnden/platform/internal/challenge"
	"github.com/pwnden/platform/internal/testutil"
)

func TestEndpointsUsePrivateAddressesAndRunIdentity(t *testing.T) {
	c := fixture(t)
	c.Endpoints = []challenge.Endpoint{{Name: "web", Service: "app", Port: 8000, Protocol: "http"}, {Name: "database", Service: "app", Port: 5432, Protocol: "tcp"}}
	calls := testutil.Docker(t, testutil.WithIsolatedNetwork(Project(c), "network-id", testutil.Reply{Match: []string{"config"}, Out: configJSON(c, false)})...)
	if err := SaveState(c, State{Project: Project(c), Flag: "pwnden{first}"}); err != nil {
		t.Fatal(err)
	}
	first, err := EndpointAddresses(context.Background(), c)
	if err != nil || len(first) != 2 || first[0].URL != "http://app:8000" || !first[0].Proxied || first[0].Published || first[1].Proxied {
		t.Fatalf("private endpoints: %+v %v", first, err)
	}
	if err := SaveState(c, State{Project: Project(c), Flag: "pwnden{second}"}); err != nil {
		t.Fatal(err)
	}
	second, err := EndpointAddresses(context.Background(), c)
	if err != nil || first[0].Instance == second[0].Instance {
		t.Fatal("restart reused browser identity", err)
	}
	for _, call := range calls() {
		for _, arg := range call {
			if arg == "port" {
				t.Fatal("requested a host port", call)
			}
		}
	}
}
