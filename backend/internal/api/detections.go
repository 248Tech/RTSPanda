package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/rtspanda/rtspanda/internal/cameras"
	"github.com/rtspanda/rtspanda/internal/detections"
)

type DetectionService interface {
	OnCameraAdded(camera cameras.Camera)
	OnCameraUpdated(camera cameras.Camera)
	OnCameraRemoved(cameraID string)
	CaptureTestFrame(camera cameras.Camera) (detections.Snapshot, error)
	TriggerTestDetection(camera cameras.Camera) (detections.Snapshot, detections.DetectResponse, error)
	ListEvents(limit int, cameraID string) ([]detections.Event, error)
	SnapshotPath(eventID string) (string, error)
	Health() detections.Health
}

// handleDetectionHealth: GET /api/v1/detections/health
func (s *server) handleDetectionHealth(w http.ResponseWriter, r *http.Request) {
	if s.detections == nil {
		writeError(w, http.StatusServiceUnavailable, "detection service unavailable")
		return
	}
	writeJSON(w, http.StatusOK, s.detections.Health())
}

// handleCaptureTestFrame: POST /api/v1/cameras/{id}/detections/test-frame
func (s *server) handleCaptureTestFrame(w http.ResponseWriter, r *http.Request) {
	if s.detections == nil {
		writeError(w, http.StatusServiceUnavailable, "detection service unavailable")
		return
	}

	camera, ok := s.getCameraByPathID(w, r)
	if !ok {
		return
	}

	snapshot, err := s.detections.CaptureTestFrame(camera)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"camera_id":     snapshot.CameraID,
		"timestamp":     snapshot.Timestamp.UTC(),
		"snapshot_path": snapshot.Path,
	})
}

// handleTriggerTestDetection: POST /api/v1/cameras/{id}/detections/test
func (s *server) handleTriggerTestDetection(w http.ResponseWriter, r *http.Request) {
	if s.detections == nil {
		writeError(w, http.StatusServiceUnavailable, "detection service unavailable")
		return
	}

	camera, ok := s.getCameraByPathID(w, r)
	if !ok {
		return
	}

	snapshot, response, err := s.detections.TriggerTestDetection(camera)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	payload := map[string]any{
		"camera_id":  response.CameraID,
		"timestamp":  response.Timestamp,
		"detections": response.Detections,
	}
	if snapshot.Path != "" {
		payload["snapshot_path"] = snapshot.Path
	}
	writeJSON(w, http.StatusOK, payload)
}

// handleListDetectionEvents: GET /api/v1/detection-events
func (s *server) handleListDetectionEvents(w http.ResponseWriter, r *http.Request) {
	if s.detections == nil {
		writeError(w, http.StatusServiceUnavailable, "detection service unavailable")
		return
	}

	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "limit must be a number")
			return
		}
		limit = n
	}
	cameraID := r.URL.Query().Get("camera_id")

	events, err := s.detections.ListEvents(limit, cameraID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, events)
}

// handleGetDetectionSnapshot: GET /api/v1/detection-events/{id}/snapshot
func (s *server) handleGetDetectionSnapshot(w http.ResponseWriter, r *http.Request) {
	if s.detections == nil {
		writeError(w, http.StatusServiceUnavailable, "detection service unavailable")
		return
	}

	eventID := r.PathValue("id")
	path, err := s.detections.SnapshotPath(eventID)
	if err != nil {
		if errors.Is(err, detections.ErrNotFound) || detections.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "detection event not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	http.ServeFile(w, r, path)
}

func (s *server) getCameraByPathID(w http.ResponseWriter, r *http.Request) (cameras.Camera, bool) {
	cameraID := r.PathValue("id")
	camera, err := s.cameras.Get(cameraID)
	if err != nil {
		if errors.Is(err, cameras.ErrNotFound) {
			writeError(w, http.StatusNotFound, "camera not found")
			return cameras.Camera{}, false
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return cameras.Camera{}, false
	}
	return camera, true
}
