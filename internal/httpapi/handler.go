package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/pwnden/platform/internal/application"
	"github.com/pwnden/platform/internal/playerweb"
)

type problemLock struct {
	gate  chan struct{}
	users int
}

type handler struct {
	base            context.Context
	backend         Backend
	host, token     string
	diagnostics     io.Writer
	mu              sync.Mutex
	closing         bool
	active          sync.WaitGroup
	locks           map[string]*problemLock
	cleanupFailures error
	terminals       map[string]*terminalConnection
}

func newHandler(base context.Context, backend Backend, host, token string, diagnostics io.Writer) *handler {
	return &handler{base: base, backend: backend, host: host, token: token, diagnostics: diagnostics, locks: make(map[string]*problemLock), terminals: make(map[string]*terminalConnection)}
}

func (h *handler) closeAdmission() {
	h.mu.Lock()
	h.closing = true
	h.mu.Unlock()
}

func (h *handler) cleanupError() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.cleanupFailures
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
	h.mu.Lock()
	if h.closing {
		h.mu.Unlock()
		transportError(w, 503, "canceled", "The server is stopping.")
		return
	}
	h.active.Add(1)
	h.mu.Unlock()
	defer h.active.Done()
	if r.Host != h.host || len(r.Header.Values("Origin")) > 1 ||
		(r.Header.Get("Origin") != "" && r.Header.Get("Origin") != "http://"+h.host) {
		transportError(w, 403, "forbidden", "The request origin is not allowed.")
		return
	}
	site := r.Header.Get("Sec-Fetch-Site")
	if site != "" && site != "none" && site != "same-origin" {
		transportError(w, 403, "forbidden", "The request origin is not allowed.")
		return
	}
	if r.URL.RawQuery != "" || r.URL.ForceQuery || r.URL.RawPath != "" {
		transportError(w, 400, "invalid_request", "Query parameters and encoded paths are not accepted.")
		return
	}
	if r.URL.Path == "/" || strings.HasPrefix(r.URL.Path, "/assets/") {
		h.asset(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, BasePath+"/problems/") && strings.HasSuffix(r.URL.Path, "/terminal") {
		slug := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, BasePath+"/problems/"), "/terminal")
		if len(slug) > 40 || !apiSlug.MatchString(slug) {
			transportError(w, 400, "invalid_argument", "The problem slug is invalid.")
			return
		}
		h.terminal(w, r, slug)
		return
	}
	if len(r.Header.Values("Authorization")) != 1 || subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+h.token)) != 1 {
		w.Header().Set("WWW-Authenticate", "Bearer")
		transportError(w, 401, "unauthorized", "A server session token is required.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), operationTimeout)
	defer cancel()
	stop := context.AfterFunc(h.base, cancel)
	defer stop()
	if h.base.Err() != nil {
		cancel()
	}
	if r.URL.Path == BasePath+"/problems" {
		if !method(w, r, "GET") || !emptyBody(w, r) {
			return
		}
		result, err := h.backend.List(ctx)
		if err != nil {
			applicationError(w, err)
			return
		}
		writeJSON(w, 200, ProblemsFrom(result))
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, BasePath+"/problems/"), "/")
	action := ""
	switch {
	case len(parts) == 1:
		action = "detail"
	case len(parts) == 2 && (parts[1] == "run" || parts[1] == "submissions" || parts[1] == "status"):
		action = parts[1]
	case len(parts) == 3 && parts[1] == "files" && fileID.MatchString(parts[2]):
		action = "download"
	}
	if !strings.HasPrefix(r.URL.Path, BasePath+"/problems/") || action == "" {
		transportError(w, 404, "not_found", "The API route was not found.")
		return
	}
	slug := parts[0]
	if len(slug) > 40 || !apiSlug.MatchString(slug) {
		transportError(w, 400, "invalid_argument", "The problem slug is invalid.")
		return
	}
	if action == "detail" || action == "download" {
		if !method(w, r, "GET") || !emptyBody(w, r) {
			return
		}
		if action == "detail" {
			result, err := h.backend.Detail(ctx, slug)
			if err != nil {
				applicationError(w, err)
				return
			}
			writeJSON(w, 200, DetailFrom(result))
		} else {
			result, err := h.backend.Download(ctx, slug, parts[2])
			if err != nil {
				applicationError(w, err)
				return
			}
			defer result.Content.Close()
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": result.Name}))
			w.Header().Set("Content-Length", fmt.Sprint(result.Size))
			w.WriteHeader(200)
			io.CopyN(w, result.Content, result.Size)
		}
		return
	}
	var flag string
	if action == "status" {
		if !method(w, r, "GET") || !emptyBody(w, r) {
			return
		}
	} else if action == "run" {
		if !method(w, r, "POST", "DELETE") || !emptyBody(w, r) {
			return
		}
	} else {
		if !method(w, r, "POST") {
			return
		}
		var ok bool
		flag, ok = submissionBody(w, r)
		if !ok {
			return
		}
	}
	release, err := h.acquire(ctx, slug)
	if err != nil {
		applicationError(w, contextError(err))
		return
	}
	defer release()
	switch {
	case action == "status":
		result, err := h.backend.Status(ctx, slug)
		if err != nil {
			applicationError(w, err)
			return
		}
		writeJSON(w, 200, StatusFrom(result))
	case action == "submissions":
		result, err := h.backend.Submit(ctx, slug, flag)
		if err != nil {
			applicationError(w, err)
			return
		}
		writeJSON(w, 200, Submission{Slug: result.Slug, Accepted: result.Accepted})
	case r.Method == "DELETE":
		if err := h.stopTerminal(slug); err != nil {
			applicationError(w, err)
			return
		}
		result, err := h.backend.Stop(ctx, slug)
		if err != nil {
			applicationError(w, err)
			return
		}
		writeJSON(w, 200, Stop{Slug: result.Slug})
	default:
		result, err := h.backend.Run(ctx, slug)
		if err != nil {
			applicationError(w, err)
			return
		}
		if ctx.Err() != nil {
			failure := contextError(ctx.Err())
			if result.Kind == application.KindService {
				if cleanupErr := h.stopUndelivered(ctx, slug); cleanupErr != nil {
					failure = cleanupErr
				}
			}
			applicationError(w, failure)
			return
		}
		if err := writeJSON(w, 200, RunFrom(result)); err != nil && result.Kind == application.KindService {
			h.stopUndelivered(ctx, slug)
		}
	}
}

