package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/pwnden/platform/internal/application"
)

type fakeTerminal struct {
	output   *io.PipeReader
	writer   *io.PipeWriter
	input    chan []byte
	resize   chan [2]int
	closed   chan struct{}
	once     sync.Once
	closeErr error
}

func newFakeTerminal() *fakeTerminal {
	r, w := io.Pipe()
	return &fakeTerminal{output: r, writer: w, input: make(chan []byte, 4), resize: make(chan [2]int, 4), closed: make(chan struct{})}
}
func (s *fakeTerminal) Read(p []byte) (int, error) { return s.output.Read(p) }
func (s *fakeTerminal) Write(p []byte) (int, error) {
	s.input <- append([]byte(nil), p...)
	return len(p), nil
}
func (s *fakeTerminal) Resize(_ context.Context, c, r int) error {
	s.resize <- [2]int{c, r}
	return nil
}
func (s *fakeTerminal) Wait(context.Context) (int, error) { return 7, nil }
func (s *fakeTerminal) Close() error {
	s.once.Do(func() { s.output.Close(); s.writer.Close(); close(s.closed) })
	return s.closeErr
}

func terminalServer(t *testing.T, b fakeBackend) (*handler, *httptest.Server) {
	t.Helper()
	server := httptest.NewUnstartedServer(nil)
	h := newHandler(context.Background(), b, server.Listener.Addr().String(), testToken, io.Discard)
	server.Config.Handler = h
	server.Start()
	t.Cleanup(func() {
		h.mu.Lock()
		for _, e := range h.terminals {
			e.cancel()
		}
		h.mu.Unlock()
		h.active.Wait()
		h.workspaces.Close()
		server.Close()
	})
	return h, server
}
func dialTerminal(t *testing.T, server *httptest.Server, token string) *websocket.Conn {
	return dialTerminalControl(t, server, map[string]any{"type": "authenticate", "token": token, "cols": 80, "rows": 24})
}
func dialTerminalControl(t *testing.T, server *httptest.Server, control any) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+BasePath+"/problems/test/terminal", &websocket.DialOptions{
		Subprotocols: []string{terminalProtocol}, HTTPHeader: http.Header{"Origin": []string{server.URL}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.CloseNow() })
	writeTerminal(t, conn, control)
	return conn
}
func writeTerminal(t *testing.T, conn *websocket.Conn, value any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	data, _ := json.Marshal(value)
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatal(err)
	}
}
func readTerminal(t *testing.T, conn *websocket.Conn) (websocket.MessageType, []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	typ, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return typ, data
}

func TestTerminalAuthenticationAndAdmission(t *testing.T) {
	var calls atomic.Int32
	b := fakeBackend{terminal: func(context.Context, string, int, int) (application.TerminalSession, error) {
		calls.Add(1)
		return newFakeTerminal(), nil
	}}
	h, server := terminalServer(t, b)
	malformed := dialTerminalControl(t, server, map[string]any{"type": "authenticate", "token": testToken, "cols": 80, "rows": 24, "colsChanged": true, "rowsChanged": true})
	_, malformedData := readTerminal(t, malformed)
	if string(malformedData) != `{"code":"invalid_request","type":"error"}` || calls.Load() != 0 {
		t.Fatalf("malformed dimensions misreported as rejected credentials: %s", malformedData)
	}
	conn := dialTerminal(t, server, "wrong")
	_, data := readTerminal(t, conn)
	if string(data) != `{"code":"unauthorized","type":"error"}` || calls.Load() != 0 {
		t.Fatalf("unauthorized shell created: %s", data)
	}
	for _, origin := range []string{"", "http://elsewhere"} {
		r := request("GET", BasePath+"/problems/test/terminal", "")
		r.Host = h.host
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatalf("origin %q: %d", origin, w.Code)
		}
	}
	conn = dialTerminal(t, server, testToken)
	_, data = readTerminal(t, conn)
	if string(data) != `{"reused":false,"type":"ready"}` {
		t.Fatal(string(data))
	}
	other := dialTerminal(t, server, testToken)
	_, data = readTerminal(t, other)
	if string(data) != `{"code":"terminal_busy","type":"error"}` || calls.Load() != 1 {
		t.Fatalf("duplicate admitted: %s", data)
	}
	conn.CloseNow()
	h.active.Wait()
}

