package streams

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rtspanda/rtspanda/internal/cameras"
)

// Manager coordinates the mediamtx subprocess lifecycle and camera path management.
type Manager struct {
	proc     *proc
	disabled bool

	mu     sync.Mutex
	camMap map[string]cameras.Camera

	ctx      context.Context
	cancel   context.CancelFunc
	reloadCh chan struct{}

	// statusClient queries the mediamtx internal API (factor 1 health check).
	statusClient *http.Client
	// hlsClient probes the HLS endpoint (factor 2 health check).
	hlsClient *http.Client
	// pathCache caches the mediamtx path list so all cameras share one API call.
	pathCache *pathListCache

	keepaliveInterval       time.Duration
	keepaliveUnhealthyGrace time.Duration
	keepaliveRepairBackoff  time.Duration
	keepaliveAPIFailures    int
	pathUnhealthySince      map[string]time.Time
	pathLastRepairAt        map[string]time.Time
	lastReloadAt            time.Time
}

// NewManager creates a Manager. If the mediamtx binary is not found, streaming
// is disabled but the app continues to run normally.
func NewManager(dataDir string) *Manager {
	binPath, err := findBinary()
	if err != nil {
		log.Printf("streams: WARNING — %v", err)
		log.Printf("streams: streaming disabled; download mediamtx and place at mediamtx/mediamtx[.exe]")
		return &Manager{disabled: true, camMap: make(map[string]cameras.Camera), pathCache: newPathListCache(3 * time.Second)}
	}

	configPath := filepath.Join(dataDir, "mediamtx.yml")
	recordDir := filepath.Join(dataDir, "recordings")
	return &Manager{
		proc:                    &proc{binPath: binPath, configPath: configPath, recordDir: recordDir},
		camMap:                  make(map[string]cameras.Camera),
		reloadCh:                make(chan struct{}, 1),
		statusClient:            &http.Client{Timeout: 3 * time.Second},
		hlsClient:               &http.Client{Timeout: 3 * time.Second},
		pathCache:               newPathListCache(3 * time.Second),
		keepaliveInterval:       15 * time.Second,
		keepaliveUnhealthyGrace: 8 * time.Second,
		keepaliveRepairBackoff:  30 * time.Second,
		pathUnhealthySince:      make(map[string]time.Time),
		pathLastRepairAt:        make(map[string]time.Time),
	}
}

// Start loads the initial camera list, writes the mediamtx config, and starts
// the subprocess. Returns immediately after launching the watchdog goroutine.
func (m *Manager) Start(ctx context.Context, cameraList []cameras.Camera) error {
	if m.disabled {
		log.Printf("streams: Start skipped (streaming disabled)")
		return nil
	}

	m.ctx, m.cancel = context.WithCancel(ctx)

	m.mu.Lock()
	for _, c := range cameraList {
		if c.Enabled {
			m.camMap[c.ID] = c
		}
	}
	n := len(m.camMap)
	m.mu.Unlock()

	log.Printf("streams: starting with %d enabled camera(s) (total in list: %d)", n, len(cameraList))
	done, err := m.startProcess()
	if err != nil {
		log.Printf("streams: start failed: %v", err)
		return err
	}

	go m.watchdog(done)
	return nil
}

// IsReady returns true when mediamtx is running and its API is reachable.
func (m *Manager) IsReady() bool {
	if m.disabled {
		return false
	}
	_, err := m.pathCache.get(m.statusClient)
	return err == nil
}

// Stop shuts down mediamtx and the watchdog.
func (m *Manager) Stop() {
	if m.disabled || m.cancel == nil {
		return
	}
	m.cancel()
}

// OnCameraAdded notifies the manager that a camera was added.
func (m *Manager) OnCameraAdded(c cameras.Camera) {
	if m.disabled || !c.Enabled {
		if !c.Enabled {
			log.Printf("streams: camera %s not added (disabled)", c.ID)
		}
		return
	}
	m.mu.Lock()
	m.camMap[c.ID] = c
	m.mu.Unlock()

	log.Printf("streams: adding camera id=%s name=%q rtsp=%s", c.ID, c.Name, c.RTSPURL)
	m.pathCache.invalidate()
	e := cameraEntry{ID: c.ID, RTSPURL: c.RTSPURL, RecordEnabled: c.RecordEnabled}
	if err := m.ensurePath(e); err != nil {
		log.Printf("streams: camera %s: add path via API failed — %v", c.ID, err)
		m.triggerReload()
	}
}

// OnCameraRemoved notifies the manager that a camera was deleted.
func (m *Manager) OnCameraRemoved(id string) {
	if m.disabled {
		return
	}
	m.mu.Lock()
	delete(m.camMap, id)
	m.mu.Unlock()

	log.Printf("streams: removing camera id=%s", id)
	m.pathCache.invalidate()
	if err := apiRemovePath(id); err != nil && !isPathNotFoundErr(err) {
		log.Printf("streams: camera %s: remove path via API failed — %v (triggering config reload)", id, err)
		m.triggerReload()
	}
}

