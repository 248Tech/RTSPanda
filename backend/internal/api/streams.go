package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/rtspanda/rtspanda/internal/cameras"
)

func (s *server) handleGetStream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	_, err := s.cameras.Get(id)
	if err != nil {
		if errors.Is(err, cameras.ErrNotFound) {
			writeError(w, http.StatusNotFound, "camera not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	status := s.streams.StreamStatus(id)
	writeJSON(w, http.StatusOK, map[string]string{
		"hls_url": fmt.Sprintf("/hls/camera-%s/index.m3u8", id),
		"status":  string(status),
	})
}
