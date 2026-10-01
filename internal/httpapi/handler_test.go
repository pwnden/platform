package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pwnden/platform/internal/application"
)

type fakeBackend struct {
	list     func(context.Context) ([]application.Problem, error)
	detail   func(context.Context, string) (application.ProblemDetail, error)
	guidance func(context.Context, string, string) (string, error)
	download func(context.Context, string, string) (application.Download, error)
	status   func(context.Context, string) (application.RunStatus, error)
	run      func(context.Context, string) (application.RunInfo, error)
	stop     func(context.Context, string) (application.StopInfo, error)
	submit   func(context.Context, string, string) (application.Submission, error)
	terminal func(context.Context, string, int, int) (application.TerminalSession, error)
}

func (b fakeBackend) OpenTerminal(ctx context.Context, slug string, cols, rows int) (application.TerminalSession, error) {
	return b.terminal(ctx, slug, cols, rows)
}

func (b fakeBackend) List(ctx context.Context) ([]application.Problem, error) { return b.list(ctx) }
func (b fakeBackend) Detail(ctx context.Context, slug string) (application.ProblemDetail, error) {
	return b.detail(ctx, slug)
}
func (b fakeBackend) Guidance(ctx context.Context, slug, id string) (string, error) {
	return b.guidance(ctx, slug, id)
}
func (b fakeBackend) Download(ctx context.Context, slug, id string) (application.Download, error) {
	return b.download(ctx, slug, id)
}
func (b fakeBackend) Status(ctx context.Context, slug string) (application.RunStatus, error) {
	if b.status == nil {
		return application.RunStatus{Slug: slug, Kind: application.KindFile, State: "ready"}, nil
	}
	return b.status(ctx, slug)
}
func (b fakeBackend) Run(ctx context.Context, slug string) (application.RunInfo, error) {
	return b.run(ctx, slug)
}
func (b fakeBackend) Stop(ctx context.Context, slug string) (application.StopInfo, error) {
	if b.stop == nil {
		return application.StopInfo{Slug: slug}, nil
	}
	return b.stop(ctx, slug)
}
func (b fakeBackend) Submit(ctx context.Context, slug, flag string) (application.Submission, error) {
	return b.submit(ctx, slug, flag)
}

const testHost = "127.0.0.1:12345"
const testToken = "test-session-secret"

func request(method, path, content string) *http.Request {
	r := httptest.NewRequest(method, "http://"+testHost+path, strings.NewReader(content))
	r.Header.Set("Authorization", "Bearer "+testToken)
	if content != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	return r
}

func TestHTTPPlayerOperations(t *testing.T) {
	b := fakeBackend{
		list: func(context.Context) ([]application.Problem, error) {
			return []application.Problem{{Slug: "example", Title: "Example", Kind: application.KindService}}, nil
		},
		run: func(ctx context.Context, slug string) (application.RunInfo, error) {
			if _, ok := ctx.Deadline(); !ok {
				t.Error("operation deadline absent")
			}
			return application.RunInfo{Slug: slug, Kind: application.KindService, Project: "private-project", Endpoints: []application.Endpoint{{Name: "web", URL: "http://127.0.0.1:9000", Published: true}, {Name: "db", URL: "tcp://private:5432"}}}, nil
		},
		stop: func(_ context.Context, slug string) (application.StopInfo, error) {
			return application.StopInfo{Slug: slug}, nil
		},
		submit: func(_ context.Context, slug, flag string) (application.Submission, error) {
			return application.Submission{Slug: slug, Accepted: flag == "correct"}, nil
		},
	}
	h := newHandler(context.Background(), b, testHost, testToken, io.Discard)
	for _, test := range []struct{ method, path, content, expected string }{
		{"GET", BasePath + "/problems", "", `{"problems":[{"slug":"example","title":"Example","category":"","kind":"service"}]}`},
		{"POST", BasePath + "/problems/example/run", "", `{"slug":"example","kind":"service","file_count":0,"endpoints":[{"name":"web","url":"http://127.0.0.1:9000"}]}`},
		{"DELETE", BasePath + "/problems/example/run", "", `{"slug":"example"}`},
		{"POST", BasePath + "/problems/example/submissions", `{"flag":"wrong"}`, `{"slug":"example","accepted":false}`},
		{"POST", BasePath + "/problems/example/submissions", `{"flag":"correct"}`, `{"slug":"example","accepted":true}`},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, request(test.method, test.path, test.content))
		if w.Code != 200 || strings.TrimSpace(w.Body.String()) != test.expected {
			t.Fatalf("%s %s: %d %s", test.method, test.path, w.Code, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Content-Type") != "application/json" || w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatal("response headers", w.Header())
		}
	}
}

