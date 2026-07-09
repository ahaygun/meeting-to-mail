package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// streamEvents, bir oturumun ilerleme olaylarını Server-Sent Events olarak yayınlar.
func (s *Server) streamEvents(w http.ResponseWriter, r *http.Request) {
	id, ok := s.loadSession(w, r)
	if !ok {
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "streaming desteklenmiyor")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch, cancel := s.hub.Subscribe(id)
	defer cancel()

	// Bağlanır bağlanmaz mevcut durumu gönder (istemci geç abone olmuş olabilir).
	if sess, err := s.st.GetSession(r.Context(), id); err == nil {
		writeSSE(w, flusher, "state", map[string]string{"status": sess.Status})
	}

	// Bağlantıyı canlı tutmak için periyodik ping.
	ping := time.NewTicker(20 * time.Second)
	defer ping.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case e, open := <-ch:
			if !open {
				return
			}
			writeSSE(w, flusher, "progress", e)
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, f http.Flusher, event string, data any) {
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
	f.Flush()
}
