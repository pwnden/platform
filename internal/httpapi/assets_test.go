package httpapi

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestEmbeddedPlayerAssets(t *testing.T) {
	h := newHandler(context.Background(), fakeBackend{}, testHost, testToken, io.Discard)
	get := func(method, path string) *httptest.ResponseRecorder {
		r := request(method, path, "")
		r.Header.Del("Authorization")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	page := get("GET", "/")
	if page.Code != 200 || !strings.Contains(page.Body.String(), `type="module"`) || strings.Contains(page.Body.String(), "session.js") {
		t.Fatal("the built player entry point was not served")
	}
	assets := regexp.MustCompile(`(?:src|href)="(/assets/[^" ]+)"`).FindAllStringSubmatch(page.Body.String(), -1)
	nonce := regexp.MustCompile(`name="pwnden-style-nonce" content="([a-f0-9]{64})"`).FindStringSubmatch(page.Body.String())
	if len(nonce) != 2 || !strings.Contains(page.Header().Get("Content-Security-Policy"), "style-src 'self' 'nonce-"+nonce[1]+"'") || strings.Contains(page.Header().Get("Content-Security-Policy"), "unsafe-inline") {
		t.Fatal("terminal style nonce policy is missing or broad")
	}
	if !strings.Contains(page.Header().Get("Content-Security-Policy"), "connect-src 'self' ws://"+testHost+";") {
		t.Fatal("terminal WebSocket policy is not confined to the server")
	}
	if policy := page.Header().Get("Content-Security-Policy"); !strings.Contains(policy, "script-src 'self' 'wasm-unsafe-eval';") || strings.Contains(policy, "'unsafe-eval'") {
		t.Fatal("markdown WASM policy is missing or permits JavaScript eval")
	}
	if policy := page.Header().Get("Content-Security-Policy"); !strings.Contains(policy, "frame-src http://127.0.0.1:*;") || !strings.Contains(policy, "frame-ancestors 'none';") {
		t.Fatal("problem frames must be confined to loopback while the platform stays unframed")
	}
	if next := get("GET", "/"); strings.Contains(next.Body.String(), nonce[1]) {
		t.Fatal("style nonce reused")
	}
	for _, path := range []string{"/challenges", "/challenges/", "/challenges/note-vault", "/challenges/note-vault?q=nmap&page=2", "/?q=nmap", "/settings", "/courses/basics/lesson-1?tab=notes", "/unknown", "/challenges/x/extra"} {
		w := get("GET", path)
		if w.Code != 200 || w.Header().Get("Content-Type") != "text/html; charset=utf-8" || !strings.Contains(w.Body.String(), `type="module"`) || !strings.Contains(w.Header().Get("Content-Security-Policy"), "nonce-") {
			t.Fatalf("player route did not serve protected HTML: %s HTTP %d", path, w.Code)
		}
		if head := get("HEAD", path); head.Code != 200 || head.Body.Len() != 0 {
			t.Fatalf("player route HEAD failed: %s", path)
		}
		if post := get("POST", path); post.Code != 405 {
			t.Fatalf("player route allowed mutation: %s", path)
		}
	}
	for _, path := range []string{"/challenges/../go.mod", "/assets/missing.js", "/missing.css", "/favicon.ico", "/.git/config", "/src/missing", "/@fs/etc/passwd", "/__open-in-editor", "/api", "/assets", "/node_modules", "/courses//lesson"} {
		if w := get("GET", path); w.Code != 404 || strings.Contains(w.Body.String(), `type="module"`) {
			t.Fatalf("non-page path received player HTML: %s HTTP %d", path, w.Code)
		}
	}
	if w := get("GET", "/api/v1/missing"); w.Code != 401 || strings.Contains(w.Body.String(), `type="module"`) {
		t.Fatal("unknown API bypassed authentication or received player HTML")
	}
	for _, path := range []string{"/api/v1/problems?q=x", "/assets/missing.js?q=x", "/challenges/%6eote-vault"} {
		if w := get("GET", path); w.Code != 400 {
			t.Fatalf("restricted path accepted query or encoding: %s HTTP %d", path, w.Code)
		}
	}
	if len(assets) < 2 {
		t.Fatal("missing script or style references")
	}
	for _, asset := range assets {
		path := asset[1]
		w := get("GET", path)
		content := w.Body.String()
		if w.Code != 200 || content == "" || strings.Contains(content, testToken) || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("invalid or private asset: %s", path)
		}
		expected := "text/javascript; charset=utf-8"
		if strings.HasSuffix(path, ".css") {
			expected = "text/css; charset=utf-8"
		} else if strings.HasSuffix(path, ".woff2") {
			expected = "font/woff2"
			if !strings.HasPrefix(content, "wOF2") {
				t.Fatalf("invalid preloaded font: %s", path)
			}
		}
		if w.Header().Get("Content-Type") != expected || w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("asset MIME mismatch: %s", path)
		}
		if strings.HasSuffix(path, ".css") {
			fonts := regexp.MustCompile(`url\((/assets/[^)]+\.woff2)\)`).FindAllStringSubmatch(content, -1)
			if len(fonts) != 4 {
				t.Fatal("the player must bundle its Pretendard body and Latin/Korean monospace fonts")
			}
			for _, font := range fonts {
				w := get("GET", font[1])
				if w.Code != 200 || w.Header().Get("Content-Type") != "font/woff2" || !strings.HasPrefix(w.Body.String(), "wOF2") {
					t.Fatalf("invalid bundled font: %s", font[1])
				}
			}
		}
		head := get("HEAD", path)
		if head.Code != 200 || head.Body.Len() != 0 || head.Header().Get("Content-Length") != fmt.Sprint(len(content)) {
			t.Fatal("invalid HEAD response")
		}
	}
	for _, name := range []string{"JetBrainsMono", "D2Coding", "Pretendard"} {
		w := get("GET", "/assets/licenses/"+name+"-OFL.txt")
		if w.Code != 200 || !strings.Contains(w.Body.String(), "SIL OPEN FONT LICENSE") {
			t.Fatalf("missing distributed font license: %s", name)
		}
	}
	for _, path := range []string{"/assets/", "/assets/missing.js", "/assets/../index.html", "/assets/../../go.mod", "/assets/index.js.map"} {
		if w := get("GET", path); w.Code != 404 {
			t.Fatalf("unexpected public asset: %s HTTP %d", path, w.Code)
		}
	}
	if w := get("POST", assets[0][1]); w.Code != 405 {
		t.Fatal("asset accepted a mutation")
	}
	for _, change := range []func(*httptest.ResponseRecorder){
		func(w *httptest.ResponseRecorder) {
			r := request("GET", assets[0][1], "")
			r.Host = "other.invalid"
			h.ServeHTTP(w, r)
		},
		func(w *httptest.ResponseRecorder) {
			r := request("GET", assets[0][1], "")
			r.Header.Set("Origin", "https://other.invalid")
			h.ServeHTTP(w, r)
		},
	} {
		w := httptest.NewRecorder()
		change(w)
		if w.Code != 403 {
			t.Fatal("web assets bypassed origin checks")
		}
	}
}
