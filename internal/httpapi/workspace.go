package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// The authenticated stream retains a visible problem without creating a PTY.
func (h *handler) workspace(w http.ResponseWriter, r *http.Request, slug string) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	stop := context.AfterFunc(h.base, cancel)
	defer stop()
	prepare, done := context.WithTimeout(ctx, operationTimeout)
	defer done()
	release, err := h.acquire(prepare, slug)
	if err != nil {
		applicationError(w, contextError(err))
		return
	}
	view, err := h.workspaces.View(prepare, slug)
	if err != nil {
		release()
		applicationError(w, err)
		return
	}
	defer view.Close()
	status := view.Status
	status.Endpoints, err = h.exposeEndpoints(prepare, slug, status.Endpoints)
	release()
	if err != nil {
		applicationError(w, err)
		return
	}
	done()
	w.Header().Set("Content-Type", "application/x-ndjson")
	controller := http.NewResponseController(w)
	send := func(value any) bool {
		controller.SetWriteDeadline(time.Now().Add(5 * time.Second))
		err := json.NewEncoder(w).Encode(value)
		if err == nil {
			err = controller.Flush()
		}
		controller.SetWriteDeadline(time.Time{})
		return err == nil
	}
	if !send(map[string]any{"type": "ready", "status": StatusFrom(status)}) {
		return
	}
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !view.Active() {
				send(map[string]any{"type": "error", "code": "not_running"})
				return
			}
			if !send(map[string]string{"type": "heartbeat"}) {
				return
			}
		}
	}
}
