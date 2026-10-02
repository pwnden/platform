package application

import (
	"context"
	"errors"
	"os"

	"github.com/pwnden/platform/internal/challenge"
	"github.com/pwnden/platform/internal/runtime"
)

type Code string

const (
	WorkspaceFull        Code = "workspace_full"
	TerminalBusy         Code = "terminal_busy"
	WorkspaceBusy        Code = "workspace_busy"
	InvalidArgument      Code = "invalid_argument"
	NotFound             Code = "not_found"
	IncompatibleContract Code = "incompatible_contract"
	AlreadyRunning       Code = "already_running"
	NotRunning           Code = "not_running"
	ValidationFailed     Code = "validation_failed"
	ExecutionFailed      Code = "execution_failed"
	VerificationFailed   Code = "verification_failed"
	CleanupFailed        Code = "cleanup_failed"
	Canceled             Code = "canceled"
	DeadlineExceeded     Code = "deadline_exceeded"
	SetupRequired        Code = "setup_required"
	SetupFailed          Code = "setup_failed"
	StorageFailed        Code = "storage_failed"
)

func loadError(ctx context.Context, operation, slug string, cause error) error {
	code := InvalidArgument
	var version *challenge.VersionError
	if errors.As(cause, &version) {
		code = IncompatibleContract
	} else if errors.Is(cause, os.ErrNotExist) {
		code = NotFound
	}
	return operationError(ctx, operation, slug, code, cause)
}

func operationError(ctx context.Context, operation, slug string, code Code, cause error) error {
	cause = errors.Join(cause, ctx.Err())
	// Cleanup failure takes precedence: resources may still exist after cancellation.
	if errors.Is(cause, runtime.ErrCleanupFailed) {
		code = CleanupFailed
	} else {
		switch {
		case errors.Is(cause, context.DeadlineExceeded):
			code = DeadlineExceeded
		case errors.Is(cause, context.Canceled):
			code = Canceled
		case errors.Is(cause, runtime.ErrAlreadyRunning):
			code = AlreadyRunning
		case errors.Is(cause, runtime.ErrNotRunning):
			code = NotRunning
		}
	}
	return &Error{Code: code, Operation: operation, Slug: slug, Cause: cause}
}
