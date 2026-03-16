package api

import (
	"net/http"
)

// handleGetStream: GET /api/v1/cameras/{id}/stream
func (s *server) handleGetStream(w http.ResponseWriter, r *http.Request) {
	camera, ok := s.getCameraByPathID(w, r)
	if !ok {
		return
	}
	st := s.streams.StreamStatus(camera.ID)
	writeJSON(w, http.StatusOK, map[string]string{
		"hls_url": "/hls/camera-" + camera.ID + "/index.m3u8",
		"status":  string(st),
	})
}

// handleResetStream: POST /api/v1/cameras/{id}/stream/reset
// Removes and re-adds the mediamtx path for a single camera, forcing a fresh
// RTSP reconnection without restarting the entire mediamtx process.
func (s *server) handleResetStream(w http.ResponseWriter, r *http.Request) {
	camera, ok := s.getCameraByPathID(w, r)
	if !ok {
		return
	}
	if err := s.streams.ResetStream(camera.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reset", "camera_id": camera.ID})
}

// handleResetAllStreams: POST /api/v1/streams/reset
// Triggers a full mediamtx reload, reconnecting every active camera.
func (s *server) handleResetAllStreams(w http.ResponseWriter, r *http.Request) {
	s.streams.ResetAllStreams()
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "reloading"})
}
