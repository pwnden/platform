package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/pwnden/platform/internal/application"
)

func TestDevelopmentAssetProxyKeepsAPIBoundaries(t *testing.T) {
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Error("player credentials reached Vite")
		}
		if r.URL.RawQuery != "vue&type=style" {
			t.Error("Vite transform query was changed")
		}
		w.Header().Set("Content-Type", "text/javascript")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Set-Cookie", "private=value")
		io.WriteString(w, "export default 'live';")
	}))
	defer upstream.Close()
	frontend, err := DevelopmentFrontend(upstream.URL, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	h := newHandler(context.Background(), fakeBackend{list: func(context.Context) ([]application.Problem, error) { return nil, nil }}, testHost, testToken, io.Discard)
	h.frontend = frontend
	r := request("GET", "/src/App.vue?vue&type=style", "")
	r.Header.Set("Cookie", "secret=value")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || calls != 1 || w.Header().Get("Content-Type") != "text/javascript" || !strings.Contains(w.Header().Get("Content-Security-Policy"), "'nonce-"+frontend.nonce+"'") || w.Header().Get("Access-Control-Allow-Origin") != "" || w.Header().Get("Set-Cookie") != "" {
		t.Fatal("invalid development asset response", w.Code, w.Header())
	}
	if !strings.Contains(w.Header().Get("Content-Security-Policy"), "frame-src http://127.0.0.1:*;") {
		t.Fatal("development must permit only local problem frames")
	}
	for _, test := range []struct {
		path string
		code int
	}{
		{BasePath + "/problems", 401}, {BasePath + "/problems?vue", 400},
		{"/__open-in-editor?file=go.mod", 404}, {"/@fs/etc/passwd", 404}, {"/__vite_hmr?token=dev", 403},
	} {
		r := request("GET", test.path, "")
		r.Header.Del("Authorization")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != test.code || calls != 1 {
			t.Fatalf("route boundary %s: %d", test.path, w.Code)
		}
	}
	for _, change := range []func(*http.Request){
		func(r *http.Request) { r.Host = "other.invalid" },
		func(r *http.Request) { r.Header.Set("Origin", "http://other.invalid") },
		func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") },
	} {
		r := request("GET", "/src/App.vue", "")
		change(r)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 403 || calls != 1 {
			t.Fatal("development bypassed origin checks")
		}
	}
}

func TestDevelopmentPlayerRoutesServeEntry(t *testing.T) {
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Errorf("invalid entry request: %s", r.URL.Path)
		}
		if r.URL.RawQuery != "q=nmap&page=2" {
			t.Error("page query was changed")
		}
		w.Header().Set("Content-Type", "text/html")
		if r.Method != "HEAD" {
			io.WriteString(w, "<!doctype html><title>Player</title>")
		}
	}))
	defer upstream.Close()
	frontend, err := DevelopmentFrontend(upstream.URL, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	h := newHandler(context.Background(), fakeBackend{}, testHost, testToken, io.Discard)
	h.frontend = frontend
	for _, path := range []string{"/", "/challenges", "/challenges/", "/challenges/diagnostic-port", "/challenges/diagnostic-port/", "/settings", "/courses/basics/lesson-1", "/unknown", "/challenges/x/extra", "/challenges/" + strings.Repeat("a", 41)} {
		for _, method := range []string{"GET", "HEAD"} {
			r := request(method, path+"?q=nmap&page=2", "")
			r.Header.Set("Cookie", "secret=value")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 200 || w.Header().Get("Content-Type") != "text/html" || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("page %s %s: %d", method, path, w.Code)
			}
			if r.URL.Path != path {
				t.Fatal("proxy mutated the original page request")
			}
			if method == "GET" && !strings.HasPrefix(w.Body.String(), "<!doctype html>") {
				t.Fatal("page did not serve the HTML entry")
			}
		}
	}
	before := calls
	for _, test := range []struct {
		method, path string
		code         int
	}{
		{"GET", "/challenges/%64iagnostic-port", 400},
		{"GET", "/missing.css", 404},
		{"GET", "/favicon.ico", 404},
		{"GET", "/.git/config", 404},
		{"GET", "/@fs/etc/passwd", 404},
		{"GET", "/__open-in-editor", 404},
		{"GET", "/api", 404},
		{"GET", "/assets", 404},
		{"GET", "/courses//lesson", 404},
		{"GET", "/api/v1/missing", 404},
		{"POST", "/challenges/diagnostic-port", 405},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, request(test.method, test.path, ""))
		if w.Code != test.code || calls != before {
			t.Fatalf("route boundary %s %s: %d", test.method, test.path, w.Code)
		}
	}
}

