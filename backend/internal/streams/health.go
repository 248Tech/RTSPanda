package streams

import (
	"encoding/json"
	"net/http"
)

// StreamStatus represents the live state of a camera's RTSP stream.
type StreamStatus string

const (
	StatusOnline     StreamStatus = "online"
	StatusOffline    StreamStatus = "offline"
	StatusConnecting StreamStatus = "connecting"

	// hlsBase is the mediamtx HLS endpoint used for liveness probes.
	hlsBase = "http://127.0.0.1:8888"
)

type pathsResponse struct {
	Items []struct {
		Name  string `json:"name"`
		Ready bool   `json:"ready"`
	} `json:"items"`
}

type pathState struct {
	Name  string
	Ready bool
}

// StreamStatus returns the current streaming status for a given camera by
// querying the cached mediamtx path list (one mediamtx call serves all cameras).
func (m *Manager) StreamStatus(cameraID string) StreamStatus {
	if m.disabled {
		return StatusOffline
	}

	paths, err := m.pathCache.get(m.statusClient)
	if err != nil {
		return StatusOffline
	}

	item, ok := paths["camera-"+cameraID]
	if !ok {
		return StatusOffline
	}
	if item.Ready {
		return StatusOnline
	}
	return StatusConnecting
}

// StreamStatusMap returns the status for every requested camera ID in one
// mediamtx round-trip (uses the shared path list cache).
func (m *Manager) StreamStatusMap(cameraIDs []string) map[string]StreamStatus {
	result := make(map[string]StreamStatus, len(cameraIDs))
	for _, id := range cameraIDs {
		result[id] = StatusOffline
	}
	if m.disabled || len(cameraIDs) == 0 {
		return result
	}

	paths, err := m.pathCache.get(m.statusClient)
	if err != nil {
		return result
	}

	for _, id := range cameraIDs {
		item, ok := paths["camera-"+id]
		if !ok {
			result[id] = StatusOffline
			continue
		}
		if item.Ready {
			result[id] = StatusOnline
		} else {
			result[id] = StatusConnecting
		}
	}
	return result
}

func listPaths(client *http.Client) (map[string]pathState, error) {
	resp, err := client.Get(apiBase + "/v3/paths/list")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data pathsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	result := make(map[string]pathState, len(data.Items))
	for _, item := range data.Items {
		result[item.Name] = pathState{Name: item.Name, Ready: item.Ready}
	}
	return result, nil
}

// checkHLSReachable probes the HLS playlist for a camera to verify mediamtx
// is actively serving segments (not just registering the path as ready).
func checkHLSReachable(client *http.Client, cameraID string) bool {
	resp, err := client.Get(hlsBase + "/camera-" + cameraID + "/index.m3u8")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
