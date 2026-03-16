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

// StreamStatus returns the current streaming status for a given camera ID by
// querying the mediamtx internal API.
func (m *Manager) StreamStatus(cameraID string) StreamStatus {
	if m.disabled {
		return StatusOffline
	}

	paths, err := listPaths(m.statusClient)
	if err != nil {
		return StatusOffline
	}

	target := "camera-" + cameraID
	item, ok := paths[target]
	if !ok {
		return StatusOffline
	}
	if item.Ready {
		return StatusOnline
	}
	return StatusConnecting
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
// This is the second factor in the multi-factor health check.
func checkHLSReachable(client *http.Client, cameraID string) bool {
	resp, err := client.Get(hlsBase + "/camera-" + cameraID + "/index.m3u8")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