func TestDevelopmentMissingFilesDoNotReceiveSPAEntry(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, "<!doctype html><title>Vite fallback</title>")
	}))
	defer upstream.Close()
	frontend, err := DevelopmentFrontend(upstream.URL, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	h := newHandler(context.Background(), fakeBackend{}, testHost, testToken, io.Discard)
	h.frontend = frontend
	for _, path := range []string{"/src/missing", "/assets/missing", "/node_modules/missing.js"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, request("GET", path, ""))
		if w.Code != 404 || w.Header().Get("Content-Type") != "application/json" || strings.Contains(w.Body.String(), "<!doctype") {
			t.Fatalf("missing asset received SPA fallback: %s %d", path, w.Code)
		}
	}
}

func TestDevelopmentWebSocketAndShutdown(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/__vite_hmr" || r.URL.RawQuery != "token=dev" {
			t.Error("HMR path/query changed")
		}
		connection, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"vite-hmr"}, OriginPatterns: []string{"127.0.0.1:*"}})
		if err != nil {
			return
		}
		defer connection.CloseNow()
		connection.Write(r.Context(), websocket.MessageText, []byte(`{"type":"connected"}`))
		connection.Read(r.Context())
	}))
	defer upstream.Close()
	frontend, err := DevelopmentFrontend(upstream.URL, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	announced := make(announcedURL, 1)
	done := make(chan error, 1)
	go func() { done <- ServeDevelopment(ctx, fakeBackend{}, announced, io.Discard, frontend) }()
	var address *url.URL
	select {
	case value := <-announced:
		address, err = url.Parse(value)
	case err := <-done:
		t.Fatal(err)
	case <-time.After(3 * time.Second):
		t.Fatal("development server did not start")
	}
	if err != nil {
		t.Fatal(err)
	}
	clientCtx, stop := context.WithTimeout(context.Background(), 3*time.Second)
	defer stop()
	connection, _, err := websocket.Dial(clientCtx, "ws://"+address.Host+"/__vite_hmr?token=dev", &websocket.DialOptions{Subprotocols: []string{"vite-hmr"}, HTTPHeader: http.Header{"Origin": []string{"http://" + address.Host}}})
	if err != nil {
		t.Fatal(err)
	}
	defer connection.CloseNow()
	_, message, err := connection.Read(clientCtx)
	if err != nil || string(message) != `{"type":"connected"}` {
		t.Fatal("HMR handshake failed", err)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("HMR connection prevented shutdown")
	}
}

func TestDevelopmentEndpointValidation(t *testing.T) {
	for _, address := range []string{"", "http://localhost:5173", "http://127.0.0.1:0", "http://127.0.0.1:65536", "https://127.0.0.1:5173", "http://127.0.0.1:5173/path", "http://127.0.0.1:5173?", "http://private@127.0.0.1:5173"} {
		if _, err := DevelopmentFrontend(address, strings.Repeat("a", 64)); err == nil {
			t.Fatal("invalid frontend accepted", address)
		}
	}
	if _, err := DevelopmentFrontend("http://127.0.0.1:5173", "injected-style-policy"); err == nil {
		t.Fatal("invalid nonce accepted")
	}
}
