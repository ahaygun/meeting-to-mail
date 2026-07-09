package httpapi

import (
	"net/http"
	"strconv"

	"meeting-to-mail/internal/domain"
)

// uploadChunk, bir ses parçasını alır. Beklenen: raw body (audio/webm),
// sıra numarası ?seq=N query parametresinde.
func (s *Server) uploadChunk(w http.ResponseWriter, r *http.Request) {
	id, ok := s.loadSession(w, r)
	if !ok {
		return
	}

	seqStr := r.URL.Query().Get("seq")
	seq, err := strconv.Atoi(seqStr)
	if err != nil || seq < 0 {
		writeErr(w, http.StatusBadRequest, "geçersiz seq")
		return
	}

	// Aşırı büyük parçalara karşı basit sınır (parça ~15-30 sn; 25MB fazlasıyla yeter).
	r.Body = http.MaxBytesReader(w, r.Body, 25<<20)
	defer r.Body.Close()

	path, size, err := s.disk.SaveChunk(id, seq, r.Body)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "parça kaydedilemedi: "+err.Error())
		return
	}

	chunk := &domain.AudioChunk{SessionID: id, Seq: seq, StoragePath: path, SizeBytes: size}
	if err := s.st.AddChunk(r.Context(), chunk); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"seq": seq, "size_bytes": size})
}
