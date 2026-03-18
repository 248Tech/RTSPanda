package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/rtspanda/rtspanda/internal/cameras"
	"github.com/rtspanda/rtspanda/internal/settings"
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
	StreamStatusMap(cameraIDs []string) map[string]streams.StreamStatus
	ResetStream(cameraID string) error
	ResetAllStreams()
	IsReady() bool
}

type SettingsService interface {
	Get() (settings.AppSettings, error)
	Update(input settings.UpdateInput) (settings.AppSettings, error)
}

// DBPingerFunc allows the server to health-check the database.
type DBPingerFunc func(ctx context.Context) error

func (f DBPingerFunc) PingContext(ctx context.Context) error { return f(ctx) }

type server struct {
	cameras      CameraService
	streams      StreamManager
	settings     SettingsService
	detections   DetectionService
	notifier     DiscordNotificationService
	alertSvc     AlertService
	recordingSvc RecordingService
	logBuf       LogBuffer
	db           DBPinger
}

func NewRouter(
	cameraSvc CameraService,
	streamMgr StreamManager,
	settingsSvc SettingsService,
	detectionSvc DetectionService,
	notifier DiscordNotificationService,
	alertSvc AlertService,
	recordingSvc RecordingService,
	logBuf LogBuffer,
	db DBPinger,
) http.Handler {
	s := &server{
		cameras:      cameraSvc,
		streams:      streamMgr,
		settings:     settingsSvc,
		detections:   detectionSvc,
		notifier:     notifier,
		alertSvc:     alertSvc,
		recordingSvc: recordingSvc,
		logBuf:       logBuf,
		db:           db,
	}

	// ── API mux (gzip compressed) ────────────────────────────────────────────
	apiMux := http.NewServeMux()

	// Health
	apiMux.HandleFunc("GET /api/v1/health", handleHealth)
	apiMux.HandleFunc("GET /api/v1/health/ready", s.handleHealthReady)

	// System stats + Prometheus metrics
	apiMux.HandleFunc("GET /api/v1/system/stats", handleSystemStats)

	// Logs
	apiMux.HandleFunc("GET /api/v1/logs", s.handleLogs)

	// App settings
	apiMux.HandleFunc("GET /api/v1/settings", s.handleGetSettings)
	apiMux.HandleFunc("PUT /api/v1/settings", s.handleUpdateSettings)

	// Camera CRUD
	apiMux.HandleFunc("GET /api/v1/cameras", s.handleListCameras)
	apiMux.HandleFunc("POST /api/v1/cameras", s.handleCreateCamera)
	apiMux.HandleFunc("GET /api/v1/cameras/{id}", s.handleGetCamera)
	apiMux.HandleFunc("PUT /api/v1/cameras/{id}", s.handleUpdateCamera)
	apiMux.HandleFunc("DELETE /api/v1/cameras/{id}", s.handleDeleteCamera)

	// Batch stream status (one mediamtx call for all cameras)
	apiMux.HandleFunc("GET /api/v1/cameras/stream-status", s.handleStreamStatusAll)

	// Stream status and control
	apiMux.HandleFunc("GET /api/v1/cameras/{id}/stream", s.handleGetStream)
	apiMux.HandleFunc("POST /api/v1/cameras/{id}/stream/reset", s.handleResetStream)
	apiMux.HandleFunc("POST /api/v1/streams/reset", s.handleResetAllStreams)

	// Detection endpoints
	apiMux.HandleFunc("GET /api/v1/detections/health", s.handleDetectionHealth)
	apiMux.HandleFunc("POST /api/v1/cameras/{id}/detections/test-frame", s.handleCaptureTestFrame)
	apiMux.HandleFunc("POST /api/v1/cameras/{id}/detections/test", s.handleTriggerTestDetection)
	apiMux.HandleFunc("POST /api/v1/cameras/{id}/discord/screenshot", s.handleSendDiscordScreenshot)
	apiMux.HandleFunc("POST /api/v1/cameras/{id}/discord/record", s.handleSendDiscordRecording)
	apiMux.HandleFunc("GET /api/v1/detection-events", s.handleListDetectionEvents)
	apiMux.HandleFunc("GET /api/v1/detection-events/{id}/snapshot", s.handleGetDetectionSnapshot)
	apiMux.HandleFunc("POST /api/v1/frigate/events", s.handleFrigateEvent)

	// Recordings
	apiMux.HandleFunc("GET /api/v1/cameras/{id}/recordings", s.handleListRecordings)
	apiMux.HandleFunc("GET /api/v1/cameras/{id}/recordings/{filename}", s.handleDownloadRecording)
	apiMux.HandleFunc("DELETE /api/v1/cameras/{id}/recordings/{filename}", s.handleDeleteRecording)

	// Alert rules
	apiMux.HandleFunc("GET /api/v1/cameras/{id}/alerts", s.handleListAlertRules)
	apiMux.HandleFunc("POST /api/v1/cameras/{id}/alerts", s.handleCreateAlertRule)
	apiMux.HandleFunc("GET /api/v1/cameras/{id}/alert-events", s.handleListCameraAlertEvents)
	apiMux.HandleFunc("PUT /api/v1/alerts/{id}", s.handleUpdateAlertRule)
	apiMux.HandleFunc("DELETE /api/v1/alerts/{id}", s.handleDeleteAlertRule)
	apiMux.HandleFunc("GET /api/v1/alerts/{id}/events", s.handleListAlertEvents)
	apiMux.HandleFunc("POST /api/v1/alerts/{id}/events", s.handleTriggerAlertEvent)

	// ── Root mux (logging + metrics wrap everything) ─────────────────────────
	mux := http.NewServeMux()

	// Prometheus metrics (not gzipped — scrapers don't send Accept-Encoding)
	mux.HandleFunc("GET /metrics", handleMetrics)

	// API routes with gzip
	mux.Handle("/api/", gzipMiddleware(apiMux))

	// HLS reverse proxy → mediamtx port 8888 (binary data, not gzipped)
	hlsTarget := &url.URL{Scheme: "http", Host: "127.0.0.1:8888"}
	hlsProxy := httputil.NewSingleHostReverseProxy(hlsTarget)
	mux.Handle("/hls/", http.StripPrefix("/hls", hlsProxy))

	// Embedded frontend SPA
	staticH, err := staticHandler()
	if err != nil {
		panic("static handler: " + err.Error())
	}
	mux.Handle("/", staticH)

	// Wrap entire mux with logging + bandwidth metrics middleware.
	return loggingMiddleware(mux)
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
