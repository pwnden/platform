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
)

const terminalProtocol = "pwnden.terminal.v2"

type terminalConnection struct {
	cancel context.CancelFunc
	done   chan struct{}
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
	if r.Method == "DELETE" {
		if !emptyBody(w, r) {
			return
		}
		if len(r.Header.Values("Authorization")) != 1 || subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+h.token)) != 1 {
			transportError(w, 401, "unauthorized", "A server session token is required.")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), operationTimeout)
		defer cancel()
		release, err := h.acquire(ctx, slug)
		if err != nil {
			applicationError(w, err)
			return
		}
		defer release()
		if err := h.stopTerminal(slug); err != nil {
			applicationError(w, err)
			return
		}
		if err := h.workspaces.EndTerminal(slug); err != nil {
			applicationError(w, err)
			return
		}
		writeJSON(w, 200, Stop{Slug: slug})
		return
	}
	if !method(w, r, "GET", "DELETE") || !emptyBody(w, r) {
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
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	stopBase := context.AfterFunc(h.base, cancel)
	defer stopBase()
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
	if err != nil {
		if errors.Is(auth.Err(), context.DeadlineExceeded) {
			fail("deadline_exceeded")
		} else {
			fail("network_error")
		}
		return
	}
	message, decodeErr := decodeTerminal(data)
	if typ != websocket.MessageText || decodeErr != nil || message.Type != "authenticate" {
		fail("invalid_request")
		return
	}
	if subtle.ConstantTimeCompare([]byte(message.Token), []byte(h.token)) != 1 {
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
	previous := h.terminals[slug]
	h.mu.Unlock()
	if previous != nil {
		// Let a closing previous attachment finish during a quick A→B→A
		// transition. A live attachment keeps ownership and receives a conflict.
		timer := time.NewTimer(300 * time.Millisecond)
		select {
		case <-previous.done:
		case <-ctx.Done():
			timer.Stop()
			release()
			return
		case <-timer.C:
			release()
			fail("terminal_busy")
			return
		}
		timer.Stop()
	}
	session, err := h.workspaces.Attach(slug, message.Cols, message.Rows)
	if err != nil {
		release()
		_, response := ErrorFrom(err)
		fail(response.Error.Code)
		return
	}
	entry := &terminalConnection{cancel: cancel, done: make(chan struct{})}
	h.mu.Lock()
	h.terminals[slug] = entry
	h.mu.Unlock()
	release()
	defer func() {
		cancel()
		session.Close()
		h.mu.Lock()
		delete(h.terminals, slug)
		close(entry.done)
		h.mu.Unlock()
	}()
	if send(map[string]any{"type": "ready", "reused": session.Reused}) != nil {
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
				if err := session.Input(data); err != nil {
					fail("input_backpressure")
					cancel()
					return
				}
				continue
			}
			control, err := decodeTerminal(data)
			if err == nil && control.Type == "ack" {
				select {
				case ack <- struct{}{}:
				default:
					fail("invalid_argument")
					cancel()
					return
				}
				continue
			}
			if err != nil || control.Type != "resize" || control.Cols < 2 || control.Cols > 500 || control.Rows < 1 || control.Rows > 200 {
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
	defer func() { cancel(); <-inputDone }()
	go func() {
		tick := time.NewTicker(20 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				ping, done := context.WithTimeout(ctx, 5*time.Second)
				err := conn.Ping(ping)
				done()
				if err != nil {
					cancel()
					return
				}
			}
		}
	}()
	write := func(data []byte) bool {
		for len(data) > 0 {
			n := min(len(data), 16<<10)
			writeCtx, done := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Write(writeCtx, websocket.MessageBinary, data[:n])
			done()
			if err != nil {
				return false
			}
			timer := time.NewTimer(15 * time.Second)
			select {
			case <-ack:
				timer.Stop()
			case <-ctx.Done():
				timer.Stop()
				return false
			case <-timer.C:
				fail("output_backpressure")
				return false
			}
			data = data[n:]
		}
		return true
	}
	if !write(session.Snapshot) {
		return
	}
	if err := session.Resize(ctx, message.Cols, message.Rows); err != nil {
		fail("execution_failed")
		return
	}
	for {
		data, err := session.Next(ctx)
		if len(data) > 0 && !write(data) {
			return
		}
		if err != nil {
			if errors.Is(err, io.EOF) && ctx.Err() == nil {
				code, failure := session.Result()
				if failure != nil {
					fail("execution_failed")
					return
				}
				if send(map[string]any{"type": "exit", "code": code}) != nil {
					return
				}
				// Keep problem presence while the player reads its visible page.
				<-ctx.Done()
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
	return nil
}
