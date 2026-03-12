package detections

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/rtspanda/rtspanda/internal/cameras"
)

type Config struct {
	FFmpegBin             string
	DetectorURL           string
	DefaultSampleInterval time.Duration
	QueueSize             int
	WorkerConcurrency     int
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
	if cfg.DetectorURL == "" {
		cfg.DetectorURL = "http://127.0.0.1:8090"
	}

	ffmpegBin, ffmpegOK := resolveFFmpeg(cfg.FFmpegBin)
	if !ffmpegOK {
		log.Printf("detections: WARNING - ffmpeg binary %q not found; scheduled sampling is disabled", cfg.FFmpegBin)
	}

	return &Manager{
		repo:           repo,
		client:         NewClient(cfg.DetectorURL, 30*time.Second),
		ffmpegBin:      ffmpegBin,
		snapshotRoot:   filepath.Join(dataDir, "snapshots", "detections"),
		defaultSample:  cfg.DefaultSampleInterval,
		queue:          make(chan detectJob, cfg.QueueSize),
		workerCount:    cfg.WorkerConcurrency,
		samplerEnabled: ffmpegOK,
		cancels:        make(map[string]context.CancelFunc),
	}
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
		if camera.Enabled {
			m.OnCameraAdded(camera)
		}
	}
	return nil
}

func (m *Manager) OnCameraAdded(camera cameras.Camera) {
	if !camera.Enabled {
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
	if camera.Enabled {
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

	if len(response.Detections) == 0 {
		_ = os.Remove(snap.Path)
		return Snapshot{}, response, nil
	}

	if _, err := m.repo.CreateEvents(camera.ID, snap.Path, snap.Timestamp, response.Detections, raw); err != nil {
		log.Printf("detections: event creation failed camera=%s err=%v", camera.ID, err)
		return Snapshot{}, DetectResponse{}, err
	}
	log.Printf("detections: event creation success camera=%s count=%d snapshot=%s", camera.ID, len(response.Detections), snap.Path)
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
	response, raw, err := m.client.DetectFile(job.camera.ID, job.snapshot.Timestamp, job.snapshot.Path)
	if err != nil {
		log.Printf("detections: detector request failed worker=%d camera=%s err=%v raw=%s", workerID, job.camera.ID, err, raw)
		_ = os.Remove(job.snapshot.Path)
		return
	}
	log.Printf("detections: detector request success worker=%d camera=%s detections=%d", workerID, job.camera.ID, len(response.Detections))

	if len(response.Detections) == 0 {
		_ = os.Remove(job.snapshot.Path)
		return
	}

	events, err := m.repo.CreateEvents(job.camera.ID, job.snapshot.Path, job.snapshot.Timestamp, response.Detections, raw)
	if err != nil {
		log.Printf("detections: event creation failed worker=%d camera=%s err=%v", workerID, job.camera.ID, err)
		return
	}
	log.Printf("detections: event creation success worker=%d camera=%s count=%d snapshot=%s", workerID, job.camera.ID, len(events), job.snapshot.Path)
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
