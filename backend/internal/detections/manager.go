package detections

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rtspanda/rtspanda/internal/cameras"
)

type Config struct {
	FFmpegBin             string
	DetectorURL           string
	AIMode                string
	AIWorkerURL           string
	DefaultSampleInterval time.Duration
	QueueSize             int
	WorkerConcurrency     int
	Notifier              AlertNotifier
}

type AlertNotifier interface {
	NotifyDetectionEvents(ctx context.Context, camera cameras.Camera, snapshot Snapshot, events []Event) error
}

type SnapshotNotifier interface {
	SendCameraSnapshot(ctx context.Context, camera cameras.Camera, snapshot Snapshot, includeMotionClip bool) error
	SendIntervalSnapshot(ctx context.Context, camera cameras.Camera, snapshot Snapshot, includeMotionClip bool) error
}

type Manager struct {
	repo           *Repository
	client         *Client
	ffmpegBin      string
	snapshotRoot   string
	defaultSample  time.Duration
	queue          chan detectJob
	workerCount    int
	samplerEnabled bool
	notifier       AlertNotifier
	aiMode         string
	aiWorkerURL    string

	mu      sync.Mutex
	ctx     context.Context
	cancels map[string]context.CancelFunc
}

type detectJob struct {
	camera   cameras.Camera
	snapshot Snapshot
}

func NewManager(dataDir string, repo *Repository, cfg Config) *Manager {
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 128
	}
	if cfg.WorkerConcurrency <= 0 {
		cfg.WorkerConcurrency = 2
	}
	if cfg.DefaultSampleInterval <= 0 {
		cfg.DefaultSampleInterval = 30 * time.Second
	}
	if cfg.FFmpegBin == "" {
		cfg.FFmpegBin = "ffmpeg"
	}
	aiMode := NormalizeAIMode(cfg.AIMode)
	aiWorkerURL := strings.TrimRight(strings.TrimSpace(cfg.AIWorkerURL), "/")
	if resolved, err := ResolveAIConfig(aiMode, cfg.DetectorURL, aiWorkerURL); err == nil {
		cfg.DetectorURL = resolved.DetectorURL
		aiWorkerURL = resolved.AIWorkerURL
		aiMode = resolved.Mode
	}

	ffmpegBin, ffmpegOK := resolveFFmpeg(cfg.FFmpegBin)
	if !ffmpegOK {
		log.Printf("detections: WARNING - ffmpeg binary %q not found; scheduled sampling is disabled", cfg.FFmpegBin)
	}

	manager := &Manager{
		repo:           repo,
		client:         NewClient(cfg.DetectorURL, 30*time.Second, aiMode),
		ffmpegBin:      ffmpegBin,
		snapshotRoot:   filepath.Join(dataDir, "snapshots", "detections"),
		defaultSample:  cfg.DefaultSampleInterval,
		queue:          make(chan detectJob, cfg.QueueSize),
		workerCount:    cfg.WorkerConcurrency,
		samplerEnabled: ffmpegOK,
		notifier:       cfg.Notifier,
		aiMode:         aiMode,
		aiWorkerURL:    aiWorkerURL,
		cancels:        make(map[string]context.CancelFunc),
	}
	log.Printf(
		"detections: ai_mode=%s ai_worker_url=%s detector_urls=%s",
		manager.aiMode,
		manager.aiWorkerURL,
		strings.Join(manager.client.BaseURLs(), ", "),
	)
	return manager
}

func (m *Manager) Start(ctx context.Context, cameraList []cameras.Camera) error {
	if err := os.MkdirAll(m.snapshotRoot, 0755); err != nil {
		return fmt.Errorf("create detection snapshot root: %w", err)
	}

	m.mu.Lock()
	m.ctx = ctx
	m.mu.Unlock()

	for i := 0; i < m.workerCount; i++ {
		go m.workerLoop(ctx, i+1)
	}

	if !m.samplerEnabled {
		return nil
	}

	for _, camera := range cameraList {
		if m.shouldSample(camera) {
			m.OnCameraAdded(camera)
		}
	}
	return nil
}

