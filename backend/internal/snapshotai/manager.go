// Package snapshotai implements the Snapshot Intelligence Engine for Pi mode.
//
// Instead of real-time YOLO inference (which is not viable on Raspberry Pi),
// this engine captures frames at a configurable interval, sends them to an
// external vision AI API (Claude or OpenAI), and emits structured detection
// events and Discord alerts identical in format to those produced by the YOLO
// pipeline.
//
// Configuration is driven entirely by environment variables:
//
//	SNAPSHOT_AI_ENABLED=true
//	SNAPSHOT_AI_PROVIDER=claude          # or: openai
//	SNAPSHOT_AI_API_KEY=sk-...
//	SNAPSHOT_AI_INTERVAL_SECONDS=15
//	SNAPSHOT_AI_PROMPT="Detect people, vehicles, or packages near a house."
//	SNAPSHOT_AI_THRESHOLD=medium         # low | medium | high
//
// Constraints and expectations:
//   - Not real-time: one request per camera per interval cycle.
//   - Latency depends on the external API round-trip (typically 1–5 seconds).
//   - Not suitable for continuous tracking or sub-second response.
//   - Intended positioning: "smart alerting via AI interpretation, not detection."
package snapshotai

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rtspanda/rtspanda/internal/cameras"
	"github.com/rtspanda/rtspanda/internal/detections"
	"github.com/rtspanda/rtspanda/internal/notifications"
)

// Config drives the Snapshot Intelligence Engine.
type Config struct {
	Enabled         bool
	IntervalSeconds int
	Provider        string // "openai" or "claude"
	APIKey          string
	Prompt          string
	AlertThreshold  string // "low", "medium", or "high"
}

// FrameCaptureFn captures a single JPEG frame from the camera's RTSP stream and
// writes it to outputPath. The caller creates the directory; the function creates
// the file. Returns an error if capture fails.
type FrameCaptureFn func(ctx context.Context, cam cameras.Camera, outputPath string) error

// Manager polls cameras at the configured interval, submits frames to a vision
// AI provider, and emits detection events + Discord alerts on positive results.
type Manager struct {
	dataDir  string
	cameras  *cameras.Service
	repo     *detections.Repository
	notifier *notifications.DiscordNotifier
	capture  FrameCaptureFn
	cfg      Config

	mu       sync.Mutex
	lastSent map[string]time.Time // camera ID → last successful event time
}

// NewManager creates a Manager. Call Start to begin the polling loop.
func NewManager(
	dataDir string,
	cameras *cameras.Service,
	repo *detections.Repository,
	notifier *notifications.DiscordNotifier,
	capture FrameCaptureFn,
	cfg Config,
) *Manager {
	return &Manager{
		dataDir:  dataDir,
		cameras:  cameras,
		repo:     repo,
		notifier: notifier,
		capture:  capture,
		cfg:      cfg,
		lastSent: make(map[string]time.Time),
	}
}

// Start launches the polling loop in a background goroutine and returns
// immediately. The loop stops when ctx is cancelled.
func (m *Manager) Start(ctx context.Context) {
	if !m.cfg.Enabled {
		log.Println("[snapshotai] disabled — set SNAPSHOT_AI_ENABLED=true to activate")
		return
	}

	client, err := newVisionClient(m.cfg.Provider, m.cfg.APIKey)
	if err != nil {
		log.Printf("[snapshotai] invalid provider: %v — snapshot AI not started", err)
		return
	}

	interval := time.Duration(m.cfg.IntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 30 * time.Second
	}

	log.Printf("[snapshotai] started (provider=%s interval=%s threshold=%s)",
		m.cfg.Provider, interval, m.cfg.AlertThreshold)

	go m.run(ctx, client, interval)
}

func (m *Manager) run(ctx context.Context, client VisionClient, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.tick(ctx, client)
		}
	}
}

func (m *Manager) tick(ctx context.Context, client VisionClient) {
	camList, err := m.cameras.List()
	if err != nil {
		log.Printf("[snapshotai] list cameras: %v", err)
		return
	}
	for _, cam := range camList {
		if !cam.Enabled {
			continue
		}
		m.processCamera(ctx, cam, client)
	}
}

func (m *Manager) processCamera(ctx context.Context, cam cameras.Camera, client VisionClient) {
	snapDir := filepath.Join(m.dataDir, "snapshots", "snapshot_ai", cam.ID)
	if err := os.MkdirAll(snapDir, 0o755); err != nil {
		log.Printf("[snapshotai] mkdir %s: %v", snapDir, err)
		return
	}
	snapPath := filepath.Join(snapDir, fmt.Sprintf("%d.jpg", time.Now().UnixMilli()))

	captureCtx, captureCancel := context.WithTimeout(ctx, 20*time.Second)
	defer captureCancel()

	if err := m.capture(captureCtx, cam, snapPath); err != nil {
		log.Printf("[snapshotai] capture camera %s (%s): %v", cam.ID, cam.Name, err)
		return
	}

	imgBytes, err := os.ReadFile(snapPath)
	if err != nil {
		log.Printf("[snapshotai] read snapshot %s: %v", snapPath, err)
		_ = os.Remove(snapPath)
		return
	}

	apiCtx, apiCancel := context.WithTimeout(ctx, 30*time.Second)
	defer apiCancel()

	result, err := client.Describe(apiCtx, imgBytes, m.cfg.Prompt)
	if err != nil {
		log.Printf("[snapshotai] vision API error camera %s (%s): %v", cam.ID, cam.Name, err)
		_ = os.Remove(snapPath)
		return
	}

	if !result.Detected || !meetsThreshold(result.Confidence, m.cfg.AlertThreshold) {
		_ = os.Remove(snapPath)
		return
	}

	now := time.Now().UTC()
	events, err := m.repo.CreateEvents(
		cam.ID,
		snapPath,
		now,
		0, 0,
		[]detections.Detection{{
			Label:      result.Label,
			Confidence: result.ConfidenceFloat,
			BBox:       detections.BBox{},
		}},
		result.RawJSON,
	)
	if err != nil {
		log.Printf("[snapshotai] persist event camera %s: %v", cam.ID, err)
		return
	}
	if len(events) == 0 {
		return
	}

	log.Printf("[snapshotai] %s → %s (confidence: %s) — %s",
		cam.Name, result.Label, result.Confidence, result.Summary)

	snap := detections.Snapshot{
		CameraID:  cam.ID,
		Timestamp: now,
		Path:      snapPath,
	}
	sourceLabel := fmt.Sprintf("Snapshot AI (%s)", strings.Title(m.cfg.Provider))
	if err := m.notifier.NotifyExternalDetectionEvents(ctx, cam, snap, events, sourceLabel); err != nil {
		log.Printf("[snapshotai] notify camera %s: %v", cam.ID, err)
	}
}

// meetsThreshold returns true when result confidence meets or exceeds the threshold.
func meetsThreshold(confidence, threshold string) bool {
	rank := map[string]int{"low": 1, "medium": 2, "high": 3}
	have, ok1 := rank[strings.ToLower(confidence)]
	need, ok2 := rank[strings.ToLower(threshold)]
	if !ok1 || !ok2 {
		return true // pass unknown values through rather than silently dropping
	}
	return have >= need
}
