package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"meeting-to-mail/internal/domain"
)

// listGroups, kayıtlı alıcı gruplarını (dağıtım listeleri) döner.
func (s *Server) listGroups(w http.ResponseWriter, r *http.Request) {
	gs, err := s.st.ListGroups(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if gs == nil {
		gs = []domain.Group{}
	}
	writeJSON(w, http.StatusOK, gs)
}

type groupReq struct {
	Name   string   `json:"name"`
	Emails []string `json:"emails"`
}

// createGroup, bir grup oluşturur (ad + alıcı e-postaları).
func (s *Server) createGroup(w http.ResponseWriter, r *http.Request) {
	var req groupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "geçersiz JSON")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeErr(w, http.StatusBadRequest, "grup adı gerekli")
		return
	}
	emails := cleanList(req.Emails)
	if len(emails) == 0 {
		writeErr(w, http.StatusBadRequest, "en az bir alıcı gerekli")
		return
	}
	g, err := s.st.CreateGroup(r.Context(), req.Name, emails)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Grup üyelerini kişi rehberine de kaydet.
	for _, e := range emails {
		_ = s.st.UpsertContact(r.Context(), e, "")
	}
	writeJSON(w, http.StatusCreated, g)
}

// deleteGroup, bir grubu siler (üyeler cascade ile gider; kişiler kalır).
func (s *Server) deleteGroup(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "gid"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "geçersiz id")
		return
	}
	if err := s.st.DeleteGroup(r.Context(), id); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