func (h *handler) stopUndelivered(ctx context.Context, slug string) error {
	_, err := h.backend.Stop(context.WithoutCancel(ctx), slug)
	if err != nil {
		h.mu.Lock()
		h.cleanupFailures = errors.Join(h.cleanupFailures, err)
		// Log only the code and validated slug, never the underlying cause.
		fmt.Fprintf(h.diagnostics, "pwnden: cleanup_failed for %s\n", slug)
		h.mu.Unlock()
	}
	return err
}

var apiSlug = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var fileID = regexp.MustCompile(`^[a-f0-9]{64}$`)

func contextError(err error) error {
	code := application.Canceled
	if errors.Is(err, context.DeadlineExceeded) {
		code = application.DeadlineExceeded
	}
	return &application.Error{Code: code, Cause: err}
}

func (h *handler) acquire(ctx context.Context, slug string) (func(), error) {
	h.mu.Lock()
	lock := h.locks[slug]
	if lock == nil {
		lock = &problemLock{gate: make(chan struct{}, 1)}
		h.locks[slug] = lock
	}
	lock.users++
	h.mu.Unlock()
	forget := func() {
		h.mu.Lock()
		lock.users--
		if lock.users == 0 {
			delete(h.locks, slug)
		}
		h.mu.Unlock()
	}
	select {
	case lock.gate <- struct{}{}:
		if err := ctx.Err(); err != nil {
			<-lock.gate
			forget()
			return nil, err
		}
		return func() { <-lock.gate; forget() }, nil
	case <-ctx.Done():
		forget()
		return nil, ctx.Err()
	}
}

