package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

const operationTimeout = 15 * time.Minute

// serverFailure preserves local causes for Go callers without exposing process
// output or private paths through the CLI's ordinary error printing.
type serverFailure struct{ cause error }

func (e *serverFailure) Error() string { return "local server failed; check cleanup diagnostics" }
func (e *serverFailure) Unwrap() error { return e.cause }

// Serve owns a loopback listener until cancellation. Completed problem runs
// survive shutdown; interrupted starts and failed response delivery are cleaned.
func Serve(ctx context.Context, backend Backend, stdout, stderr io.Writer) error {
	return serve(ctx, backend, stdout, stderr, nil)
}

func serve(ctx context.Context, backend Backend, stdout, stderr io.Writer, frontend *Frontend) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return err
	}
	token := hex.EncodeToString(secret)
	base, cancel := context.WithCancel(ctx)
	defer cancel()
	handler := newHandler(base, backend, listener.Addr().String(), token, stderr)
	handler.frontend = frontend
	server := &http.Server{
		Handler: handler, ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 10 * time.Second, WriteTimeout: operationTimeout + 10*time.Second,
		IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10,
		// net/http panic diagnostics can include arbitrary backend values.
		// Keep request bodies and credentials out of the server's diagnostics.
		ErrorLog:    log.New(io.Discard, "", 0),
		BaseContext: func(net.Listener) context.Context { return base },
	}
	if _, err := fmt.Fprintf(stdout, "http://%s/#%s\n", listener.Addr(), token); err != nil {
		return err
	}
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	select {
	case err = <-served:
	case <-ctx.Done():
	}
	handler.closeAdmission()
	cancel()
	shutdown, stop := context.WithTimeout(context.Background(), 10*time.Second)
	defer stop()
	shutdownErr := server.Shutdown(shutdown)
	if shutdownErr != nil {
		shutdownErr = errors.Join(shutdownErr, server.Close())
	}
	// Shutdown tracks connections. A disconnected handler can still be performing
	// independent Docker cleanup, so wait for our own admitted handlers as well.
	handler.active.Wait()
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	if failure := errors.Join(err, shutdownErr, handler.cleanupError()); failure != nil {
		return &serverFailure{cause: failure}
	}
	return nil
}