// OnCameraUpdated notifies the manager that a camera was updated.
func (m *Manager) OnCameraUpdated(c cameras.Camera) {
	if m.disabled {
		return
	}
	m.mu.Lock()
	previous, hadPrevious := m.camMap[c.ID]
	wasEnabled := hadPrevious
	if c.Enabled {
		m.camMap[c.ID] = c
	} else {
		delete(m.camMap, c.ID)
	}
	m.mu.Unlock()

	log.Printf("streams: updating camera id=%s enabled=%v rtsp=%s", c.ID, c.Enabled, c.RTSPURL)
	m.pathCache.invalidate()
	if !c.Enabled {
		if wasEnabled {
			if err := apiRemovePath(c.ID); err != nil && !isPathNotFoundErr(err) {
				log.Printf("streams: camera %s: disable/remove path failed — %v", c.ID, err)
				m.triggerReload()
			}
		}
		return
	}

	needsPathRecreate := hadPrevious && wasEnabled && pathConfigChanged(previous, c)
	if needsPathRecreate {
		if err := apiRemovePath(c.ID); err != nil && !isPathNotFoundErr(err) {
			log.Printf("streams: camera %s: pre-update remove path failed — %v", c.ID, err)
			m.triggerReload()
			return
		}
		m.pathCache.invalidate()
	}

	e := cameraEntry{ID: c.ID, RTSPURL: c.RTSPURL, RecordEnabled: c.RecordEnabled}
	if err := m.ensurePath(e); err != nil {
		log.Printf("streams: camera %s: update path via API failed — %v", c.ID, err)
		m.triggerReload()
	}
}

func (m *Manager) triggerReload() {
	select {
	case m.reloadCh <- struct{}{}:
	default: // reload already pending
	}
}

// startProcess writes the config and starts a fresh mediamtx process.
func (m *Manager) startProcess() (<-chan error, error) {
	m.mu.Lock()
	entries := m.entries()
	m.mu.Unlock()

	if err := m.proc.writeConfig(entries); err != nil {
		log.Printf("streams: write mediamtx config failed: %v", err)
		return nil, err
	}
	log.Printf("streams: mediamtx config written for %d path(s)", len(entries))

	done, err := m.proc.start(m.ctx)
	if err != nil {
		log.Printf("streams: mediamtx process start failed: %v", err)
		return nil, err
	}

	// Wait for the mediamtx API to be ready before returning (best-effort).
	go waitForAPI(m.ctx)
	return done, nil
}

// watchdog monitors mediamtx and handles crashes and reload requests.
func (m *Manager) watchdog(done <-chan error) {
	keepaliveTicker := time.NewTicker(m.keepaliveInterval)
	defer keepaliveTicker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			m.proc.stop()
			if done != nil {
				select {
				case <-done:
				case <-time.After(5 * time.Second):
				}
			}
			return

		case <-m.reloadCh:
			m.mu.Lock()
			entries := m.entries()
			m.mu.Unlock()
			if err := m.proc.writeConfig(entries); err != nil {
				log.Printf("streams: reload aborted — config render failed (mediamtx left running): %v", err)
				continue
			}
			m.proc.stop()
			if done != nil {
				select {
				case <-done:
				case <-time.After(3 * time.Second):
					log.Printf("streams: timeout waiting for mediamtx to stop during reload")
				}
			}
			time.Sleep(200 * time.Millisecond)
			var err error
			done, err = m.startProcess()
			if err != nil {
				log.Printf("streams: reload failed: %v", err)
				ch := make(chan error, 1)
				go func() {
					time.Sleep(10 * time.Second)
					ch <- fmt.Errorf("reload retry")
				}()
				done = ch
			} else {
				log.Printf("streams: mediamtx reloaded")
			}

		case <-keepaliveTicker.C:
			m.runKeepalive()

		case err := <-done:
			if m.ctx.Err() != nil {
				return
			}
			log.Printf("streams: mediamtx exited (%v) — restarting in 3s", err)
			time.Sleep(3 * time.Second)
			done, err = m.startProcess()
			if err != nil {
				log.Printf("streams: restart failed: %v — will retry in 10s", err)
				ch := make(chan error, 1)
				go func() {
					time.Sleep(10 * time.Second)
					ch <- fmt.Errorf("restart placeholder")
				}()
				done = ch
			}
		}
	}
}

