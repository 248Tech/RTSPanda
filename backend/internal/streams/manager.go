package streams

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/rtspanda/rtspanda/internal/cameras"
)

// Manager coordinates the mediamtx subprocess lifecycle and camera path management.
type Manager struct {
	proc     *proc
	disabled bool

	mu       sync.Mutex
	camMap   map[string]cameras.Camera

	ctx      context.Context
	cancel   context.CancelFunc
	reloadCh chan struct{}
}

// NewManager creates a Manager. If the mediamtx binary is not found, streaming
// is disabled but the app continues to run normally.
func NewManager(dataDir string) *Manager {
	binPath, err := findBinary()
	if err != nil {
		log.Printf("streams: WARNING — %v", err)
		log.Printf("streams: streaming disabled; download mediamtx and place at mediamtx/mediamtx[.exe]")
		return &Manager{disabled: true, camMap: make(map[string]cameras.Camera)}
	}

	configPath := filepath.Join(dataDir, "mediamtx.yml")
	recordDir := filepath.Join(dataDir, "recordings")
	return &Manager{
		proc:     &proc{binPath: binPath, configPath: configPath, recordDir: recordDir},
		camMap:   make(map[string]cameras.Camera),
		reloadCh: make(chan struct{}, 1),
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
	e := cameraEntry{ID: c.ID, RTSPURL: c.RTSPURL, RecordEnabled: c.RecordEnabled}
	if err := apiAddPath(e); err != nil {
		log.Printf("streams: camera %s: add path via API failed — %v (triggering config reload)", c.ID, err)
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
	if err := apiRemovePath(id); err != nil {
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
	_, wasEnabled := m.camMap[c.ID]
	if c.Enabled {
		m.camMap[c.ID] = c
	} else {
		delete(m.camMap, c.ID)
	}
	m.mu.Unlock()

	log.Printf("streams: updating camera id=%s enabled=%v rtsp=%s", c.ID, c.Enabled, c.RTSPURL)
	// Remove old path (ignore error if it wasn't there)
	if wasEnabled {
		_ = apiRemovePath(c.ID)
	}
	if c.Enabled {
		e := cameraEntry{ID: c.ID, RTSPURL: c.RTSPURL, RecordEnabled: c.RecordEnabled}
		if err := apiAddPath(e); err != nil {
			log.Printf("streams: camera %s: update path via API failed — %v (triggering config reload)", c.ID, err)
			m.triggerReload()
		}
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
				done = nil
			} else {
				log.Printf("streams: mediamtx reloaded")
			}

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

func (m *Manager) entries() []cameraEntry {
	result := make([]cameraEntry, 0, len(m.camMap))
	for _, c := range m.camMap {
		result = append(result, cameraEntry{ID: c.ID, RTSPURL: c.RTSPURL, RecordEnabled: c.RecordEnabled})
	}
	return result
}
