package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pwnden/platform/internal/application"
)

func browserBackend(target string) fakeBackend {
	return fakeBackend{status: func(context.Context, string) (application.RunStatus, error) {
		return application.RunStatus{Kind: application.KindService, State: "running", Endpoints: []application.Endpoint{{Name: "web", URL: target, Published: true}}}, nil
	}}
}

func TestBrowserPreservesProblemHTTP(t *testing.T) {
	var target string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("platform credentials reached problem")
		}
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		switch r.URL.Path {
		case "/login":
			if r.Method != "POST" || r.Header.Get("Origin") != "http://"+r.Host {
				t.Error("form origin/method changed")
			}
			if err := r.ParseForm(); err != nil || r.Form.Get("name") != "guest" {
				t.Error("form changed")
			}
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "guest", Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
			http.Redirect(w, r, target+"/notes?id=2", 303)
		case "/notes":
			cookie, err := r.Cookie("session")
			if err != nil || cookie.Value != "guest" || r.URL.RawQuery != "id=2" {
				t.Error("login state/query lost")
			}
			_, _ = io.WriteString(w, "unchanged problem content")
		case "/encoded/path":
			if r.URL.RawPath != "/encoded%2Fpath" || r.URL.RawQuery != "q=%2B" {
				t.Error("encoded request changed", r.URL)
			}
			_, _ = io.WriteString(w, "encoded")
		case "/unusual":
			if r.URL.RawQuery != "q=a;b&x=%2B" || r.Header.Get("X-Forwarded-For") != "challenge-input" {
				t.Error("exercise request normalized", r.URL, r.Header)
			}
			_, _ = io.WriteString(w, "unusual")
		default:
			_, _ = io.WriteString(w, "<html><script src='/app.js'></script><form method='post' action='/login'></form></html>")
		}
	}))
	defer upstream.Close()
	target = upstream.URL
	manager := newBrowserManager(context.Background(), browserBackend(target), "http://"+testHost, nil)
	defer manager.close()
	first, err := manager.open(context.Background(), "example", "web")
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := manager.open(context.Background(), "example", "web")
	if err != nil || first != repeated || first.Target != target {
		t.Fatal("browser not reused", first, repeated, err)
	}
	wrapperURL, _ := url.Parse(first.URL)
	origin := "http://" + wrapperURL.Host
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 3 * time.Second}
	read := func(response *http.Response, err error) string {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		content, err := io.ReadAll(response.Body)
		if err != nil || response.StatusCode != 200 {
			t.Fatal(response.Status, err, string(content))
		}
		return string(content)
	}
	wrapper := read(client.Get(first.URL))
	if !strings.Contains(wrapper, "data-parent=\"http://"+testHost+"\"") || strings.Contains(wrapper, testToken) {
		t.Fatal("wrapper leaked platform context", wrapper)
	}
	source := read(client.Get(origin + "/"))
	if strings.Contains(source, "pwnden") || !strings.Contains(source, "<form method='post'") {
		t.Fatal("target document modified", source)
	}
	request, _ := http.NewRequest("POST", origin+"/login", strings.NewReader("name=guest"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", origin)
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.Header.Get("Content-Security-Policy") != "default-src 'self'" || response.Request.URL.Host != wrapperURL.Host {
		t.Fatal("headers/redirect origin changed", response.Header, response.Request.URL)
	}
	if content := read(response, nil); content != "unchanged problem content" {
		t.Fatal(content)
	}
	read(client.Get(origin + "/encoded%2Fpath?q=%2B"))
	unusual, _ := http.NewRequest("GET", origin+"/unusual?q=a;b&x=%2B", nil)
	unusual.Header.Set("X-Forwarded-For", "challenge-input")
	read(client.Do(unusual))
	bad, _ := http.NewRequest("GET", first.URL, nil)
	bad.Host = "attacker.example"
	response, err = client.Do(bad)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 403 {
		t.Fatal("host rebinding accepted", response.Status)
	}
	manager.close()
	if response, err := client.Get(first.URL); err == nil {
		response.Body.Close()
		t.Fatal("listener survived shutdown")
	}
}

func TestBrowserRejectsUnknownOrStaleTargets(t *testing.T) {
	for _, target := range []string{"http://example.com:80", "http://127.0.0.1:0", "https://127.0.0.1:9000", "http://127.0.0.1:9000/path", "http://user@127.0.0.1:9000", "http://127.0.0.1:9000?x=y", "http://" + testHost} {
		manager := newBrowserManager(context.Background(), browserBackend(target), "http://"+testHost, nil)
		_, err := manager.open(context.Background(), "example", "web")
		manager.close()
		if err == nil {
			t.Fatal("invalid target accepted", target)
		}
	}
	var stopped atomic.Bool
	backend := browserBackend("http://127.0.0.1:9000")
	previous := backend.status
	backend.status = func(ctx context.Context, slug string) (application.RunStatus, error) {
		status, err := previous(ctx, slug)
		if stopped.Load() {
			status.State = "stopped"
		}
		return status, err
	}
	manager := newBrowserManager(context.Background(), backend, "http://"+testHost, nil)
	defer manager.close()
	if _, err := manager.open(context.Background(), "example", "private"); err == nil {
		t.Fatal("undeclared endpoint accepted")
	}
	session, err := manager.open(context.Background(), "example", "web")
	if err != nil {
		t.Fatal(err)
	}
	stopped.Store(true)
	u, _ := url.Parse(session.URL)
	response, err := http.Get("http://" + u.Host + "/secret")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 410 {
		t.Fatal("stopped environment accessed", response.Status)
	}
}

func TestBrowserAPIAndRetention(t *testing.T) {
	h := newHandler(context.Background(), browserBackend("http://127.0.0.1:9000"), testHost, testToken, io.Discard)
	defer h.browsers.close()
	path := BasePath + "/problems/example/browser"
	for _, body := range []string{`{"name":"web","target":"http://evil"}`, `{"name":"web","name":"other"}`, `{"name":""}`, `{"flag":"web"}`} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, request("POST", path, body))
		if w.Code != 400 {
			t.Fatal("malformed browser request accepted", body, w.Code)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, request("POST", path, `{"name":"web"}`))
	var session Browser
	if err := json.Unmarshal(w.Body.Bytes(), &session); err != nil || w.Code != 200 || session.Slug != "example" || session.Name != "web" {
		t.Fatal(w.Code, w.Body.String(), err)
	}
	// No retained workspace owns this endpoint, so the common sweep closes it.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		h.browsers.mu.Lock()
		count := len(h.browsers.entries)
		h.browsers.mu.Unlock()
		if count == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("unowned browser listener retained")
}

func TestBrowserClosesUpgradedConnections(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, buffered, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		_, _ = buffered.WriteString("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\n")
		_ = buffered.Flush()
		_, _ = io.Copy(conn, conn)
	}))
	defer upstream.Close()
	manager := newBrowserManager(context.Background(), browserBackend(upstream.URL), "http://"+testHost, nil)
	session, err := manager.open(context.Background(), "example", "web")
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(session.URL)
	conn, err := net.DialTimeout("tcp", u.Host, time.Second)
	if err != nil {
		manager.close()
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	_, _ = io.WriteString(conn, "GET /socket HTTP/1.1\r\nHost: "+u.Host+"\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\n")
	response, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil || response.StatusCode != 101 {
		manager.close()
		t.Fatal("upgrade failed", err)
	}
	done := make(chan struct{})
	go func() { manager.close(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("upgraded connection blocked shutdown")
	}
	if _, err := conn.Read(make([]byte, 1)); err == nil {
		t.Fatal("upgraded socket retained")
	}
}
