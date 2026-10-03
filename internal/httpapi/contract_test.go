package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pwnden/platform/internal/application"
	"github.com/pwnden/platform/internal/challenge"
	"github.com/pwnden/platform/internal/progress"
)

func TestCompletionResponseIncludesOnlyAcceptedPlayerState(t *testing.T) {
	solved := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	p := application.Problem{Slug: "test", SolvedAt: &solved}
	list, err := json.Marshal(ProblemsFrom([]application.Problem{p}))
	if err != nil || !strings.Contains(string(list), "solved_at") || strings.Contains(string(list), "answer") {
		t.Fatal(string(list), err)
	}
	detail, err := json.Marshal(DetailFrom(application.ProblemDetail{Problem: p, Completion: &progress.Completion{Slug: "test", Answer: "pwnden{saved}", SolvedAt: solved}}))
	if err != nil || !strings.Contains(string(detail), `"answer":"pwnden{saved}"`) {
		t.Fatal(string(detail), err)
	}
}

func TestPlayerResponsesDoNotExposeExecutionInternals(t *testing.T) {
	result := application.RunInfo{Slug: "example", Kind: application.KindService, Project: "private-project",
		Endpoints: []application.Endpoint{
			{Name: "web", URL: "http://127.0.0.1:1234", Published: true},
			{Name: "tcp", URL: "tcp://127.0.0.1:1235", Published: true},
			{Name: "shell", URL: "tcp://app:9000", Instance: "private-run-identity"},
			{Name: "database", URL: "http://172.20.0.2:5432", Published: false},
		}}
	data, err := json.Marshal(RunFrom(result))
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"slug":"example","kind":"service","file_count":0,"endpoints":[{"name":"web","url":"http://127.0.0.1:1234"},{"name":"tcp","url":"tcp://127.0.0.1:1235"},{"name":"shell","url":"tcp://app:9000"}]}`
	if string(data) != expected {
		t.Fatalf("unexpected public run response: %s", data)
	}
	data, _ = json.Marshal(ProblemsFrom(nil))
	if string(data) != `{"problems":[]}` {
		t.Fatalf("empty list: %s", data)
	}
	data, _ = json.Marshal(RunFrom(application.RunInfo{Slug: "example", Kind: application.KindFile, FileCount: 1}))
	if !strings.Contains(string(data), `"endpoints":[]`) {
		t.Fatalf("file endpoints: %s", data)
	}
}

func TestProblemResponsesIncludeDeclaredCLI(t *testing.T) {
	p := application.Problem{Slug: "diagnostic-port", Title: "개발용 점검 포트", Kind: application.KindService, CLI: []string{"nmap", "ncat"}}
	list := ProblemsFrom([]application.Problem{p})
	data, err := json.Marshal(list)
	if err != nil || !strings.Contains(string(data), `"cli":["nmap","ncat"]`) || strings.Contains(string(data), "search_text") {
		t.Fatal("declared CLI omitted", string(data), err)
	}
	data, err = json.Marshal(DetailFrom(application.ProblemDetail{Problem: p}))
	if err != nil || !strings.Contains(string(data), `"cli":["nmap","ncat"]`) {
		t.Fatal("detail CLI omitted", string(data), err)
	}
}

func TestLearningResponsesPreserveArrayShapeAndLegacyOmission(t *testing.T) {
	p := application.Problem{Slug: "example", Learning: &application.Learning{Requires: []challenge.Concept{{ID: "base", Title: "Starting ability"}}}}
	for _, response := range []any{ProblemsFrom([]application.Problem{p}), DetailFrom(application.ProblemDetail{Problem: p})} {
		data, err := json.Marshal(response)
		if err != nil || !strings.Contains(string(data), `"learning":`) || !strings.Contains(string(data), `"requires":[],"related":[]`) || !strings.Contains(string(data), `"teaches":[]`) {
			t.Fatalf("learning shape: %s %v", data, err)
		}
	}
	p.Learning = nil
	data, err := json.Marshal(ProblemsFrom([]application.Problem{p}))
	if err != nil || strings.Contains(string(data), "learning") {
		t.Fatalf("legacy metadata: %s %v", data, err)
	}
}

func TestPublicErrorClassificationAndPrivacy(t *testing.T) {
	for code, expected := range map[application.Code]int{
		application.InvalidArgument: http.StatusBadRequest, application.NotFound: http.StatusNotFound,
		application.AlreadyRunning: http.StatusConflict, application.NotRunning: http.StatusConflict,
		application.WorkspaceFull: http.StatusConflict, application.ResourceLimit: http.StatusConflict,
		application.IncompatibleContract: http.StatusUnprocessableEntity, application.ValidationFailed: http.StatusUnprocessableEntity,
		application.SetupRequired: http.StatusServiceUnavailable, application.SetupFailed: http.StatusServiceUnavailable,
		application.StorageFailed: http.StatusServiceUnavailable,
		application.Canceled:      http.StatusServiceUnavailable, application.DeadlineExceeded: http.StatusGatewayTimeout,
		application.ExecutionFailed: http.StatusInternalServerError, application.CleanupFailed: http.StatusInternalServerError,
	} {
		t.Run(string(code), func(t *testing.T) {
			private := errors.New("private host path /outside/secret; pwnden{private_flag}")
			failure := &application.Error{Code: code, Operation: "run", Slug: "example", Cause: private}
			status, response := ErrorFrom(fmt.Errorf("adapter: %w", failure))
			if status != expected || response.Error.Code != string(code) || response.Error.Operation != "run" || response.Error.Slug != "example" {
				t.Fatalf("classification: %d %+v", status, response)
			}
			data, err := json.Marshal(response)
			if err != nil || strings.Contains(string(data), "private") || strings.Contains(string(data), "secret") {
				t.Fatalf("private error exposed: %s %v", data, err)
			}
		})
	}
	for _, failure := range []error{errors.New("pwnden{private_flag}"), &application.Error{Code: "future_code"}} {
		status, response := ErrorFrom(failure)
		if status != 500 || response.Error.Code != "internal_error" || response.Error.Operation != "" {
			t.Fatalf("unknown error: %d %+v", status, response)
		}
	}
}