func method(w http.ResponseWriter, r *http.Request, allowed ...string) bool {
	for _, value := range allowed {
		if r.Method == value {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	transportError(w, 405, "method_not_allowed", "The request method is not allowed.")
	return false
}

func body(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	content, err := io.ReadAll(r.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			transportError(w, 413, "payload_too_large", "The request body exceeds 4096 bytes.")
		} else {
			transportError(w, 400, "invalid_request", "The request body could not be read.")
		}
		return nil, false
	}
	return content, true
}

func emptyBody(w http.ResponseWriter, r *http.Request) bool {
	content, ok := body(w, r)
	if ok && len(content) != 0 {
		transportError(w, 400, "invalid_request", "This operation accepts no request body.")
		return false
	}
	return ok
}

func submissionBody(w http.ResponseWriter, r *http.Request) (string, bool) {
	media, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if len(r.Header.Values("Content-Type")) != 1 || err != nil || media != "application/json" || len(params) > 1 || (len(params) == 1 && params["charset"] == "") {
		transportError(w, 415, "unsupported_media_type", "Use application/json with an optional charset.")
		return "", false
	}
	content, ok := body(w, r)
	if !ok {
		return "", false
	}
	// Token parsing also rejects duplicate keys, case variants, and extra values.
	d := json.NewDecoder(strings.NewReader(string(content)))
	open, err := d.Token()
	key, keyErr := d.Token()
	var flag string
	valueErr := d.Decode(&flag)
	close, closeErr := d.Token()
	_, endErr := d.Token()
	if !utf8.Valid(content) || err != nil || open != json.Delim('{') || keyErr != nil || key != "flag" || valueErr != nil || closeErr != nil || close != json.Delim('}') || endErr != io.EOF || strings.TrimSpace(flag) == "" {
		transportError(w, 400, "invalid_request", "Submit one JSON object containing a nonempty flag string.")
		return "", false
	}
	return flag, true
}

func applicationError(w http.ResponseWriter, err error) {
	status, response := ErrorFrom(err)
	writeJSON(w, status, response)
}

func transportError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{Error: Error{Code: code, Message: message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) error {
	content, err := json.Marshal(value)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(append(content, '\n')); err != nil {
		return err
	}
	// Observe errors hidden by net/http's small-response buffering while the
	// caller still owns any new run. This cannot acknowledge application receipt.
	err = http.NewResponseController(w).Flush()
	if errors.Is(err, http.ErrNotSupported) {
		return nil
	}
	return err
}

func (h *handler) asset(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET", "HEAD") || !emptyBody(w, r) {
		return
	}
	content, err := playerweb.Read(r.URL.Path)
	if err != nil {
		transportError(w, 404, "not_found", "The web asset was not found.")
		return
	}
	typeName := "text/html; charset=utf-8"
	if r.URL.Path != "/" {
		switch {
		case strings.HasSuffix(r.URL.Path, ".js"):
			typeName = "text/javascript; charset=utf-8"
		case strings.HasSuffix(r.URL.Path, ".css"):
			typeName = "text/css; charset=utf-8"
		case strings.HasSuffix(r.URL.Path, ".woff2"):
			typeName = "font/woff2"
		default:
			typeName = http.DetectContentType(content)
		}
	} else {
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			transportError(w, 500, "internal_error", "The page could not be prepared.")
			return
		}
		nonce := hex.EncodeToString(secret)
		// Only generated terminal styles carry this per-page nonce. Scripts keep
		// the original default-src policy and arbitrary inline styles stay blocked.
		w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self' ws://"+h.host+"; style-src 'self' 'nonce-"+nonce+"'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		content = []byte(strings.Replace(string(content), "<head>", "<head><meta name=\"pwnden-style-nonce\" content=\""+nonce+"\">", 1))
	}
	w.Header().Set("Content-Type", typeName)
	w.Header().Set("Content-Length", fmt.Sprint(len(content)))
	w.WriteHeader(http.StatusOK)
	if r.Method != "HEAD" {
		w.Write(content)
	}
}