func TestHTTPRejectsUntrustedAndMalformedRequests(t *testing.T) {
	// No backend callback is installed: every invalid request must stop at transport.
	h := newHandler(context.Background(), fakeBackend{}, testHost, testToken, io.Discard)
	for _, test := range []struct {
		name   string
		modify func(*http.Request)
		status int
		code   string
	}{
		{"missing token", func(r *http.Request) { r.Header.Del("Authorization") }, 401, "unauthorized"},
		{"wrong token", func(r *http.Request) { r.Header.Set("Authorization", "Bearer other") }, 401, "unauthorized"},
		{"duplicate token", func(r *http.Request) { r.Header.Add("Authorization", "Bearer "+testToken) }, 401, "unauthorized"},
		{"host", func(r *http.Request) { r.Host = "outside.example" }, 403, "forbidden"},
		{"origin", func(r *http.Request) { r.Header.Set("Origin", "http://127.0.0.1:9999") }, 403, "forbidden"},
		{"null origin", func(r *http.Request) { r.Header.Set("Origin", "null") }, 403, "forbidden"},
		{"duplicate origin", func(r *http.Request) {
			r.Header.Add("Origin", "http://"+testHost)
			r.Header.Add("Origin", "http://"+testHost)
		}, 403, "forbidden"},
		{"same site", func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "same-site") }, 403, "forbidden"},
		{"query", func(r *http.Request) { r.URL.RawQuery = "repo=outside" }, 400, "invalid_request"},
		{"encoded path", func(r *http.Request) { r.URL.RawPath = "/api/v1/%70roblems" }, 400, "invalid_request"},
		{"method", func(r *http.Request) { r.Method = "OPTIONS" }, 405, "method_not_allowed"},
		{"unknown route", func(r *http.Request) { r.URL.Path = BasePath + "/author/verify" }, 404, "not_found"},
		{"bad slug", func(r *http.Request) { r.URL.Path = BasePath + "/problems/../run"; r.Method = "POST" }, 400, "invalid_argument"},
		{"unexpected body", func(r *http.Request) { r.Body = io.NopCloser(strings.NewReader(" ")) }, 400, "invalid_request"},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := request("GET", BasePath+"/problems", "")
			test.modify(r)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			assertError(t, w, test.status, test.code)
		})
	}
	for _, content := range []string{`{}`, `null`, `[]`, `{"flag":null}`, `{"flag":3}`, `{"flag":" "}`, `{"Flag":"x"}`, `{"flag":"x","other":0}`, `{"flag":"x","flag":"y"}`, `{"flag":"x"} {}`, `{"flag":`, "{\"flag\":\"\xff\"}"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, request("POST", BasePath+"/problems/example/submissions", content))
		assertError(t, w, 400, "invalid_request")
	}
	r := request("POST", BasePath+"/problems/example/submissions", `{"flag":"`+strings.Repeat("x", 4096)+`"}`)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	assertError(t, w, 413, "payload_too_large")
	for _, media := range []string{"", "text/plain", "application/json; other=yes"} {
		r := request("POST", BasePath+"/problems/example/submissions", `{"flag":"x"}`)
		r.Header.Set("Content-Type", media)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		assertError(t, w, 415, "unsupported_media_type")
	}
}

func assertError(t *testing.T, w *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	var failure ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &failure); err != nil || w.Code != status || failure.Error.Code != code {
		t.Fatalf("expected %d %s, got %d %s (%v)", status, code, w.Code, w.Body.String(), err)
	}
}

