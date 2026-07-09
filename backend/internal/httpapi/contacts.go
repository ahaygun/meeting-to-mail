package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"meeting-to-mail/internal/domain"
)

// listContacts, kayıtlı alıcıları (en son kullanılan önce) döner.
func (s *Server) listContacts(w http.ResponseWriter, r *http.Request) {
	cs, err := s.st.ListContacts(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if cs == nil {
		cs = []domain.Contact{}
	}
	writeJSON(w, http.StatusOK, cs)
}

type contactReq struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// createContact, yeni bir kişi ekler (ya da mevcutsa günceller).
func (s *Server) createContact(w http.ResponseWriter, r *http.Request) {
	var req contactReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "geçersiz JSON")
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	if !strings.Contains(req.Email, "@") {
		writeErr(w, http.StatusBadRequest, "geçersiz e-posta")
		return
	}
	if err := s.st.UpsertContact(r.Context(), req.Email, strings.TrimSpace(req.Name)); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"email": req.Email})
}

// updateContact, bir kişinin adını günceller.
func (s *Server) updateContact(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "cid"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "geçersiz id")
		return
	}
	var req contactReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "geçersiz JSON")
		return
	}
	if err := s.st.UpdateContact(r.Context(), id, strings.TrimSpace(req.Name)); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"name": req.Name})
}

// deleteContact, bir kişiyi rehberden siler.
func (s *Server) deleteContact(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "cid"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "geçersiz id")
		return
	}
	if err := s.st.DeleteContact(r.Context(), id); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
