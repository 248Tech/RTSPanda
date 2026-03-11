package api

import (
	"errors"
	"net/http"

	"github.com/rtspanda/rtspanda/internal/recordings"
)

// RecordingService is the interface the API requires.
type RecordingService interface {
	List(cameraID string) ([]recordings.Recording, error)
	FilePath(cameraID, filename string) (string, error)
	Delete(cameraID, filename string) error
}

// handleListRecordings: GET /api/v1/cameras/{id}/recordings
func (s *server) handleListRecordings(w http.ResponseWriter, r *http.Request) {
	cameraID := r.PathValue("id")
	list, err := s.recordingSvc.List(cameraID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleDownloadRecording: GET /api/v1/cameras/{id}/recordings/{filename}
func (s *server) handleDownloadRecording(w http.ResponseWriter, r *http.Request) {
	cameraID := r.PathValue("id")
	filename := r.PathValue("filename")

	path, err := s.recordingSvc.FilePath(cameraID, filename)
	if err != nil {
		if errors.Is(err, recordings.ErrNotFound) {
			writeError(w, http.StatusNotFound, "recording not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	http.ServeFile(w, r, path)
}

// handleDeleteRecording: DELETE /api/v1/cameras/{id}/recordings/{filename}
func (s *server) handleDeleteRecording(w http.ResponseWriter, r *http.Request) {
	cameraID := r.PathValue("id")
	filename := r.PathValue("filename")

	if err := s.recordingSvc.Delete(cameraID, filename); err != nil {
		if errors.Is(err, recordings.ErrNotFound) {
			writeError(w, http.StatusNotFound, "recording not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
