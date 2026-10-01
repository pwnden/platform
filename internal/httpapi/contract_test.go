package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/pwnden/platform/internal/application"
)

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

func TestPublicErrorClassificationAndPrivacy(t *testing.T) {
	for code, expected := range map[application.Code]int{
		application.InvalidArgument: http.StatusBadRequest, application.NotFound: http.StatusNotFound,
		application.AlreadyRunning: http.StatusConflict, application.NotRunning: http.StatusConflict,
		application.IncompatibleContract: http.StatusUnprocessableEntity, application.ValidationFailed: http.StatusUnprocessableEntity,
		application.SetupRequired: http.StatusServiceUnavailable, application.SetupFailed: http.StatusServiceUnavailable,
		application.Canceled: http.StatusServiceUnavailable, application.DeadlineExceeded: http.StatusGatewayTimeout,
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
