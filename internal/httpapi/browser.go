package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "embed"

	"github.com/pwnden/platform/internal/application"
)

//go:embed browser-controller.js
var browserController string

var errBrowserUnavailable = errors.New("browser endpoint unavailable")

type Browser struct {
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	URL    string `json:"url"`
	Target string `json:"target"`
}

type browserManager struct {
	ctx      context.Context
	status   func(context.Context, string) (application.RunStatus, error)
	dial     application.EndpointDialer
	observe  application.EndpointObserver
	parent   string
	owned    func() []application.WorkspaceInfo
	mu       sync.Mutex
	closed   bool
	sweeping bool
	entries  map[string]*browserEntry
	active   sync.WaitGroup
}

type browserEntry struct {
	manager                            *browserManager
	slug, name, target, origin, prefix string
	instance                           string
	isolated                           bool
	cancel                             context.CancelFunc
	server                             *http.Server
	transport                          *http.Transport
	proxy                              *httputil.ReverseProxy
	closed                             atomic.Bool
}

func newBrowserManager(ctx context.Context, backend Backend, parent string, owned func() []application.WorkspaceInfo) *browserManager {
	dial, _ := backend.(application.EndpointDialer)
	observe, _ := backend.(application.EndpointObserver)
	return &browserManager{ctx: ctx, status: backend.Status, dial: dial, observe: observe, parent: parent, owned: owned, entries: make(map[string]*browserEntry)}
}

func localHTTP(raw string) (*url.URL, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() == "" || u.User != nil ||
		(u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return nil, false
	}
	port, err := strconv.Atoi(u.Port())
	return u, err == nil && port > 0 && port <= 65535
}

func browserTarget(status application.RunStatus, name string) (string, bool) {
	if status.State != "running" || status.Kind != application.KindService {
		return "", false
	}
	for _, endpoint := range status.Endpoints {
		if endpoint.Name == name && endpoint.Proxied && endpoint.Instance != "" {
			u, err := url.Parse(endpoint.URL)
			if err == nil && u.Scheme == "http" && u.Hostname() != "" && u.User == nil && u.Path == "" && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" {
				port, err := strconv.Atoi(u.Port())
				if err == nil && port > 0 && port <= 65535 {
					return endpoint.URL, true
				}
			}
		}
		if endpoint.Name == name && endpoint.Published {
			if _, ok := localHTTP(endpoint.URL); ok {
				return strings.TrimSuffix(endpoint.URL, "/"), true
			}
		}
	}
	return "", false
}

func (m *browserManager) open(ctx context.Context, slug, name string) (Browser, error) {
	status, err := m.status(ctx, slug)
	if err != nil {
		return Browser{}, err
	}
	return m.openObserved(ctx, slug, name, status)
}

// The HTTP run/status adapter already holds the problem lock and has a current
// observation. Reuse it rather than trying to acquire the same lock again.
func (m *browserManager) openObserved(ctx context.Context, slug, name string, status application.RunStatus) (Browser, error) {
	target, ok := browserTarget(status, name)
	if !ok || target == m.parent {
		return Browser{}, errBrowserUnavailable
	}
	instance, isolated := browserInstance(status, name)
	if isolated && m.dial == nil {
		return Browser{}, errBrowserUnavailable
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.ctx.Err() != nil || ctx.Err() != nil {
		return Browser{}, context.Canceled
	}
	key := slug + "\x00" + name
	if entry := m.entries[key]; entry != nil {
		if entry.target == target && entry.instance == instance && !entry.closed.Load() {
			return entry.description(), nil
		}
		entry.stop()
		delete(m.entries, key)
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return Browser{}, err
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		listener.Close()
		return Browser{}, err
	}
	base, cancel := context.WithCancel(m.ctx)
	entry := &browserEntry{manager: m, slug: slug, name: name, target: target,
		instance: instance, isolated: isolated,
		origin: "http://" + listener.Addr().String(), prefix: "/__pwnden_browser/" + hex.EncodeToString(secret), cancel: cancel}
	upstream, _ := url.Parse(target)
	entry.transport = &http.Transport{Proxy: nil, DisableCompression: true, ResponseHeaderTimeout: 30 * time.Second,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			var conn net.Conn
			var err error
			if isolated {
				conn, err = m.dial.DialEndpoint(ctx, slug, name, instance)
			} else {
				conn, err = (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, address)
			}
			if err != nil {
				return nil, err
			}
			stop := context.AfterFunc(base, func() { conn.Close() })
			return &browserConnection{Conn: conn, cancel: stop}, nil
		}}
	entry.proxy = &httputil.ReverseProxy{Transport: entry.transport, ErrorLog: log.New(io.Discard, "", 0),
		Rewrite: func(request *httputil.ProxyRequest) {
			request.SetURL(upstream)
			// Exercise requests may intentionally use unusual queries or forwarded
			// headers. Preserve them rather than applying gateway normalization.
			request.Out.URL.RawQuery = request.In.URL.RawQuery
			for _, key := range []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto"} {
				if values, ok := request.In.Header[key]; ok {
					request.Out.Header[key] = append([]string(nil), values...)
				}
			}
			// Keep the browser-visible host for generated links and Origin checks.
			request.Out.Host = request.In.Host
			request.Out.Header.Del("Proxy-Authorization")
		},
		ModifyResponse: func(response *http.Response) error {
			if location := response.Header.Get("Location"); location != "" {
				u, err := url.Parse(location)
				if err == nil && u.Host == upstream.Host && (u.Scheme == "http" || u.Scheme == "") {
					u.Host = strings.TrimPrefix(entry.origin, "http://")
					response.Header.Set("Location", u.String())
				}
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, "Problem web unavailable", http.StatusBadGateway)
		},
	}
	entry.server = &http.Server{Handler: entry, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second,
		MaxHeaderBytes: 16 << 10, ErrorLog: log.New(io.Discard, "", 0), BaseContext: func(net.Listener) context.Context { return base }}
	m.entries[key] = entry
	go func() { _ = entry.server.Serve(listener) }()
	if !m.sweeping && m.owned != nil {
		m.sweeping = true
		go m.sweep()
	}
	return entry.description(), nil
}

