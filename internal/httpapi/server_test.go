package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/pwnden/platform/internal/application"
)

type announcedURL chan string

func (w announcedURL) Write(content []byte) (int, error) {
	w <- strings.TrimSpace(string(content))
	return len(content), nil
}

func startServer(t *testing.T, b Backend) (*url.URL, context.CancelFunc, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	announced := make(announcedURL, 1)
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, b, announced, io.Discard) }()
	t.Cleanup(cancel)
	select {
	case value := <-announced:
		u, err := url.Parse(value)
		if err != nil {
			t.Fatal(err)
		}
		if u.Hostname() != "127.0.0.1" || u.Port() == "" || len(u.Fragment) != 64 {
			t.Fatal("invalid loopback session URL")
		}
		return u, cancel, done
	case err := <-done:
		t.Fatalf("startup failed: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("no server URL")
	}
	return nil, nil, nil
}

func TestServerLoopbackSessionAndGracefulShutdown(t *testing.T) {
	u, cancel, done := startServer(t, fakeBackend{list: func(context.Context) ([]application.Problem, error) { return nil, nil }})
	client := &http.Client{Timeout: 3 * time.Second}
	r, _ := http.NewRequest("GET", "http://"+u.Host+BasePath+"/problems", nil)
	r.Header.Set("Authorization", "Bearer "+u.Fragment)
	r.Header.Set("Origin", "http://"+u.Host)
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	response, err := client.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var result ProblemList
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil || response.StatusCode != 200 || result.Problems == nil {
		t.Fatalf("live API: %d %v", response.StatusCode, err)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown did not finish")
	}
	client.CloseIdleConnections()
	if response, err := client.Do(r); err == nil {
		response.Body.Close()
		t.Fatal("listener remained open")
	}
}

func TestShutdownWaitsForDisconnectedHandlerCleanup(t *testing.T) {
	entered := make(chan struct{})
	cleaning := make(chan struct{})
	finish := make(chan struct{})
	cleanupFailure := &application.Error{Code: application.CleanupFailed, Cause: errors.New("pwnden{private_flag} /private/host/path")}
	b := fakeBackend{
		run: func(ctx context.Context, _ string) (application.RunInfo, error) {
			close(entered)
			<-ctx.Done()
			// Simulate startup completing at the cancellation boundary.
			return application.RunInfo{Slug: "example", Kind: application.KindService}, nil
		},
		stop: func(ctx context.Context, slug string) (application.StopInfo, error) {
			if ctx.Err() != nil {
				t.Error("shutdown canceled independent cleanup")
			}
			close(cleaning)
			<-finish
			return application.StopInfo{}, cleanupFailure
		},
	}
	u, cancelServer, done := startServer(t, b)
	requestCtx, cancelRequest := context.WithCancel(context.Background())
	defer cancelRequest()
	r, _ := http.NewRequestWithContext(requestCtx, "POST", "http://"+u.Host+BasePath+"/problems/example/run", nil)
	r.Header.Set("Authorization", "Bearer "+u.Fragment)
	client := &http.Client{Timeout: 3 * time.Second}
	requestDone := make(chan struct{})
	go func() {
		response, _ := client.Do(r)
		if response != nil {
			response.Body.Close()
		}
		close(requestDone)
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("backend was not called")
	}
	cancelRequest()
	<-requestDone
	select {
	case <-cleaning:
	case <-time.After(3 * time.Second):
		t.Fatal("disconnected start was not cleaned")
	}
	cancelServer()
	select {
	case err := <-done:
		t.Fatalf("server returned before cleanup: %v", err)
	case <-time.After(40 * time.Millisecond):
	}
	close(finish)
	select {
	case err := <-done:
		if !errors.Is(err, cleanupFailure) {
			t.Fatalf("cleanup cause lost: %v", err)
		}
		if strings.Contains(err.Error(), "private_flag") || strings.Contains(err.Error(), "/private") {
			t.Fatal("server diagnostic exposed cleanup internals")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cleanup completion did not release server")
	}
}

type brokenAnnouncement struct{}

func (brokenAnnouncement) Write([]byte) (int, error) { return 0, errDelivery }

func TestServerStartupFailureAndCancellation(t *testing.T) {
	if err := Serve(context.Background(), fakeBackend{}, brokenAnnouncement{}, io.Discard); !errors.Is(err, errDelivery) {
		t.Fatal("startup output failure lost", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Serve(ctx, fakeBackend{}, io.Discard, io.Discard); !errors.Is(err, context.Canceled) {
		t.Fatal("canceled server started", err)
	}
}
