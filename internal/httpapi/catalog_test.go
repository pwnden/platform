package httpapi

import (
	"context"
	"io"
	"mime"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pwnden/platform/internal/application"
)

type downloaded struct {
	*strings.Reader
	closed bool
}

func (d *downloaded) Close() error { d.closed = true; return nil }

func TestDetailDownloadAndStatus(t *testing.T) {
	id := strings.Repeat("a", 64)
	stream := &downloaded{Reader: strings.NewReader("file content")}
	b := fakeBackend{
		guidance: func(_ context.Context, slug, id string) (string, error) {
			if slug != "example" || id != "hint-1" {
				t.Fatal(slug, id)
			}
			return "Read the supplied material.", nil
		},
		detail: func(_ context.Context, slug string) (application.ProblemDetail, error) {
			return application.ProblemDetail{Problem: application.Problem{Slug: slug, Title: "Example", Kind: application.KindFile}, Description: "<script>example</script>", Files: []application.ProblemFile{{ID: id, Name: "files/파일.txt", Size: 12}}}, nil
		},
		download: func(_ context.Context, slug, fileID string) (application.Download, error) {
			if slug != "example" || fileID != id {
				t.Fatal(slug, fileID)
			}
			return application.Download{Name: "파일.txt", Size: 12, Content: stream}, nil
		},
		status: func(_ context.Context, slug string) (application.RunStatus, error) {
			return application.RunStatus{Slug: slug, Kind: application.KindService, State: "running", Endpoints: []application.Endpoint{{Name: "web", URL: "http://127.0.0.1:1234", Published: true}, {Name: "private", URL: "http://database"}}}, nil
		},
	}
	h := newHandler(context.Background(), b, testHost, testToken, io.Discard)
	for _, path := range []string{"/example", "/example/status", "/example/files/" + id, "/example/guidance/hint-1"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, request("GET", BasePath+"/problems"+path, ""))
		if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || strings.Contains(w.Body.String(), "database") {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
		if strings.Contains(path, "/files/") {
			media, parameters, err := mime.ParseMediaType(w.Header().Get("Content-Disposition"))
			if err != nil || media != "attachment" || parameters["filename"] != "파일.txt" || w.Body.String() != "file content" || !stream.closed || w.Header().Get("Content-Type") != "application/octet-stream" {
				t.Fatal("binary attachment contract", w.Header(), stream.closed, err)
			}
		}
	}
	for _, path := range []string{"/example", "/example/status", "/example/files/" + id, "/example/guidance/hint-1"} {
		for _, test := range []struct {
			method, content, token string
			status                 int
		}{
			{"GET", "", "", 401}, {"POST", "", testToken, 405}, {"GET", "x", testToken, 400},
		} {
			w := httptest.NewRecorder()
			r := request(test.method, BasePath+"/problems"+path, test.content)
			r.Header.Set("Authorization", "Bearer "+test.token)
			h.ServeHTTP(w, r)
			if w.Code != test.status {
				t.Fatalf("%s %s: %d", test.method, path, w.Code)
			}
		}
	}
	for _, path := range []string{"/example/files/README.md", "/example/files/../solve", "/example/files/" + strings.Repeat("A", 64), "/example/guidance/solve.py", "/example/guidance/hint-11", "/example/guidance/hint-01"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, request("GET", BasePath+"/problems"+path, ""))
		assertError(t, w, 404, "not_found")
	}
}

func TestStatusWaitsForProblemMutationAndNeverStops(t *testing.T) {
	called := make(chan struct{}, 1)
	h := newHandler(context.Background(), fakeBackend{status: func(context.Context, string) (application.RunStatus, error) {
		called <- struct{}{}
		return application.RunStatus{Slug: "example", Kind: application.KindService, State: "running"}, nil
	}}, testHost, testToken, io.Discard)
	release, err := h.acquire(context.Background(), "example")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithCancel(context.Background())
	r := request("GET", BasePath+"/problems/example/status", "").WithContext(ctx)
	w := httptest.NewRecorder()
	done := make(chan struct{})
	go func() { h.ServeHTTP(w, r); close(done) }()
	select {
	case <-called:
		t.Fatal("status raced the mutation")
	case <-time.After(20 * time.Millisecond):
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("waiting status did not cancel")
	}
	assertError(t, w, 503, "canceled")
	select {
	case <-called:
		t.Fatal("canceled observer reached backend")
	default:
	}
}