func TestMutationSerializationAndWaitingCancellation(t *testing.T) {
	entered := make(chan string, 4)
	release := make(chan struct{})
	var calls atomic.Int32
	b := fakeBackend{run: func(ctx context.Context, slug string) (application.RunInfo, error) {
		calls.Add(1)
		entered <- slug
		<-release
		return application.RunInfo{Slug: slug, Kind: application.KindFile}, nil
	}}
	h := newHandler(context.Background(), b, testHost, testToken, io.Discard)
	done := make(chan struct{})
	go func() {
		h.ServeHTTP(httptest.NewRecorder(), request("POST", BasePath+"/problems/first/run", ""))
		close(done)
	}()
	if <-entered != "first" {
		t.Fatal("first mutation missing")
	}
	ctx, cancel := context.WithCancel(context.Background())
	waited := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, request("POST", BasePath+"/problems/first/run", "").WithContext(ctx))
		waited <- w
	}()
	other := make(chan struct{})
	go func() {
		h.ServeHTTP(httptest.NewRecorder(), request("POST", BasePath+"/problems/second/run", ""))
		close(other)
	}()
	select {
	case slug := <-entered:
		if slug != "second" {
			t.Fatal("same problem overlapped")
		}
	case <-time.After(time.Second):
		t.Fatal("independent problem blocked")
	}
	cancel()
	select {
	case w := <-waited:
		assertError(t, w, 503, "canceled")
	case <-time.After(time.Second):
		t.Fatal("queued cancellation blocked")
	}
	close(release)
	<-done
	<-other
	if calls.Load() != 2 || len(h.locks) != 0 {
		t.Fatalf("calls=%d locks=%d", calls.Load(), len(h.locks))
	}
}

type deliveryFailure struct {
	*httptest.ResponseRecorder
	failWrite bool
}

var errDelivery = errors.New("delivery failed")

func (w deliveryFailure) Write(p []byte) (int, error) {
	if w.failWrite {
		return 0, errDelivery
	}
	return w.ResponseRecorder.Write(p)
}
func (w deliveryFailure) FlushError() error { return errDelivery }

func TestNewRunDeliveryFailureCleanupAndExistingOwnership(t *testing.T) {
	for _, mode := range []string{"write", "flush", "canceled", "existing", "cleanup failure", "canceled cleanup failure"} {
		t.Run(mode, func(t *testing.T) {
			var stopped bool
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cleanupErr := &application.Error{Code: application.CleanupFailed, Cause: errors.New("pwnden{secret} /private")}
			b := fakeBackend{run: func(context.Context, string) (application.RunInfo, error) {
				if mode == "existing" {
					return application.RunInfo{}, &application.Error{Code: application.AlreadyRunning}
				}
				if strings.HasPrefix(mode, "canceled") {
					cancel()
				}
				return application.RunInfo{Slug: "example", Kind: application.KindService}, nil
			}, stop: func(ctx context.Context, slug string) (application.StopInfo, error) {
				stopped = true
				if ctx.Err() != nil {
					t.Error("cleanup context canceled")
				}
				if strings.Contains(mode, "cleanup failure") {
					return application.StopInfo{}, cleanupErr
				}
				return application.StopInfo{Slug: slug}, nil
			}}
			var diagnostics bytes.Buffer
			h := newHandler(context.Background(), b, testHost, testToken, &diagnostics)
			w := deliveryFailure{httptest.NewRecorder(), mode == "write"}
			h.ServeHTTP(w, request("POST", BasePath+"/problems/example/run", "").WithContext(ctx))
			if stopped != (mode != "existing") {
				t.Fatalf("cleanup=%v mode=%s", stopped, mode)
			}
			if strings.Contains(mode, "cleanup failure") && !errors.Is(h.cleanupError(), cleanupErr) {
				t.Fatal("cleanup cause lost")
			}
			if mode == "canceled" {
				assertError(t, w.ResponseRecorder, 503, "canceled")
			}
			if mode == "canceled cleanup failure" {
				assertError(t, w.ResponseRecorder, 500, "cleanup_failed")
			}
			if strings.Contains(diagnostics.String(), "secret") || strings.Contains(diagnostics.String(), "private") {
				t.Fatal("cleanup diagnostics leak")
			}
		})
	}
}

func TestBootstrapKeepsTokenOutOfHTTPResponses(t *testing.T) {
	h := newHandler(context.Background(), fakeBackend{}, testHost, testToken, io.Discard)
	w := httptest.NewRecorder()
	r := request("GET", "/", "")
	r.Header.Del("Authorization")
	h.ServeHTTP(w, r)
	if w.Code != 200 || strings.Contains(w.Body.String(), testToken) || !strings.Contains(w.Body.String(), `id="app"`) {
		t.Fatal("player entry point missing or disclosed token")
	}
	h.closeAdmission()
	w = httptest.NewRecorder()
	h.ServeHTTP(w, request("GET", BasePath+"/problems", ""))
	assertError(t, w, 503, "canceled")
}