type browserConnection struct {
	net.Conn
	cancel func() bool
}

func (c *browserConnection) Close() error { c.cancel(); return c.Conn.Close() }

func (e *browserEntry) description() Browser {
	return Browser{e.slug, e.name, e.origin + e.prefix, e.visibleTarget()}
}

func (e *browserEntry) visibleTarget() string {
	if e.isolated {
		return e.origin
	}
	return e.target
}

func browserInstance(status application.RunStatus, name string) (string, bool) {
	for _, endpoint := range status.Endpoints {
		if endpoint.Name == name {
			return endpoint.Instance, endpoint.Proxied
		}
	}
	return "", false
}
func (e *browserEntry) stop() {
	if !e.closed.Swap(true) {
		e.cancel()
		_ = e.server.Close()
		e.transport.CloseIdleConnections()
	}
}

func (m *browserManager) sweep() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
		}
		owned := make(map[string]bool)
		for _, workspace := range m.owned() {
			owned[workspace.Slug] = true
		}
		m.mu.Lock()
		if m.closed {
			m.mu.Unlock()
			return
		}
		for key, entry := range m.entries {
			if !owned[entry.slug] {
				entry.stop()
				delete(m.entries, key)
			}
		}
		m.mu.Unlock()
	}
}

func (m *browserManager) close() {
	m.mu.Lock()
	m.closed = true
	for _, entry := range m.entries {
		entry.stop()
	}
	m.entries = make(map[string]*browserEntry)
	m.mu.Unlock()
	m.active.Wait()
}

var browserDocument = template.Must(template.New("browser").Parse(`<!doctype html><html lang="ko"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>문제 웹</title><link rel="stylesheet" href="{{.Prefix}}/style.css"><script src="{{.Prefix}}/controller.js" defer></script></head><body data-parent="{{.Parent}}" data-target="{{.Target}}"><iframe id="problem" title="문제 사이트" referrerpolicy="no-referrer" sandbox="allow-scripts allow-same-origin allow-forms allow-modals allow-popups allow-downloads"></iframe></body></html>`))

func (e *browserEntry) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := e.manager
	m.mu.Lock()
	if m.closed || e.closed.Load() {
		m.mu.Unlock()
		http.Error(w, "Browser closed", 410)
		return
	}
	m.active.Add(1)
	m.mu.Unlock()
	defer m.active.Done()
	if r.Host != strings.TrimPrefix(e.origin, "http://") {
		http.Error(w, "Invalid host", 403)
		return
	}
	if strings.HasPrefix(r.URL.Path, e.prefix) {
		if r.Method != "GET" || r.URL.RawQuery != "" || r.URL.RawPath != "" {
			http.Error(w, "Invalid browser request", 400)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; frame-src 'self'; frame-ancestors "+m.parent+"; base-uri 'none'; form-action 'none'")
		switch r.URL.Path {
		case e.prefix:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_ = browserDocument.Execute(w, struct{ Prefix, Parent, Target string }{e.prefix, m.parent, e.visibleTarget()})
		case e.prefix + "/controller.js":
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			_, _ = io.WriteString(w, browserController)
		case e.prefix + "/style.css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
			_, _ = io.WriteString(w, "html,body{margin:0;height:100%;overflow:hidden}iframe{display:block;width:100%;height:100%;border:0}")
		default:
			http.NotFound(w, r)
		}
		return
	}
	// Resolve only the declared, still-running endpoint. The caller supplies paths,
	// never a destination URL. Stop/restart cannot retarget an old browser origin.
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	var err error
	if e.isolated && m.observe != nil {
		err = m.observe.CheckEndpoint(ctx, e.slug, e.name, e.instance, e.target)
	} else {
		var status application.RunStatus
		status, err = m.status(ctx, e.slug)
		target, ok := browserTarget(status, e.name)
		instance, _ := browserInstance(status, e.name)
		if !ok || target != e.target || instance != e.instance {
			err = errBrowserUnavailable
		}
	}
	cancel()
	if err != nil {
		http.Error(w, "Problem environment closed", 410)
		return
	}
	e.proxy.ServeHTTP(w, r)
}

// Expose only declared HTTP services, on separate loopback origins owned by the
// local server. HTTP container addresses stay inside the application layer.
func (h *handler) exposeEndpoints(ctx context.Context, slug string, endpoints []application.Endpoint) ([]application.Endpoint, error) {
	result := make([]application.Endpoint, 0, len(endpoints))
	status := application.RunStatus{Slug: slug, Kind: application.KindService, State: "running", Endpoints: endpoints}
	for _, endpoint := range endpoints {
		if endpoint.Proxied {
			browser, err := h.browsers.openObserved(ctx, slug, endpoint.Name, status)
			if err != nil {
				return nil, err
			}
			endpoint.URL, endpoint.Published = browser.Target, true
		}
		result = append(result, endpoint)
	}
	return result, nil
}
