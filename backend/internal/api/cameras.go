package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rtspanda/rtspanda/internal/cameras"
)

func (s *server) handleListCameras(w http.ResponseWriter, r *http.Request) {
	list, err := s.cameras.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *server) handleGetCamera(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c, err := s.cameras.Get(id)
	if err != nil {
		if errors.Is(err, cameras.ErrNotFound) {
			writeError(w, http.StatusNotFound, "camera not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

const maxRequestBodyBytes = 256 * 1024 // 256 KB

func (s *server) handleCreateCamera(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	var input cameras.CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	c, err := s.cameras.Create(input)
	if err != nil {
		if errors.Is(err, cameras.ErrInvalid) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.streams.OnCameraAdded(c)
	if s.detections != nil {
		s.detections.OnCameraAdded(c)
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *server) handleUpdateCamera(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	var input cameras.UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	c, err := s.cameras.Update(id, input)
	if err != nil {
		if errors.Is(err, cameras.ErrNotFound) {
			writeError(w, http.StatusNotFound, "camera not found")
			return
		}
		if errors.Is(err, cameras.ErrInvalid) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.streams.OnCameraUpdated(c)
	if s.detections != nil {
		s.detections.OnCameraUpdated(c)
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *server) handleDeleteCamera(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.cameras.Delete(id); err != nil {
		if errors.Is(err, cameras.ErrNotFound) {
			writeError(w, http.StatusNotFound, "camera not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.streams.OnCameraRemoved(id)
	if s.detections != nil {
		s.detections.OnCameraRemoved(id)
	}
	w.WriteHeader(http.StatusNoContent)
}
