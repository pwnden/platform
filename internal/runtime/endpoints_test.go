package runtime

import "testing"

func TestPublishedAddress(t *testing.T) {
	for _, address := range []string{"127.0.0.1:49153", "0.0.0.0:8000", "[::1]:49153"} {
		if err := checkPublishedAddress(address); err != nil {
			t.Errorf("valid Docker address %q rejected: %v", address, err)
		}
	}
	// Compose can report this when a declared port was not actually published.
	for _, address := range []string{"invalid IP:0", "", "127.0.0.1:0", "127.0.0.1:65536"} {
		if err := checkPublishedAddress(address); err == nil {
			t.Errorf("invalid Docker address %q accepted", address)
		}
	}
}