func (m *Manager) OnCameraAdded(camera cameras.Camera) {
	if !m.shouldSample(camera) {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ctx == nil || !m.samplerEnabled {
		return
	}
	if _, exists := m.cancels[camera.ID]; exists {
		return
	}
	m.startSamplerLocked(camera)
}

func (m *Manager) OnCameraUpdated(camera cameras.Camera) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ctx == nil || !m.samplerEnabled {
		return
	}

	m.stopSamplerLocked(camera.ID)
	if m.shouldSample(camera) {
		m.startSamplerLocked(camera)
	}
}

func (m *Manager) OnCameraRemoved(cameraID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopSamplerLocked(cameraID)
}

func (m *Manager) CaptureTestFrame(camera cameras.Camera) (Snapshot, error) {
	snap, err := captureFrame(m.ffmpegBin, m.snapshotRoot, camera)
	if err != nil {
		log.Printf("detections: frame capture failed camera=%s err=%v", camera.ID, err)
		return Snapshot{}, err
	}
	log.Printf("detections: frame capture success camera=%s snapshot=%s", camera.ID, snap.Path)
	return snap, nil
}

func (m *Manager) TriggerTestDetection(camera cameras.Camera) (Snapshot, DetectResponse, error) {
	snap, err := m.CaptureTestFrame(camera)
	if err != nil {
		return Snapshot{}, DetectResponse{}, err
	}

	response, raw, err := m.client.DetectFile(camera.ID, snap.Timestamp, snap.Path)
	if err != nil {
		log.Printf("detections: detector request failed camera=%s err=%v raw=%s", camera.ID, err, raw)
		_ = os.Remove(snap.Path)
		return Snapshot{}, DetectResponse{}, err
	}
	log.Printf("detections: detector request success camera=%s detections=%d", camera.ID, len(response.Detections))

	filtered, stats := m.filterDetections(camera, response.Detections, response.ImageWidth, response.ImageHeight)
	log.Printf(
		"detections: filter summary camera=%s raw=%d kept=%d dropped_confidence=%d dropped_label=%d dropped_ignore_zone=%d",
		camera.ID,
		stats.raw,
		stats.kept,
		stats.droppedConfidence,
		stats.droppedLabel,
		stats.droppedIgnoreZone,
	)
	if len(filtered) == 0 {
		_ = os.Remove(snap.Path)
		return Snapshot{}, response, nil
	}

	if _, err := m.repo.CreateEvents(camera.ID, snap.Path, snap.Timestamp, response.ImageWidth, response.ImageHeight, filtered, raw); err != nil {
		log.Printf("detections: event creation failed camera=%s err=%v", camera.ID, err)
		return Snapshot{}, DetectResponse{}, err
	}
	log.Printf("detections: event creation success camera=%s count=%d snapshot=%s", camera.ID, len(filtered), snap.Path)
	response.Detections = filtered
	return snap, response, nil
}

func (m *Manager) ListEvents(limit int, cameraID string) ([]Event, error) {
	return m.repo.ListRecent(limit, cameraID)
}

func (m *Manager) SnapshotPath(eventID string) (string, error) {
	event, err := m.repo.GetByID(eventID)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(event.SnapshotPath); err != nil {
		return "", fmt.Errorf("snapshot missing on disk: %w", err)
	}
	return event.SnapshotPath, nil
}

func (m *Manager) Health() Health {
	detectorHealthy := m.client.Healthy()
	status := "ok"
	if !detectorHealthy || !m.samplerEnabled {
		status = "degraded"
	}
	return Health{
		Status:            status,
		AIMode:            m.aiMode,
		AIWorkerURL:       m.aiWorkerURL,
		DetectorURL:       m.client.BaseURL(),
		DetectorHealthy:   detectorHealthy,
		QueueDepth:        len(m.queue),
		QueueCapacity:     cap(m.queue),
		SamplerEnabled:    m.samplerEnabled,
		WorkerConcurrency: m.workerCount,
	}
}

