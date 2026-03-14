package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

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

type DiscordNotificationService interface {
	SendCameraSnapshot(ctx context.Context, camera cameras.Camera, snapshot detections.Snapshot, includeMotionClip bool) error
	SendCameraRecording(ctx context.Context, camera cameras.Camera, durationSeconds int, format string) error
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

type sendDiscordScreenshotRequest struct {
	IncludeMotion bool `json:"include_motion"`
}

type sendDiscordRecordingRequest struct {
	DurationSeconds *int   `json:"duration_seconds"`
	Format          string `json:"format"`
}

// handleSendDiscordScreenshot: POST /api/v1/cameras/{id}/discord/screenshot
func (s *server) handleSendDiscordScreenshot(w http.ResponseWriter, r *http.Request) {
	if s.detections == nil {
		writeError(w, http.StatusServiceUnavailable, "detection service unavailable")
		return
	}
	if s.notifier == nil {
		writeError(w, http.StatusServiceUnavailable, "discord notifier unavailable")
		return
	}

	camera, ok := s.getCameraByPathID(w, r)
	if !ok {
		return
	}

	includeMotion := false
	if r.Body != nil {
		var req sendDiscordScreenshotRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		includeMotion = req.IncludeMotion
	}

	snapshot, err := s.detections.CaptureTestFrame(camera)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := s.notifier.SendCameraSnapshot(ctx, camera, snapshot, includeMotion); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "sent",
		"camera_id":      camera.ID,
		"snapshot_path":  snapshot.Path,
		"include_motion": includeMotion,
	})
}

// handleSendDiscordRecording: POST /api/v1/cameras/{id}/discord/record
func (s *server) handleSendDiscordRecording(w http.ResponseWriter, r *http.Request) {
	if s.notifier == nil {
		writeError(w, http.StatusServiceUnavailable, "discord notifier unavailable")
		return
	}

	camera, ok := s.getCameraByPathID(w, r)
	if !ok {
		return
	}

	durationSeconds := camera.DiscordRecordDurationSeconds
	format := camera.DiscordRecordFormat
	if r.Body != nil {
		var req sendDiscordRecordingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.DurationSeconds != nil {
			durationSeconds = *req.DurationSeconds
		}
		if req.Format != "" {
			format = req.Format
		}
	}

	if durationSeconds <= 0 {
		writeError(w, http.StatusBadRequest, "duration_seconds must be > 0")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(durationSeconds+45)*time.Second)
	defer cancel()

	if err := s.notifier.SendCameraRecording(ctx, camera, durationSeconds, format); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":           "sent",
		"camera_id":        camera.ID,
		"duration_seconds": durationSeconds,
		"format":           format,
	})
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
