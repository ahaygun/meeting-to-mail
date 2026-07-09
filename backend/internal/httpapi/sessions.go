package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"meeting-to-mail/internal/domain"
	"meeting-to-mail/internal/store"
)

// createSessionReq, kayıt öncesi kurulum ekranından gelen konfig.
type createSessionReq struct {
	Title               string   `json:"title"`
	Recipients          []string `json:"recipients"`
	Participants        []string `json:"participants"`
	SummaryStyle        string   `json:"summary_style"`
	SendPolicy          string   `json:"send_policy"`
	CancelWindowSeconds int      `json:"cancel_window_seconds"`
}

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	var req createSessionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "geçersiz JSON")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeErr(w, http.StatusBadRequest, "başlık gerekli")
		return
	}

	recipients := cleanList(req.Recipients)
	if len(recipients) == 0 {
		writeErr(w, http.StatusBadRequest, "en az bir alıcı gerekli")
		return
	}
	for _, e := range recipients {
		if !strings.Contains(e, "@") {
			writeErr(w, http.StatusBadRequest, "geçersiz e-posta: "+e)
			return
		}
	}

	style := req.SummaryStyle
	if style == "" {
		style = "decisions_actions"
	}
	policy := req.SendPolicy
	if policy != domain.SendCancelWindow {
		policy = domain.SendImmediate
	}
	window := req.CancelWindowSeconds
	if policy != domain.SendCancelWindow {
		window = 0
	}

	sess := &domain.Session{
		ID:                  uuid.New(),
		Title:               req.Title,
		Status:              domain.StatusConfiguring,
		SummaryStyle:        style,
		SendPolicy:          policy,
		CancelWindowSeconds: window,
		Recipients:          recipients,
		Participants:        cleanList(req.Participants),
	}
	if err := s.st.CreateSession(r.Context(), sess); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Alıcıları kişi rehberine kaydet (bir dahaki sefere seçilebilsinler).
	for _, e := range recipients {
		_ = s.st.UpsertContact(r.Context(), e, "")
	}

	writeJSON(w, http.StatusCreated, sess)
}

// listSessions, geçmiş oturumları (en yeni önce) döner.
func (s *Server) listSessions(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	items, err := s.st.ListSessions(r.Context(), limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []domain.SessionListItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) getSession(w http.ResponseWriter, r *http.Request) {
	id, err := sessionID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "geçersiz oturum id")
		return
	}
	sess, err := s.st.GetSession(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "oturum bulunamadı")
		} else {
			writeErr(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request) {
	id, ok := s.loadSession(w, r)
	if !ok {
		return
	}
	if err := s.st.MarkStarted(r.Context(), id, nowUTC()); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": domain.StatusRecording})
}

// finalizeSession, kaydı sonlandırır ve boru hattını (transcribe işi) başlatır.
func (s *Server) finalizeSession(w http.ResponseWriter, r *http.Request) {
	id, ok := s.loadSession(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	if err := s.st.MarkEnded(ctx, id, nowUTC()); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := s.st.CreateJob(ctx, id, domain.JobTranscribe, nowUTC()); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": domain.StatusProcessing})
}

// cancelSession, bekleyen send işini iptal eder (cancel_window akışı).
func (s *Server) cancelSession(w http.ResponseWriter, r *http.Request) {
	id, ok := s.loadSession(w, r)
	if !ok {
		return
	}
	n, err := s.st.CancelPendingSend(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if n == 0 {
		writeErr(w, http.StatusConflict, "iptal edilecek bekleyen gönderim yok")
		return
	}
	_ = s.st.UpdateSessionStatus(r.Context(), id, domain.StatusCancelled)
	writeJSON(w, http.StatusOK, map[string]string{"status": domain.StatusCancelled})
}

func (s *Server) getTranscript(w http.ResponseWriter, r *http.Request) {
	id, ok := s.loadSession(w, r)
	if !ok {
		return
	}
	t, err := s.st.GetTranscript(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "transkript henüz yok")
		} else {
			writeErr(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) getSummary(w http.ResponseWriter, r *http.Request) {
	id, ok := s.loadSession(w, r)
	if !ok {
		return
	}
	sum, err := s.st.LatestSummary(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "özet henüz yok")
		} else {
			writeErr(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

func (s *Server) getDeliveries(w http.ResponseWriter, r *http.Request) {
	id, ok := s.loadSession(w, r)
	if !ok {
		return
	}
	ds, err := s.st.ListDeliveries(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ds)
}

// cleanList, boşlukları temizler ve boş öğeleri atar.
func cleanList(in []string) []string {
	var out []string
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}
