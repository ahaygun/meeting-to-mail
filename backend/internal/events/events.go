// Package events, oturum bazında ilerleme olaylarını yayınlayan basit bir
// pub/sub hub'ı sağlar. SSE endpoint'i abone olur, worker yayınlar.
package events

import (
	"sync"

	"github.com/google/uuid"
)

// Event, frontend'e gönderilen bir ilerleme olayı.
type Event struct {
	Status  string `json:"status"`            // güncel session durumu
	Message string `json:"message,omitempty"` // insana okunur açıklama
	Step    string `json:"step,omitempty"`    // transcribe | summarize | send ...
}

// Hub, oturum ID'sine göre abonelikleri yönetir.
type Hub struct {
	mu   sync.RWMutex
	subs map[uuid.UUID]map[chan Event]struct{}
}

// New bir Hub oluşturur.
func New() *Hub {
	return &Hub{subs: make(map[uuid.UUID]map[chan Event]struct{})}
}

// Subscribe, bir oturum için tamponlanmış kanal döner ve abonelik iptali fonksiyonu verir.
func (h *Hub) Subscribe(sessionID uuid.UUID) (<-chan Event, func()) {
	ch := make(chan Event, 16)
	h.mu.Lock()
	if h.subs[sessionID] == nil {
		h.subs[sessionID] = make(map[chan Event]struct{})
	}
	h.subs[sessionID][ch] = struct{}{}
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		if set, ok := h.subs[sessionID]; ok {
			if _, ok := set[ch]; ok {
				delete(set, ch)
				close(ch)
			}
			if len(set) == 0 {
				delete(h.subs, sessionID)
			}
		}
		h.mu.Unlock()
	}
	return ch, cancel
}

// Publish, bir oturumun tüm abonelerine olayı iletir (bloklamadan).
func (h *Hub) Publish(sessionID uuid.UUID, e Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs[sessionID] {
		select {
		case ch <- e:
		default:
			// Abone yavaşsa olayı düşür; SSE zaten güncel durumu tekrar çeker.
		}
	}
}
