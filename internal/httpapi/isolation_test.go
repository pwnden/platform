package httpapi

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pwnden/platform/internal/application"
)

type isolatedBrowserBackend struct {
	*fakeBackend
	upstream string
	instance atomic.Int32
	calls    atomic.Int32
}

func TestHTTPRunAndStatusReuseObservationWhileProblemIsLocked(t *testing.T) {
	endpoints := []application.Endpoint{{Name: "web", URL: "http://app:8000", Proxied: true, Instance: "first"}}
	b := &isolatedBrowserBackend{fakeBackend: &fakeBackend{
		run: func(context.Context, string) (application.RunInfo, error) {
			return application.RunInfo{Slug: "example", Kind: application.KindService, Endpoints: endpoints}, nil
		},
	}}
	h := newHandler(context.Background(), b, testHost, testToken, io.Discard)
	defer h.browsers.close()
	defer func() {
		if err := h.workspaces.Close(); err != nil {
			t.Error(err)
		}
	}()
	for _, method := range []string{"POST", "GET"} {
		action := "run"
		if method == "GET" {
			action = "status"
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		r := request(method, BasePath+"/problems/example/"+action, "").WithContext(ctx)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		cancel()
		if w.Code != 200 || !strings.Contains(w.Body.String(), "http://127.0.0.1:") {
			t.Fatalf("%s: %d %s", action, w.Code, w.Body.String())
		}
	}
}

func (b *isolatedBrowserBackend) Status(context.Context, string) (application.RunStatus, error) {
	id := "first"
	if b.instance.Load() != 0 {
		id = "second"
	}
	return application.RunStatus{Slug: "example", Kind: application.KindService, State: "running",
		Endpoints: []application.Endpoint{{Name: "web", URL: "http://app:8000", Proxied: true, Instance: id}}}, nil
}
func (b *isolatedBrowserBackend) DialEndpoint(ctx context.Context, slug, name, instance string) (net.Conn, error) {
	if slug != "example" || name != "web" || instance != "first" {
		panic("unexpected endpoint identity")
	}
	b.calls.Add(1)
	return (&net.Dialer{}).DialContext(ctx, "tcp", strings.TrimPrefix(b.upstream, "http://"))
}

func TestIsolatedBrowserUsesDeclaredStreamAndRejectsRestart(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "value=%ff" {
			t.Error("query changed", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte{0, 13, 10, 255, 1})
	}))
	defer upstream.Close()
	b := &isolatedBrowserBackend{fakeBackend: &fakeBackend{}, upstream: upstream.URL}
	h := newHandler(context.Background(), b, testHost, testToken, io.Discard)
	defer h.browsers.close()
	status, _ := b.Status(context.Background(), "example")
	endpoints, err := h.exposeEndpoints(context.Background(), "example", status.Endpoints)
	if err != nil || len(endpoints) != 1 || !endpoints[0].Published || !strings.HasPrefix(endpoints[0].URL, "http://127.0.0.1:") {
		t.Fatalf("expose: %+v %v", endpoints, err)
	}
	browser, err := h.browsers.open(context.Background(), "example", "web")
	if err != nil || browser.Target != endpoints[0].URL {
		t.Fatalf("browser: %+v %v", browser, err)
	}
	response, err := http.Get(browser.Target + "/?value=%ff")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if string(data) != string([]byte{0, 13, 10, 255, 1}) || b.calls.Load() != 1 {
		t.Fatal("stream bytes changed", data, b.calls.Load())
	}
	b.instance.Store(1)
	response, err = http.Get(browser.Target + "/?value=%ff")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 410 || b.calls.Load() != 1 {
		t.Fatal("old origin reached restarted problem", response.StatusCode)
	}
}