func (m *Manager) startSamplerLocked(camera cameras.Camera) {
	cameraCtx, cancel := context.WithCancel(m.ctx)
	m.cancels[camera.ID] = cancel

	interval := m.sampleInterval(camera)
	log.Printf("detections: sampler started camera=%s interval=%s", camera.ID, interval)

	go func(c cameras.Camera, sampleInterval time.Duration) {
		ticker := time.NewTicker(sampleInterval)
		defer ticker.Stop()
		var lastIntervalDiscordSent time.Time

		for {
			select {
			case <-cameraCtx.Done():
				log.Printf("detections: sampler stopped camera=%s", c.ID)
				return
			case <-ticker.C:
				snap, err := captureFrame(m.ffmpegBin, m.snapshotRoot, c)
				if err != nil {
					log.Printf("detections: frame capture failed camera=%s err=%v", c.ID, err)
					continue
				}
				log.Printf("detections: frame capture success camera=%s snapshot=%s", c.ID, snap.Path)
				m.maybeSendIntervalDiscordNotification(c, snap, &lastIntervalDiscordSent)

				job := detectJob{camera: c, snapshot: snap}
				select {
				case m.queue <- job:
				default:
					log.Printf("detections: queue full; dropping frame camera=%s snapshot=%s", c.ID, snap.Path)
					_ = os.Remove(snap.Path)
				}
			}
		}
	}(camera, interval)
}

func (m *Manager) stopSamplerLocked(cameraID string) {
	cancel, ok := m.cancels[cameraID]
	if !ok {
		return
	}
	delete(m.cancels, cameraID)
	cancel()
}

func (m *Manager) sampleInterval(camera cameras.Camera) time.Duration {
	if camera.DetectionSampleSeconds != nil && *camera.DetectionSampleSeconds > 0 {
		return time.Duration(*camera.DetectionSampleSeconds) * time.Second
	}
	return m.defaultSample
}

func (m *Manager) workerLoop(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-m.queue:
			m.handleJob(workerID, job)
		}
	}
}

func (m *Manager) handleJob(workerID int, job detectJob) {
	log.Printf("detections: detector request start worker=%d camera=%s snapshot=%s detector=%s", workerID, job.camera.ID, job.snapshot.Path, m.client.BaseURL())
	response, raw, err := m.client.DetectFile(job.camera.ID, job.snapshot.Timestamp, job.snapshot.Path)
	if err != nil {
		log.Printf("detections: detector request failed worker=%d camera=%s err=%v raw=%s", workerID, job.camera.ID, err, raw)
		_ = os.Remove(job.snapshot.Path)
		return
	}
	log.Printf("detections: detector request success worker=%d camera=%s detections=%d labels=%s", workerID, job.camera.ID, len(response.Detections), summarizeDetectionLabels(response.Detections))

	if len(response.Detections) == 0 {
		_ = os.Remove(job.snapshot.Path)
		return
	}

	filtered, stats := m.filterDetections(job.camera, response.Detections, response.ImageWidth, response.ImageHeight)
	log.Printf(
		"detections: filter summary worker=%d camera=%s raw=%d kept=%d dropped_confidence=%d dropped_label=%d dropped_ignore_zone=%d",
		workerID,
		job.camera.ID,
		stats.raw,
		stats.kept,
		stats.droppedConfidence,
		stats.droppedLabel,
		stats.droppedIgnoreZone,
	)
	if len(filtered) == 0 {
		_ = os.Remove(job.snapshot.Path)
		return
	}

	events, err := m.repo.CreateEvents(
		job.camera.ID,
		job.snapshot.Path,
		job.snapshot.Timestamp,
		response.ImageWidth,
		response.ImageHeight,
		filtered,
		raw,
	)
	if err != nil {
		log.Printf("detections: event creation failed worker=%d camera=%s err=%v", workerID, job.camera.ID, err)
		return
	}
	log.Printf("detections: event creation success worker=%d camera=%s count=%d snapshot=%s", workerID, job.camera.ID, len(events), job.snapshot.Path)

	if m.notifier != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := m.notifier.NotifyDetectionEvents(ctx, job.camera, job.snapshot, events); err != nil {
			log.Printf("detections: notifier failed worker=%d camera=%s err=%v", workerID, job.camera.ID, err)
		}
	}
}

