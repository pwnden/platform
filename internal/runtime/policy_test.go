package runtime

import (
	"context"
	"errors"
	"strings"
	"testing"
)

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
