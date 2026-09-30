package application

import (
	"context"
	"testing"
)

func TestTerminalRejectsInvalidSizeAndStoppedService(t *testing.T) {
	s, c := fixture(t, true)
	for _, size := range [][2]int{{1, 24}, {80, 0}, {501, 24}, {80, 201}} {
		_, err := s.OpenTerminal(context.Background(), c.Slug, size[0], size[1])
		requireCode(t, err, InvalidArgument)
	}
	_, err := s.OpenTerminal(context.Background(), c.Slug, 80, 24)
	requireCode(t, err, NotRunning)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = s.OpenTerminal(ctx, c.Slug, 80, 24)
	requireCode(t, err, Canceled)
}
