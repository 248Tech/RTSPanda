package api

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/rtspanda/rtspanda/internal/cameras"
	"github.com/rtspanda/rtspanda/internal/streams"
)

// CameraService is the interface the API requires of the camera service layer.
type CameraService interface {
	List() ([]cameras.Camera, error)
	Get(id string) (cameras.Camera, error)
	Create(input cameras.CreateInput) (cameras.Camera, error)
	Update(id string, input cameras.UpdateInput) (cameras.Camera, error)
	Delete(id string) error
}

// StreamManager is the interface the API requires of the stream manager.
type StreamManager interface {
	OnCameraAdded(c cameras.Camera)
	OnCameraRemoved(id string)
	OnCameraUpdated(c cameras.Camera)
	StreamStatus(cameraID string) streams.StreamStatus
}

type server struct {
	cameras      CameraService
	streams      StreamManager
	detections   DetectionService
	alertSvc     AlertService
	recordingSvc RecordingService
	logBuf       LogBuffer
}

func NewRouter(cameraSvc CameraService, streamMgr StreamManager, detectionSvc DetectionService, alertSvc AlertService, recordingSvc RecordingService, logBuf LogBuffer) http.Handler {
	s := &server{cameras: cameraSvc, streams: streamMgr, detections: detectionSvc, alertSvc: alertSvc, recordingSvc: recordingSvc, logBuf: logBuf}
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /api/v1/health", handleHealth)

	// Logs (in-memory buffer)
	mux.HandleFunc("GET /api/v1/logs", s.handleLogs)

	// Camera CRUD
	mux.HandleFunc("GET /api/v1/cameras", s.handleListCameras)
	mux.HandleFunc("POST /api/v1/cameras", s.handleCreateCamera)
	mux.HandleFunc("GET /api/v1/cameras/{id}", s.handleGetCamera)
	mux.HandleFunc("PUT /api/v1/cameras/{id}", s.handleUpdateCamera)
	mux.HandleFunc("DELETE /api/v1/cameras/{id}", s.handleDeleteCamera)

	// Stream status
	mux.HandleFunc("GET /api/v1/cameras/{id}/stream", s.handleGetStream)

	// Detection foundation endpoints
	mux.HandleFunc("GET /api/v1/detections/health", s.handleDetectionHealth)
	mux.HandleFunc("POST /api/v1/cameras/{id}/detections/test-frame", s.handleCaptureTestFrame)
	mux.HandleFunc("POST /api/v1/cameras/{id}/detections/test", s.handleTriggerTestDetection)
	mux.HandleFunc("GET /api/v1/detection-events", s.handleListDetectionEvents)
	mux.HandleFunc("GET /api/v1/detection-events/{id}/snapshot", s.handleGetDetectionSnapshot)

	// Recordings (per-camera)
	mux.HandleFunc("GET /api/v1/cameras/{id}/recordings", s.handleListRecordings)
	mux.HandleFunc("GET /api/v1/cameras/{id}/recordings/{filename}", s.handleDownloadRecording)
	mux.HandleFunc("DELETE /api/v1/cameras/{id}/recordings/{filename}", s.handleDeleteRecording)

	// Alert rules (per-camera)
	mux.HandleFunc("GET /api/v1/cameras/{id}/alerts", s.handleListAlertRules)
	mux.HandleFunc("POST /api/v1/cameras/{id}/alerts", s.handleCreateAlertRule)
	mux.HandleFunc("GET /api/v1/cameras/{id}/alert-events", s.handleListCameraAlertEvents)

	// Alert rules (by rule ID)
	mux.HandleFunc("PUT /api/v1/alerts/{id}", s.handleUpdateAlertRule)
	mux.HandleFunc("DELETE /api/v1/alerts/{id}", s.handleDeleteAlertRule)
	mux.HandleFunc("GET /api/v1/alerts/{id}/events", s.handleListAlertEvents)
	mux.HandleFunc("POST /api/v1/alerts/{id}/events", s.handleTriggerAlertEvent)

	// HLS reverse proxy → mediamtx port 8888
	hlsTarget := &url.URL{Scheme: "http", Host: "127.0.0.1:8888"}
	hlsProxy := httputil.NewSingleHostReverseProxy(hlsTarget)
	mux.Handle("/hls/", http.StripPrefix("/hls", hlsProxy))

	// Embedded frontend (SPA): all other routes serve index.html
	staticH, err := staticHandler()
	if err != nil {
		panic("static handler: " + err.Error())
	}
	mux.Handle("/", staticH)

	return mux
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
