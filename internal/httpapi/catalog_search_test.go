package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/pwnden/platform/internal/application"
)

func TestPublishedCatalogSearch(t *testing.T) {
	repo := os.Getenv("PWNDEN_TEST_CHALLENGES")
	if repo == "" {
		t.Skip("set PWNDEN_TEST_CHALLENGES to check tool search against the published catalog")
	}
	backend, err := application.New(repo)
	if err != nil {
		t.Fatal(err)
	}
	h := newHandler(context.Background(), backend, testHost, testToken, io.Discard)
	defer h.browsers.close()
	defer h.workspaces.Close()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, request("GET", BasePath+"/problems", ""))
	var catalog ProblemList
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &catalog) != nil {
		t.Fatalf("catalog: %d %s", w.Code, w.Body.String())
	}
	for _, tool := range []string{"nmap", "ffuf", "sqlite3", "exiftool", "tshark", "jq", "git", "openssl"} {
		var found []string
		for _, problem := range catalog.Problems {
			if strings.Contains(strings.ToLower(problem.SearchText), tool) {
				found = append(found, problem.Slug)
			}
		}
		if len(found) == 0 || (tool == "nmap" && (len(found) != 1 || found[0] != "diagnostic-port")) {
			t.Fatalf("tool %q not indexed correctly: %v", tool, found)
		}
		t.Logf("%s: %v", tool, found)
	}
}
