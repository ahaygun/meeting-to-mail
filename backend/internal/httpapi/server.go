// Package httpapi, HTTP router'ını ve handler'larını içerir.
package httpapi

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"

	"meeting-to-mail/internal/events"
	"meeting-to-mail/internal/storage"
	"meeting-to-mail/internal/store"
)

// Server, HTTP handler'larının bağımlılıklarını taşır.
type Server struct {
	st         store.Store
	disk       *storage.Disk
	hub        *events.Hub
	corsOrigin string
}

// NewServer bir Server oluşturur.
func NewServer(st store.Store, disk *storage.Disk, hub *events.Hub, corsOrigin string) *Server {
	return &Server{st: st, disk: disk, hub: hub, corsOrigin: corsOrigin}
}

// Router, tüm rotaları kurar ve http.Handler döner.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		// Dev kolaylığı + telefondan LAN erişimi: yapılandırılmış origin'e ek olarak
		// localhost / 127.0.0.1 / özel ağ (LAN) origin'lerine izin ver.
		AllowOriginFunc:  s.allowOrigin,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Accept"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/contacts", func(r chi.Router) {
		r.Get("/", s.listContacts)
		r.Post("/", s.createContact)
		r.Patch("/{cid}", s.updateContact)
		r.Delete("/{cid}", s.deleteContact)
	})

	r.Route("/api/groups", func(r chi.Router) {
		r.Get("/", s.listGroups)
		r.Post("/", s.createGroup)
		r.Delete("/{gid}", s.deleteGroup)
	})

	r.Route("/api/sessions", func(r chi.Router) {
		r.Get("/", s.listSessions)
		r.Post("/", s.createSession)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", s.getSession)
			r.Post("/start", s.startSession)
			r.Post("/chunks", s.uploadChunk)
			r.Post("/finalize", s.finalizeSession)
			r.Post("/cancel", s.cancelSession)
			r.Get("/transcript", s.getTranscript)
			r.Get("/summary", s.getSummary)
			r.Get("/deliveries", s.getDeliveries)
			r.Get("/events", s.streamEvents)
		})
	})

	return r
}

// allowOrigin, CORS için origin'i onaylar: yapılandırılmış origin, localhost,
// 127.0.0.1 ve özel ağ (LAN) adresleri kabul edilir. Public origin'ler reddedilir.
func (s *Server) allowOrigin(r *http.Request, origin string) bool {
	if origin == s.corsOrigin {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsPrivate() || ip.IsLoopback()
	}
	return false
}

// --- yardımcılar ---

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// sessionID, URL'den oturum ID'sini ayrıştırır ve doğrular.
func sessionID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, "id"))
}

// loadSession, ID'yi ayrıştırıp oturumu getirir; hata durumunda cevabı yazar.
func (s *Server) loadSession(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := sessionID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "geçersiz oturum id")
		return uuid.Nil, false
	}
	if _, err := s.st.GetSession(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "oturum bulunamadı")
		} else {
			writeErr(w, http.StatusInternalServerError, err.Error())
		}
		return uuid.Nil, false
	}
	return id, true
}

// nowUTC, tutarlı zaman damgası.
func nowUTC() time.Time { return time.Now().UTC() }
