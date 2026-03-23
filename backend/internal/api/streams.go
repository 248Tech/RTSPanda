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
	hlsURL := ""
	// Keep HLS URL available for enabled cameras even when stream status is
	// still initializing, so sourceOnDemand streams can be activated by the
	// first viewer request.
	if camera.Enabled {
		hlsURL = "/hls/camera-" + camera.ID + "/index.m3u8"
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"hls_url": hlsURL,
		"status":  string(st),
	})
}

// handleStreamStatusAll: GET /api/v1/cameras/stream-status
// Returns the stream status for every camera in one mediamtx round-trip.
// Response: { "camera-id": { "status": "online|offline|connecting|initializing", "hls_url": "..." }, ... }
func (s *server) handleStreamStatusAll(w http.ResponseWriter, _ *http.Request) {
	cameras, err := s.cameras.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ids := make([]string, len(cameras))
	for i, c := range cameras {
		ids[i] = c.ID
	}

	statusMap := s.streams.StreamStatusMap(ids)

	type entry struct {
		Status string `json:"status"`
		HLSURL string `json:"hls_url"`
	}
	result := make(map[string]entry, len(cameras))
	for _, c := range cameras {
		st := statusMap[c.ID]
		hlsURL := ""
		if st == "online" {
			hlsURL = "/hls/camera-" + c.ID + "/index.m3u8"
		}
		result[c.ID] = entry{
			Status: string(st),
			HLSURL: hlsURL,
		}
	}
	writeJSON(w, http.StatusOK, result)
}

// handleResetStream: POST /api/v1/cameras/{id}/stream/reset
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
func (s *server) handleResetAllStreams(w http.ResponseWriter, _ *http.Request) {
	s.streams.ResetAllStreams()
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "reloading"})
}
