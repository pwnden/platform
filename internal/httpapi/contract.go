// Package httpapi adapts the application capabilities to the local player API.
package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/pwnden/platform/internal/application"
)

const BasePath = "/api/v1"

// Backend is the capability set needed by the initial player HTTP API.
type Backend interface {
	application.Catalog
	application.Details
	application.Guidance
	application.Files
	application.Observer
	application.Terminals
	application.Runner
	Submit(context.Context, string, string) (application.Submission, error)
}

type Problem struct {
	Slug     string           `json:"slug"`
	Title    string           `json:"title"`
	Category string           `json:"category"`
	Kind     application.Kind `json:"kind"`
}

type ProblemList struct {
	Problems []Problem `json:"problems"`
}

type ProblemFile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type ProblemDetail struct {
	Problem
	Description string        `json:"description"`
	Files       []ProblemFile `json:"files"`
	HintCount   int           `json:"hint_count"`
	Walkthrough bool          `json:"walkthrough"`
}

type Guidance struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

type Status struct {
	Slug      string           `json:"slug"`
	Kind      application.Kind `json:"kind"`
	State     string           `json:"state"`
	Endpoints []Endpoint       `json:"endpoints"`
}

type Endpoint struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Run struct {
	Slug      string           `json:"slug"`
	Kind      application.Kind `json:"kind"`
	FileCount int              `json:"file_count"`
	Endpoints []Endpoint       `json:"endpoints"`
}

type Stop struct {
	Slug string `json:"slug"`
}

type SubmissionRequest struct {
	Flag string `json:"flag"`
}

type Submission struct {
	Slug     string `json:"slug"`
	Accepted bool   `json:"accepted"`
}

type Error struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Operation string `json:"operation,omitempty"`
	Slug      string `json:"slug,omitempty"`
}

type ErrorResponse struct {
	Error Error `json:"error"`
}

func ProblemsFrom(result []application.Problem) ProblemList {
	response := ProblemList{Problems: make([]Problem, 0, len(result))}
	for _, p := range result {
		response.Problems = append(response.Problems, Problem{p.Slug, p.Title, p.Category, p.Kind})
	}
	return response
}

func DetailFrom(result application.ProblemDetail) ProblemDetail {
	response := ProblemDetail{Problem: Problem{result.Slug, result.Title, result.Category, result.Kind}, Description: result.Description, Files: make([]ProblemFile, 0, len(result.Files))}
	response.HintCount, response.Walkthrough = result.HintCount, result.Walkthrough
	for _, file := range result.Files {
		response.Files = append(response.Files, ProblemFile{file.ID, file.Name, file.Size})
	}
	return response
}

func StatusFrom(result application.RunStatus) Status {
	run := RunFrom(application.RunInfo{Endpoints: result.Endpoints})
	return Status{Slug: result.Slug, Kind: result.Kind, State: result.State, Endpoints: run.Endpoints}
}

func RunFrom(result application.RunInfo) Run {
	response := Run{Slug: result.Slug, Kind: result.Kind, FileCount: result.FileCount, Endpoints: make([]Endpoint, 0)}
	for _, endpoint := range result.Endpoints {
		if endpoint.Published || (endpoint.Instance != "" && strings.HasPrefix(endpoint.URL, "tcp://")) {
			response.Endpoints = append(response.Endpoints, Endpoint{endpoint.Name, endpoint.URL})
		}
	}
	return response
}

// ErrorFrom keeps process output, host paths, flags and underlying errors private.
func ErrorFrom(err error) (int, ErrorResponse) {
	response := ErrorResponse{Error: Error{Code: "internal_error", Message: "The operation failed."}}
	var failure *application.Error
	if !errors.As(err, &failure) {
		return http.StatusInternalServerError, response
	}
	status := http.StatusInternalServerError
	message := ""
	switch failure.Code {
	case application.WorkspaceFull:
		status, message = http.StatusConflict, "End a retained environment before opening another."
	case application.TerminalBusy:
		status, message = http.StatusConflict, "The terminal is attached in another tab."
	case application.WorkspaceBusy:
		status, message = http.StatusConflict, "The problem installation is in use."
	case application.InvalidArgument:
		status, message = http.StatusBadRequest, "The problem input is invalid."
	case application.NotFound:
		status, message = http.StatusNotFound, "The problem or a required input was not found."
	case application.AlreadyRunning:
		status, message = http.StatusConflict, "The problem is already running."
	case application.NotRunning:
		status, message = http.StatusConflict, "Start the problem first."
	case application.IncompatibleContract:
		status, message = http.StatusUnprocessableEntity, "The problem contract is unsupported."
	case application.ValidationFailed:
		status, message = http.StatusUnprocessableEntity, "The problem did not pass execution checks."
	case application.SetupRequired:
		status, message = http.StatusServiceUnavailable, "Run setup first."
	case application.SetupFailed:
		status, message = http.StatusServiceUnavailable, "The problem installation is unavailable."
	case application.Canceled:
		status, message = http.StatusServiceUnavailable, "The operation was canceled."
	case application.DeadlineExceeded:
		status, message = http.StatusGatewayTimeout, "The operation timed out."
	case application.ExecutionFailed:
		message = "The problem operation failed."
	case application.CleanupFailed:
		message = "Cleanup failed; the problem may still be running."
	default:
		return status, response
	}
	response.Error = Error{Code: string(failure.Code), Message: message, Operation: failure.Operation, Slug: failure.Slug}
	return status, response
}

var _ Backend = (*application.Service)(nil)
