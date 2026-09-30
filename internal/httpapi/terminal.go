package httpapi

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/pwnden/platform/internal/application"
)

const terminalProtocol = "pwnden.terminal.v1"

type terminalConnection struct {
	cancel context.CancelFunc
	done   chan struct{}
	err    error // published before done closes
}
type terminalMessage struct {
	Type  string `json:"type"`
	Token string `json:"token,omitempty"`
	Cols  int    `json:"cols,omitempty"`
	Rows  int    `json:"rows,omitempty"`
}

func decodeTerminal(data []byte) (terminalMessage, error) {
	var result terminalMessage
	// Reject duplicate keys as well as unknown/case-variant fields.
	d := json.NewDecoder(bytes.NewReader(data))
	open, err := d.Token()
	if err != nil || open != json.Delim('{') {
		return result, errors.New("invalid control message")
	}
	seen := map[string]bool{}
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return result, err
		}
		name, ok := key.(string)
		if !ok || seen[name] {
			return result, errors.New("duplicate control field")
		}
		seen[name] = true
		switch name {
		case "type":
			err = d.Decode(&result.Type)
		case "token":
			err = d.Decode(&result.Token)
		case "cols":
			err = d.Decode(&result.Cols)
		case "rows":
			err = d.Decode(&result.Rows)
		default:
			return result, errors.New("unknown control field")
		}
		if err != nil {
			return result, err
		}
	}
	if _, err := d.Token(); err != nil {
		return result, err
	}
	if _, err := d.Token(); err != io.EOF {
		return result, errors.New("trailing control data")
	}
	valid := false
	switch result.Type {
	case "authenticate":
		valid = len(seen) == 4 && seen["type"] && seen["token"] && seen["cols"] && seen["rows"]
	case "resize":
		valid = len(seen) == 3 && seen["type"] && seen["cols"] && seen["rows"]
	case "ack":
		valid = len(seen) == 1 && seen["type"]
	}
	if !valid {
		return result, errors.New("invalid control fields")
	}
	return result, nil
}

