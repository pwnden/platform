package verify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/pwnden/platform/internal/challenge"
	"github.com/pwnden/platform/internal/runtime"
)

type Result struct {
	Flag    string
	Patched bool
}

func Challenge(ctx context.Context, c *challenge.Loaded) (result Result, err error) {
	if c.Compose == "" {
		output, runErr := runtime.RunTool(ctx, c, "", c.Solve.Image, c.Solve.Command)
		if runErr != nil {
			return result, runErr
		}
		flag := strings.TrimSpace(output)
		digest := sha256.Sum256([]byte(flag))
		if hex.EncodeToString(digest[:]) != strings.ToLower(c.Flag.SHA256) {
			return result, errors.New("file challenge flag hash does not match")
		}
		return Result{Flag: flag}, nil
	}

	state, err := runtime.ReadState(c)
	if err != nil {
		return result, fmt.Errorf("read run state; run challenge first: %w", err)
	}
	if _, err := runtime.Validate(ctx, c, state.Flag, false); err != nil {
		return result, err
	}
	output, err := runtime.RunTool(ctx, c, state.Project, c.Solve.Image, c.Solve.Command)
	if err != nil {
		return result, err
	}
	if strings.TrimSpace(output) != state.Flag {
		return result, errors.New("solve did not return the current run flag")
	}
	result.Flag = state.Flag
	if c.Patched == nil {
		return result, nil
	}

	patchedProject, err := runtime.StartPatched(ctx, c, state.Flag)
	if err != nil {
		return result, err
	}
	defer func() {
		if downErr := runtime.Down(context.Background(), c, patchedProject, state.Flag, true); downErr != nil {
			err = errors.Join(err, downErr)
		}
	}()
	output, attackErr := runtime.RunTool(ctx, c, patchedProject, c.Solve.Image, c.Solve.Command)
	if strings.TrimSpace(output) == state.Flag {
		return result, errors.New("solve still succeeds against patched challenge")
	}
	if attackErr != nil {
		var rejected *runtime.ToolExitError
		if !errors.As(attackErr, &rejected) || rejected.Code != c.Contract.AttackRejectedExit {
			return result, fmt.Errorf("patched attack execution: %w", attackErr)
		}
	}
	image := c.Patched.Image
	if image == "" {
		image = c.Solve.Image
	}
	if _, err := runtime.RunTool(ctx, c, patchedProject, image, c.Patched.Check); err != nil {
		return result, fmt.Errorf("patched functional check: %w", err)
	}
	result.Patched = true
	return result, nil
}