func (m *Manager) maybeSendIntervalDiscordNotification(camera cameras.Camera, snapshot Snapshot, lastSent *time.Time) {
	if m.notifier == nil || !camera.DiscordAlertsEnabled || !camera.DiscordTriggerOnInterval {
		return
	}

	notifier, ok := m.notifier.(SnapshotNotifier)
	if !ok {
		return
	}

	intervalSeconds := camera.DiscordScreenshotIntervalSeconds
	if intervalSeconds <= 0 {
		intervalSeconds = 300
	}
	interval := time.Duration(intervalSeconds) * time.Second

	now := time.Now()
	if !lastSent.IsZero() && now.Sub(*lastSent) < interval {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	if err := notifier.SendIntervalSnapshot(ctx, camera, snapshot, camera.DiscordIncludeMotionClip); err != nil {
		log.Printf("detections: interval discord alert failed camera=%s err=%v", camera.ID, err)
		return
	}

	*lastSent = now
	log.Printf("detections: interval discord alert sent camera=%s interval=%s snapshot=%s", camera.ID, interval, snapshot.Path)
}

func summarizeDetectionLabels(detections []Detection) string {
	if len(detections) == 0 {
		return "none"
	}
	max := len(detections)
	if max > 5 {
		max = 5
	}
	parts := make([]string, 0, max)
	for i := 0; i < max; i++ {
		d := detections[i]
		parts = append(parts, fmt.Sprintf("%s(%.0f%%)", d.Label, d.Confidence*100))
	}
	if len(detections) > max {
		parts = append(parts, "...")
	}
	return strings.Join(parts, ", ")
}

func resolveFFmpeg(configured string) (string, bool) {
	if configured == "" {
		return "", false
	}
	if filepath.IsAbs(configured) {
		if _, err := os.Stat(configured); err == nil {
			return configured, true
		}
		return configured, false
	}
	if p, err := exec.LookPath(configured); err == nil {
		return p, true
	}
	return configured, false
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func (m *Manager) shouldSample(camera cameras.Camera) bool {
	return camera.Enabled && camera.TrackingEnabled
}

type detectionFilterStats struct {
	raw               int
	kept              int
	droppedConfidence int
	droppedLabel      int
	droppedIgnoreZone int
}

func (m *Manager) filterDetections(camera cameras.Camera, detections []Detection, frameWidth int, frameHeight int) ([]Detection, detectionFilterStats) {
	stats := detectionFilterStats{raw: len(detections)}
	if len(detections) == 0 {
		return []Detection{}, stats
	}

	minConfidence := camera.TrackingMinConfidence
	if minConfidence <= 0 {
		minConfidence = 0.25
	}

	labelFilter := make(map[string]struct{}, len(camera.TrackingLabels))
	for _, label := range camera.TrackingLabels {
		trimmed := strings.ToLower(strings.TrimSpace(label))
		if trimmed == "" {
			continue
		}
		labelFilter[trimmed] = struct{}{}
	}

	filtered := make([]Detection, 0, len(detections))
	for _, detection := range detections {
		if detection.Confidence < minConfidence {
			stats.droppedConfidence++
			continue
		}

		if len(labelFilter) > 0 {
			label := strings.ToLower(strings.TrimSpace(detection.Label))
			if _, ok := labelFilter[label]; !ok {
				stats.droppedLabel++
				continue
			}
		}

		if shouldIgnoreByPolygon(camera.TrackingIgnorePolygons, detection, frameWidth, frameHeight) {
			stats.droppedIgnoreZone++
			continue
		}

		filtered = append(filtered, detection)
	}
	stats.kept = len(filtered)
	return filtered, stats
}
