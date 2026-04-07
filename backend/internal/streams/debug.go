package streams

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

const maxMediamtxDebugBody = 512 << 10
const maxHLSDebugBody = 4096

// StreamDebug aggregates mediamtx runtime/config and an HLS probe for one camera.
// Used by GET /api/v1/cameras/{id}/stream/debug when diagnosing manifest 404s.
type StreamDebug struct {
	StreamingDisabled    bool `json:"streaming_disabled"`
	MediamtxAPIReachable bool `json:"mediamtx_api_reachable"`

	CameraID      string `json:"camera_id"`
	PathName      string `json:"path_name"`
	CameraEnabled bool   `json:"camera_enabled"`

	AppStreamStatus string `json:"app_stream_status"`
	HLSURL          string `json:"hls_url"`

	Profile json.RawMessage `json:"profile"`

	PathListSnapshot json.RawMessage `json:"path_list_snapshot,omitempty"`
	PathListError    string          `json:"path_list_error,omitempty"`

	MediamtxPathsGet           json.RawMessage `json:"mediamtx_paths_get,omitempty"`
	MediamtxPathsGetHTTPStatus int             `json:"mediamtx_paths_get_http_status"`
	MediamtxPathsGetError      string          `json:"mediamtx_paths_get_error,omitempty"`

	MediamtxConfigGet           json.RawMessage `json:"mediamtx_config_get,omitempty"`
	MediamtxConfigGetHTTPStatus int             `json:"mediamtx_config_get_http_status"`
	MediamtxConfigGetError      string          `json:"mediamtx_config_get_error,omitempty"`

	HLSProbeHTTPStatus int    `json:"hls_probe_http_status"`
	HLSProbeError      string `json:"hls_probe_error,omitempty"`
	HLSBodyPrefix      string `json:"hls_body_prefix,omitempty"`
}

// StreamDebug returns a fresh snapshot (bypasses path list cache for list + mediamtx GETs).
func (m *Manager) StreamDebug(cameraID string, cameraEnabled bool) StreamDebug {
	pathName := "camera-" + cameraID
	out := StreamDebug{
		CameraID:      cameraID,
		PathName:      pathName,
		CameraEnabled: cameraEnabled,
	}
	if cameraEnabled {
		out.HLSURL = "/hls/" + pathName + "/index.m3u8"
	}

	p := currentStreamProfile()
	prof, _ := json.Marshal(map[string]any{
		"hls_always_remux":             p.HLSAlwaysRemux,
		"hls_segment_count":            p.HLSSegmentCount,
		"hls_segment_duration":         p.HLSSegmentDuration.String(),
		"hls_part_duration":            p.HLSPartDuration.String(),
		"hls_variant":                  p.HLSVariant,
		"source_on_demand":             p.SourceOnDemand,
		"source_on_demand_close_after": p.SourceOnDemandCloseAfter.String(),
		"log_level":                    p.LogLevel,
		"rtsp_transport":               p.RTSPTransport,
	})
	out.Profile = prof

	if m.disabled {
		out.StreamingDisabled = true
		out.AppStreamStatus = string(StatusOffline)
		return out
	}

	out.AppStreamStatus = string(m.StreamStatus(cameraID))

	paths, err := listPaths(m.statusClient)
	if err != nil {
		out.PathListError = err.Error()
	} else {
		out.MediamtxAPIReachable = true
		if ps, ok := paths[pathName]; ok {
			out.PathListSnapshot, _ = json.Marshal(map[string]any{
				"name":   ps.Name,
				"ready":  ps.Ready,
				"source": ps.Source,
			})
		}
	}

	esc := url.PathEscape(pathName)
	fillMediamtxGET := func(apiPath string, target *json.RawMessage, httpSt *int, errStr *string) {
		st, body, reqErr := mediamtxHTTPGet(apiPath)
		*httpSt = st
		if reqErr != nil {
			*errStr = reqErr.Error()
			return
		}
		if st != http.StatusOK {
			*errStr = strings.TrimSpace(string(truncateBytes(body, 2048)))
			return
		}
		*target = bytesToJSONRaw(body)
	}

	fillMediamtxGET("/v3/paths/get/"+esc, &out.MediamtxPathsGet, &out.MediamtxPathsGetHTTPStatus, &out.MediamtxPathsGetError)
	fillMediamtxGET("/v3/config/paths/get/"+esc, &out.MediamtxConfigGet, &out.MediamtxConfigGetHTTPStatus, &out.MediamtxConfigGetError)

	if !cameraEnabled {
		return out
	}

	hlsReq := hlsBase + "/" + pathName + "/index.m3u8"
	resp, err := m.hlsClient.Get(hlsReq)
	if err != nil {
		out.HLSProbeError = err.Error()
		return out
	}
	defer resp.Body.Close()
	out.HLSProbeHTTPStatus = resp.StatusCode
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxHLSDebugBody))
	if resp.StatusCode != http.StatusOK {
		out.HLSProbeError = strings.TrimSpace(string(truncateBytes(body, 512)))
	}
	if len(body) > 0 {
		out.HLSBodyPrefix = string(truncateRunes(strings.TrimSpace(string(body)), 400))
	}
	return out
}

func mediamtxHTTPGet(path string) (status int, body []byte, err error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(apiBase + path)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxMediamtxDebugBody))
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, b, nil
}

func bytesToJSONRaw(b []byte) json.RawMessage {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return json.RawMessage("null")
	}
	if json.Valid(b) {
		return json.RawMessage(b)
	}
	enc, _ := json.Marshal(string(truncateBytes(b, 8000)))
	return json.RawMessage(enc)
}

func truncateBytes(b []byte, max int) []byte {
	if len(b) <= max {
		return b
	}
	return b[:max]
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	var n int
	for i := range s {
		if n == max {
			return s[:i] + "…"
		}
		n++
	}
	return s
}
