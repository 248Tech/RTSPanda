package streams

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// StreamStatus represents the live state of a camera's RTSP stream.
type StreamStatus string

const (
	StatusOnline     StreamStatus = "online"
	StatusOffline    StreamStatus = "offline"
	StatusConnecting StreamStatus = "connecting"
)

type pathsResponse struct {
	Items []struct {
		Name  string `json:"name"`
		Ready bool   `json:"ready"`
	} `json:"items"`
}

// StreamStatus returns the current streaming status for a given camera ID by
// querying the mediamtx internal API.
func (m *Manager) StreamStatus(cameraID string) StreamStatus {
	if m.disabled {
		return StatusOffline
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(apiBase + "/v3/paths/list")
	if err != nil {
		log.Printf("streams: camera %s: status=offline (mediamtx API unreachable: %v)", cameraID, err)
		return StatusOffline
	}
	defer resp.Body.Close()

	var data pathsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("streams: camera %s: status=offline (mediamtx API response invalid: %v)", cameraID, err)
		return StatusOffline
	}

	target := "camera-" + cameraID
	for _, item := range data.Items {
		if item.Name == target {
			if item.Ready {
				return StatusOnline
			}
			log.Printf("streams: camera %s: status=connecting (path %s exists but stream not ready yet)", cameraID, target)
			return StatusConnecting
		}
	}
	log.Printf("streams: camera %s: status=offline (path %s not in mediamtx — add/update may have failed or mediamtx not reloaded)", cameraID, target)
	return StatusOffline
}
