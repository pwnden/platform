package httpapi

import (
	"context"
	"net"
	"net/http"
	"strings"
)

// Problem origins are <slug>.localhost:<port>. Browsers resolve that name to
// loopback, but a test host's resolver may not, so tests dial the loopback
// listener directly while still sending the problem's own Host.
func loopbackDial(ctx context.Context, network, address string) (net.Conn, error) {
	if host, port, err := net.SplitHostPort(address); err == nil && strings.HasSuffix(host, ".localhost") {
		address = net.JoinHostPort("127.0.0.1", port)
	}
	return (&net.Dialer{}).DialContext(ctx, network, address)
}

func init() {
	if transport, ok := http.DefaultTransport.(*http.Transport); ok {
		transport.DialContext = loopbackDial
	}
}