func (m *Manager) runKeepalive() {
	if m.disabled {
		return
	}

	entries := m.currentEntries()
	if len(entries) == 0 {
		return
	}

	paths, err := m.pathCache.get(m.statusClient)
	if err != nil {
		m.mu.Lock()
		m.keepaliveAPIFailures++
		failures := m.keepaliveAPIFailures
		shouldReload := failures >= 3 && time.Since(m.lastReloadAt) > 2*time.Minute
		if shouldReload {
			m.lastReloadAt = time.Now()
			m.keepaliveAPIFailures = 0
		}
		m.mu.Unlock()
		log.Printf("streams: keepalive API error (%d/3): %v", failures, err)
		if shouldReload {
			log.Printf("streams: keepalive triggering mediamtx reload after repeated API failures")
			m.triggerReload()
		}
		return
	}

	type repairCandidate struct {
		entry  cameraEntry
		reason string
	}
	repairs := make([]repairCandidate, 0)
	now := time.Now()

	m.mu.Lock()
	m.keepaliveAPIFailures = 0

	activeIDs := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		activeIDs[entry.ID] = struct{}{}
		pathName := "camera-" + entry.ID
		path, exists := paths[pathName]
		needsRepair := !exists || !pathMatchesEntry(path, entry)
		if !needsRepair {
			delete(m.pathUnhealthySince, entry.ID)
			continue
		}
		if _, ok := m.pathUnhealthySince[entry.ID]; !ok {
			m.pathUnhealthySince[entry.ID] = now
		}
		if now.Sub(m.pathUnhealthySince[entry.ID]) < m.keepaliveUnhealthyGrace {
			continue
		}
		lastRepairAt := m.pathLastRepairAt[entry.ID]
		if !lastRepairAt.IsZero() && now.Sub(lastRepairAt) < m.keepaliveRepairBackoff {
			continue
		}

		m.pathLastRepairAt[entry.ID] = now
		reason := "missing"
		if exists {
			reason = "source-changed"
		}
		repairs = append(repairs, repairCandidate{entry: entry, reason: reason})
	}

	for cameraID := range m.pathUnhealthySince {
		if _, ok := activeIDs[cameraID]; !ok {
			delete(m.pathUnhealthySince, cameraID)
		}
	}
	for cameraID := range m.pathLastRepairAt {
		if _, ok := activeIDs[cameraID]; !ok {
			delete(m.pathLastRepairAt, cameraID)
		}
	}
	m.mu.Unlock()

	for _, repair := range repairs {
		log.Printf("streams: keepalive repairing camera=%s reason=%s", repair.entry.ID, repair.reason)
		if err := m.ensurePath(repair.entry); err != nil {
			log.Printf("streams: keepalive repair failed camera=%s err=%v", repair.entry.ID, err)
		}
	}
}

// ResetStream removes and re-adds the mediamtx path for a single camera,
// forcing a fresh RTSP reconnection. Falls back to a full reload if the API
// calls fail.
func (m *Manager) ResetStream(cameraID string) error {
	if m.disabled {
		return nil
	}

	m.mu.Lock()
	cam, ok := m.camMap[cameraID]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("camera %s not found or not enabled", cameraID)
	}

	log.Printf("streams: manual reset camera=%s", cameraID)
	_ = apiRemovePath(cameraID)
	time.Sleep(300 * time.Millisecond)

	e := cameraEntry{ID: cam.ID, RTSPURL: cam.RTSPURL, RecordEnabled: cam.RecordEnabled}
	if err := apiAddPath(e); err != nil {
		log.Printf("streams: manual reset re-add failed camera=%s err=%v; triggering full reload", cameraID, err)
		m.triggerReload()
	}

	m.mu.Lock()
	delete(m.pathUnhealthySince, cameraID)
	delete(m.pathLastRepairAt, cameraID)
	m.mu.Unlock()
	return nil
}

// ResetAllStreams triggers a full mediamtx reload, reconnecting every camera.
func (m *Manager) ResetAllStreams() {
	if m.disabled {
		return
	}
	log.Printf("streams: manual reset all streams")
	m.triggerReload()
}

func (m *Manager) currentEntries() []cameraEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.entries()
}

func (m *Manager) entries() []cameraEntry {
	result := make([]cameraEntry, 0, len(m.camMap))
	for _, c := range m.camMap {
		result = append(result, cameraEntry{ID: c.ID, RTSPURL: c.RTSPURL, RecordEnabled: c.RecordEnabled})
	}
	return result
}

func (m *Manager) ensurePath(entry cameraEntry) error {
	paths, err := m.pathCache.get(m.statusClient)
	if err != nil {
		return fmt.Errorf("list paths: %w", err)
	}

	name := "camera-" + entry.ID
	if path, ok := paths[name]; ok {
		if pathMatchesEntry(path, entry) {
			return nil
		}
		if err := apiRemovePath(entry.ID); err != nil && !isPathNotFoundErr(err) {
			return fmt.Errorf("remove outdated path %s: %w", name, err)
		}
	}

	if err := apiAddPath(entry); err != nil {
		return fmt.Errorf("add path %s: %w", name, err)
	}
	log.Printf("streams: mediamtx path added name=%s camera=%s", name, entry.ID)
	m.pathCache.invalidate()
	return nil
}

func pathMatchesEntry(path pathState, entry cameraEntry) bool {
	if strings.TrimSpace(path.Source) == "" {
		return true
	}
	return strings.TrimSpace(path.Source) == strings.TrimSpace(entry.RTSPURL)
}

func pathConfigChanged(previous, current cameras.Camera) bool {
	return strings.TrimSpace(previous.RTSPURL) != strings.TrimSpace(current.RTSPURL) ||
		previous.RecordEnabled != current.RecordEnabled
}

func isPathNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "not found")
}
