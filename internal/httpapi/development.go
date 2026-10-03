package httpapi

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// Frontend is a launcher-owned Vite endpoint, never an API request parameter.
type Frontend struct {
	proxy *httputil.ReverseProxy
	nonce string
}

var developmentNonce = regexp.MustCompile(`^[a-f0-9]{64}$`)

func DevelopmentFrontend(address, nonce string) (*Frontend, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, errors.New("invalid development frontend URL")
	}
	host, port, err := net.SplitHostPort(u.Host)
	number, portErr := strconv.Atoi(port)
	if err != nil || portErr != nil || number < 1 || number > 65535 || host != "127.0.0.1" ||
		u.Scheme != "http" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || !developmentNonce.MatchString(nonce) {
		return nil, errors.New("dev requires its launcher-owned loopback frontend and style nonce")
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(request *httputil.ProxyRequest) {
			request.SetURL(u)
			// The frontend never receives the player's bearer token or cookies.
			request.Out.Header.Del("Authorization")
			request.Out.Header.Del("Cookie")
		},
		Transport: &http.Transport{Proxy: nil},
		ErrorLog:  log.New(io.Discard, "", 0),
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			transportError(w, 502, "development_unavailable", "The development frontend is unavailable.")
		},
	}
	return &Frontend{proxy: proxy, nonce: nonce}, nil
}

func ServeDevelopment(ctx context.Context, backend Backend, stdout, stderr io.Writer, frontend *Frontend) error {
	if frontend == nil {
		return errors.New("development frontend is required")
	}
	defer frontend.proxy.Transport.(*http.Transport).CloseIdleConnections()
	return serve(ctx, backend, stdout, stderr, frontend)
}

func (h *handler) development(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	page := playerPage(path)
	if page && r.URL.RawPath != "" {
		transportError(w, 400, "invalid_request", "Encoded page paths are not accepted.")
		return
	}
	allowed := page || path == "/__vite_hmr"
	for _, prefix := range []string{"/src/", "/@vite/", "/@id/", "/@fs/web/", "/node_modules/", "/assets/"} {
		allowed = allowed || strings.HasPrefix(path, prefix)
	}
	if !allowed {
		transportError(w, 404, "not_found", "The development asset was not found.")
		return
	}
	if !method(w, r, "GET", "HEAD") || !emptyBody(w, r) {
		return
	}
	if path == "/__vite_hmr" && r.Header.Get("Origin") != "http://"+h.host {
		transportError(w, 403, "forbidden", "A same-origin HMR connection is required.")
		return
	}
	w.Header().Set("Content-Security-Policy", pagePolicy(h.host, h.frontend.nonce))
	// Override upstream headers while retaining Vite's MIME and cache semantics.
	proxy := *h.frontend.proxy
	proxy.ModifyResponse = func(response *http.Response) error {
		response.Header.Set("Content-Security-Policy", pagePolicy(h.host, h.frontend.nonce))
		response.Header.Set("Cache-Control", "no-store")
		response.Header.Set("X-Content-Type-Options", "nosniff")
		response.Header.Set("Referrer-Policy", "no-referrer")
		response.Header.Del("Access-Control-Allow-Origin")
		response.Header.Del("Set-Cookie")
		return nil
	}
	if page {
		// Serve Vite's entry independently of its Accept-based SPA fallback.
		// The browser keeps the requested route for the client router.
		r = r.Clone(r.Context())
		r.URL.Path = "/"
		r.URL.RawPath = ""
	}
	proxy.ServeHTTP(w, r)
}

func pagePolicy(host, nonce string) string {
	return "default-src 'self'; script-src 'self' 'wasm-unsafe-eval'; connect-src 'self' ws://" + host + "; style-src 'self' 'nonce-" + nonce + "'; frame-src http://127.0.0.1:*; frame-ancestors 'none'; base-uri 'none'; form-action 'none'"
}