func (h *handler) terminal(w http.ResponseWriter, r *http.Request, slug string) {
	if !method(w, r, "GET") || !emptyBody(w, r) {
		return
	}
	if r.Header.Get("Origin") != "http://"+h.host {
		transportError(w, 403, "forbidden", "A same-origin terminal connection is required.")
		return
	}
	if len(r.Header.Values("Sec-WebSocket-Protocol")) != 1 || r.Header.Get("Sec-WebSocket-Protocol") != terminalProtocol {
		transportError(w, 400, "invalid_request", "Use the terminal protocol.")
		return
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{terminalProtocol}})
	if err != nil {
		return
	}
	defer conn.CloseNow()
	conn.SetReadLimit(16 << 10)
	ctx, cancel := context.WithTimeout(r.Context(), operationTimeout)
	defer cancel()
	stopBase := context.AfterFunc(h.base, cancel)
	defer stopBase()
	if h.base.Err() != nil {
		cancel()
	}
	stopSocket := context.AfterFunc(ctx, func() { conn.CloseNow() })
	defer stopSocket()
	send := func(value any) error {
		data, _ := json.Marshal(value)
		writeCtx, done := context.WithTimeout(ctx, 5*time.Second)
		defer done()
		return conn.Write(writeCtx, websocket.MessageText, data)
	}
	fail := func(code string) { send(map[string]any{"type": "error", "code": code}) }
	auth, done := context.WithTimeout(ctx, 5*time.Second)
	typ, data, err := conn.Read(auth)
	done()
	message, decodeErr := decodeTerminal(data)
	if err != nil || typ != websocket.MessageText || decodeErr != nil || message.Type != "authenticate" ||
		subtle.ConstantTimeCompare([]byte(message.Token), []byte(h.token)) != 1 {
		fail("unauthorized")
		return
	}
	if message.Cols < 2 || message.Cols > 500 || message.Rows < 1 || message.Rows > 200 {
		fail("invalid_argument")
		return
	}
	release, err := h.acquire(ctx, slug)
	if err != nil {
		fail("canceled")
		return
	}
	h.mu.Lock()
	_, busy := h.terminals[slug]
	h.mu.Unlock()
	if busy {
		release()
		fail("terminal_busy")
		return
	}
	session, err := h.backend.OpenTerminal(ctx, slug, message.Cols, message.Rows)
	if err != nil {
		release()
		_, response := ErrorFrom(err)
		send(map[string]any{"type": "error", "code": response.Error.Code})
		return
	}
	entry := &terminalConnection{cancel: cancel, done: make(chan struct{})}
	h.mu.Lock()
	h.terminals[slug] = entry
	h.mu.Unlock()
	release()
	defer func() {
		cancel()
		entry.err = session.Close()
		h.mu.Lock()
		delete(h.terminals, slug)
		if entry.err != nil {
			h.cleanupFailures = errors.Join(h.cleanupFailures, entry.err)
			// No shell output or backend cause enters diagnostics.
			io.WriteString(h.diagnostics, "pwnden: terminal cleanup_failed for "+slug+"\n")
		}
		close(entry.done)
		h.mu.Unlock()
	}()
	if send(map[string]any{"type": "ready"}) != nil {
		return
	}
	inputDone := make(chan struct{})
	ack := make(chan struct{}, 1)
	go func() {
		defer close(inputDone)
		for {
			typ, data, err := conn.Read(ctx)
			if err != nil {
				cancel()
				return
			}
			if typ == websocket.MessageBinary {
				if _, err := session.Write(data); err != nil {
					cancel()
					return
				}
				continue
			}
			control, err := decodeTerminal(data)
			if err == nil && control.Type == "ack" && control.Token == "" && control.Cols == 0 && control.Rows == 0 {
				select {
				case ack <- struct{}{}:
				default:
					fail("invalid_argument")
					cancel()
					return
				}
				continue
			}
			if err != nil || control.Type != "resize" || control.Token != "" || control.Cols < 2 || control.Cols > 500 || control.Rows < 1 || control.Rows > 200 {
				fail("invalid_argument")
				cancel()
				return
			}
			resizeCtx, done := context.WithTimeout(ctx, 5*time.Second)
			err = session.Resize(resizeCtx, control.Cols, control.Rows)
			done()
			if err != nil {
				fail("execution_failed")
				cancel()
				return
			}
		}
	}()
	// Closing both ends on cancellation unblocks writes as well as reads. Wait
	// for the reader before publishing cleanup completion to a problem stop.
	stopStream := context.AfterFunc(ctx, func() { session.Close() })
	defer func() { cancel(); session.Close(); <-inputDone; stopStream() }()
	buffer := make([]byte, 16<<10)
	for {
		n, err := session.Read(buffer)
		if n > 0 {
			writeCtx, done := context.WithTimeout(ctx, 5*time.Second)
			failure := conn.Write(writeCtx, websocket.MessageBinary, buffer[:n])
			done()
			if failure != nil {
				return
			}
			ackTimer := time.NewTimer(15 * time.Second)
			select {
			case <-ack:
				ackTimer.Stop()
			case <-ctx.Done():
				ackTimer.Stop()
				return
			case <-ackTimer.C:
				fail("output_backpressure")
				return
			}
		}
		if err != nil {
			if err == io.EOF && ctx.Err() == nil {
				waitCtx, done := context.WithTimeout(ctx, 5*time.Second)
				code, waitErr := session.Wait(waitCtx)
				done()
				if waitErr == nil {
					send(map[string]any{"type": "exit", "code": code})
				} else {
					fail("execution_failed")
				}
			} else if ctx.Err() == nil {
				fail("execution_failed")
			}
			return
		}
	}
}

func (h *handler) stopTerminal(slug string) error {
	h.mu.Lock()
	entry := h.terminals[slug]
	h.mu.Unlock()
	if entry == nil {
		return nil
	}
	entry.cancel()
	<-entry.done
	if entry.err != nil {
		return &application.Error{Code: application.CleanupFailed, Operation: "terminal", Slug: slug, Cause: entry.err}
	}
	return nil
}