func TestTerminalBytesResizeFlowAndExit(t *testing.T) {
	session := newFakeTerminal()
	h, server := terminalServer(t, fakeBackend{terminal: func(_ context.Context, _ string, c, r int) (application.TerminalSession, error) {
		if c != 80 || r != 24 {
			t.Errorf("size %dx%d", c, r)
		}
		return session, nil
	}})
	conn := dialTerminal(t, server, testToken)
	readTerminal(t, conn)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageBinary, []byte{3, 0, 255}); err != nil {
		t.Fatal(err)
	}
	select {
	case data := <-session.input:
		if string(data) != string([]byte{3, 0, 255}) {
			t.Fatal(data)
		}
	case <-ctx.Done():
		t.Fatal("input blocked")
	}
	writeTerminal(t, conn, map[string]any{"type": "resize", "cols": 120, "rows": 40})
	select {
	case size := <-session.resize:
		if size != [2]int{80, 24} {
			t.Fatal(size)
		}
	case <-ctx.Done():
		t.Fatal("initial resize blocked")
	}
	select {
	case size := <-session.resize:
		if size != [2]int{120, 40} {
			t.Fatal(size)
		}
	case <-ctx.Done():
		t.Fatal("resize blocked")
	}
	go func() { session.writer.Write([]byte{0, 255, 10}); session.writer.Close() }()
	typ, data := readTerminal(t, conn)
	if typ != websocket.MessageBinary || string(data) != string([]byte{0, 255, 10}) {
		t.Fatalf("bytes %v %v", typ, data)
	}
	writeTerminal(t, conn, map[string]any{"type": "ack"})
	_, data = readTerminal(t, conn)
	if string(data) != `{"code":7,"type":"exit"}` {
		t.Fatal(string(data))
	}
	conn.CloseNow()
	h.active.Wait()
	select {
	case <-session.closed:
	default:
		t.Fatal("exit leaked shell")
	}
}

func TestTerminalImmediateReconnectReusesShell(t *testing.T) {
	shell := newFakeTerminal()
	var calls atomic.Int32
	_, server := terminalServer(t, fakeBackend{terminal: func(context.Context, string, int, int) (application.TerminalSession, error) {
		calls.Add(1)
		return shell, nil
	}})
	first := dialTerminal(t, server, testToken)
	readTerminal(t, first)
	first.CloseNow()
	next := dialTerminal(t, server, testToken)
	_, data := readTerminal(t, next)
	if string(data) != `{"reused":true,"type":"ready"}` || calls.Load() != 1 {
		t.Fatalf("fast reattach replaced shell: %s", data)
	}
}

func TestProblemStopClosesTerminalBeforeService(t *testing.T) {
	session := newFakeTerminal()
	h, server := terminalServer(t, fakeBackend{
		terminal: func(context.Context, string, int, int) (application.TerminalSession, error) { return session, nil },
		stop: func(context.Context, string) (application.StopInfo, error) {
			select {
			case <-session.closed:
			default:
				t.Error("service stopped before terminal")
			}
			return application.StopInfo{Slug: "test"}, nil
		},
	})
	conn := dialTerminal(t, server, testToken)
	readTerminal(t, conn)
	r := request("DELETE", BasePath+"/problems/test/run", "")
	r.Host = h.host
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("stop %d %s", w.Code, w.Body.String())
	}
	h.active.Wait()
}

func TestTerminalInvalidControlAndServerCancellation(t *testing.T) {
	for _, control := range []string{`{"type":"resize","cols":0,"rows":24}`, `{"type":"resize","type":"resize","cols":80,"rows":24}`, `{"type":"resize","cols":80,"rows":24,"path":"/etc"}`} {
		session := newFakeTerminal()
		h, server := terminalServer(t, fakeBackend{terminal: func(context.Context, string, int, int) (application.TerminalSession, error) { return session, nil }})
		conn := dialTerminal(t, server, testToken)
		readTerminal(t, conn)
		if err := conn.Write(context.Background(), websocket.MessageText, []byte(control)); err != nil {
			t.Fatal(err)
		}
		_, data := readTerminal(t, conn)
		if string(data) != `{"code":"invalid_argument","type":"error"}` {
			t.Fatal(string(data))
		}
		h.active.Wait()
	}
	session := newFakeTerminal()
	h, server := terminalServer(t, fakeBackend{terminal: func(context.Context, string, int, int) (application.TerminalSession, error) { return session, nil }})
	base, cancel := context.WithCancel(context.Background())
	h.base = base
	conn := dialTerminal(t, server, testToken)
	readTerminal(t, conn)
	cancel()
	h.active.Wait()
	h.workspaces.Close()
	select {
	case <-session.closed:
	default:
		t.Fatal("server shutdown leaked shell")
	}
}
