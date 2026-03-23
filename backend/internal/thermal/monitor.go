package thermal

import (
	"context"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ThermalBand represents host thermal state.
type ThermalBand int

const (
	ThermalBandNormal ThermalBand = iota
	ThermalBandWarm
	ThermalBandHot
	ThermalBandCritical
)

func (b ThermalBand) String() string {
	switch b {
	case ThermalBandWarm:
		return "warm"
	case ThermalBandHot:
		return "hot"
	case ThermalBandCritical:
		return "critical"
	default:
		return "normal"
	}
}

// ThermalBandEvent is published whenever the active band changes.
type ThermalBandEvent struct {
	From         ThermalBand `json:"from"`
	To           ThermalBand `json:"to"`
	TemperatureC float64     `json:"temperature_c"`
	Source       string      `json:"source"`
	At           time.Time   `json:"at"`
}

type Config struct {
	Enabled      bool
	AutoResume   bool
	PollInterval time.Duration
}

type Monitor struct {
	mu sync.RWMutex

	enabled    bool
	autoResume bool

	currentBand ThermalBand
	lastTempC   float64
	lastSource  string

	coolingTarget ThermalBand
	coolingSince  time.Time

	samplingUnavailable bool

	subscribers map[chan ThermalBandEvent]struct{}
}

var (
	globalMu      sync.RWMutex
	globalMonitor *Monitor
)

func Start(ctx context.Context, cfg Config) *Monitor {
	pollInterval := cfg.PollInterval
	if pollInterval <= 0 {
		pollInterval = 15 * time.Second
	}

	m := &Monitor{
		enabled:       cfg.Enabled,
		autoResume:    cfg.AutoResume,
		currentBand:   ThermalBandNormal,
		lastTempC:     0,
		lastSource:    "disabled",
		subscribers:   make(map[chan ThermalBandEvent]struct{}),
		coolingTarget: ThermalBandNormal,
	}
	setGlobal(m)

	if !cfg.Enabled {
		log.Printf("WARN thermal: monitor disabled (THERMAL_MONITOR_ENABLED=false and non-arm64/pi mode)")
		return m
	}

	go m.run(ctx, pollInterval)
	log.Printf("WARN thermal: monitor enabled (auto_resume=%v poll_interval=%s)", cfg.AutoResume, pollInterval)
	return m
}

func GetCurrentBand() ThermalBand {
	m := getGlobal()
	if m == nil {
		return ThermalBandNormal
	}
	return m.getCurrentBand()
}

func Subscribe(ch chan ThermalBandEvent) {
	m := getGlobal()
	if m == nil {
		return
	}
	m.subscribe(ch)
}

func (m *Monitor) getCurrentBand() ThermalBand {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentBand
}

func (m *Monitor) subscribe(ch chan ThermalBandEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribers[ch] = struct{}{}
}

func (m *Monitor) run(ctx context.Context, pollInterval time.Duration) {
	// Immediate first sample, then periodic sampling.
	m.sample()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.sample()
		}
	}
}

func (m *Monitor) sample() {
	tempC, source, ok := readThermalSensors()
	if !ok {
		tempC, source, ok = readCPULoadProxy()
	}
	if !ok {
		m.mu.Lock()
		m.lastSource = "disabled"
		m.lastTempC = 0
		alreadyUnavailable := m.samplingUnavailable
		m.samplingUnavailable = true
		m.mu.Unlock()
		if !alreadyUnavailable {
			log.Printf("WARN thermal: no sensor source found (/sys/class/thermal or load proxy unavailable); monitor in disabled sampling mode")
		}
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.samplingUnavailable {
		log.Printf("WARN thermal: sensor sampling resumed via %s", source)
	}
	m.samplingUnavailable = false

	m.lastTempC = tempC
	m.lastSource = source
	now := time.Now()

	rawBand := classify(tempC)
	if rawBand > m.currentBand {
		prev := m.currentBand
		m.currentBand = rawBand
		m.coolingSince = time.Time{}
		m.coolingTarget = ThermalBandNormal
		m.emitLocked(ThermalBandEvent{
			From:         prev,
			To:           rawBand,
			TemperatureC: tempC,
			Source:       source,
			At:           now,
		})
		return
	}

	if rawBand == m.currentBand {
		m.coolingSince = time.Time{}
		m.coolingTarget = ThermalBandNormal
		return
	}

	// Downshift with required hysteresis windows:
	// Critical->Hot 5m, Hot->Warm 5m, Warm->Normal 3m.
	target := nextLowerBand(m.currentBand)
	if rawBand > target {
		// Not cool enough to consider downshifting.
		m.coolingSince = time.Time{}
		m.coolingTarget = ThermalBandNormal
		return
	}
	if m.coolingTarget != target {
		m.coolingTarget = target
		m.coolingSince = now
		return
	}
	if now.Sub(m.coolingSince) < downshiftDelay(m.currentBand, target) {
		return
	}
	prev := m.currentBand
	m.currentBand = target
	m.coolingSince = time.Time{}
	m.coolingTarget = ThermalBandNormal
	m.emitLocked(ThermalBandEvent{
		From:         prev,
		To:           target,
		TemperatureC: tempC,
		Source:       source,
		At:           now,
	})
}

func (m *Monitor) emitLocked(event ThermalBandEvent) {
	for ch := range m.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

func classify(tempC float64) ThermalBand {
	switch {
	case tempC >= 65:
		return ThermalBandCritical
	case tempC >= 55:
		return ThermalBandHot
	case tempC >= 45:
		return ThermalBandWarm
	default:
		return ThermalBandNormal
	}
}

func nextLowerBand(b ThermalBand) ThermalBand {
	switch b {
	case ThermalBandCritical:
		return ThermalBandHot
	case ThermalBandHot:
		return ThermalBandWarm
	case ThermalBandWarm:
		return ThermalBandNormal
	default:
		return ThermalBandNormal
	}
}

func downshiftDelay(from, to ThermalBand) time.Duration {
	switch {
	case from == ThermalBandCritical && to == ThermalBandHot:
		return 5 * time.Minute
	case from == ThermalBandHot && to == ThermalBandWarm:
		return 5 * time.Minute
	case from == ThermalBandWarm && to == ThermalBandNormal:
		return 3 * time.Minute
	default:
		return 0
	}
}

func readThermalSensors() (float64, string, bool) {
	if runtime.GOOS != "linux" {
		return 0, "", false
	}
	matches, err := filepath.Glob("/sys/class/thermal/thermal_zone*/temp")
	if err != nil || len(matches) == 0 {
		return 0, "", false
	}

	maxTemp := -1.0
	for _, path := range matches {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
		if err != nil {
			continue
		}
		// Most Linux thermal zones report milli-Celsius.
		tempC := v
		if v >= 1000 {
			tempC = v / 1000.0
		}
		if tempC > maxTemp {
			maxTemp = tempC
		}
	}
	if maxTemp < 0 {
		return 0, "", false
	}
	return maxTemp, "thermal_zone", true
}

func readCPULoadProxy() (float64, string, bool) {
	if runtime.GOOS != "linux" {
		return 0, "", false
	}
	raw, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, "", false
	}
	fields := strings.Fields(string(raw))
	if len(fields) < 1 {
		return 0, "", false
	}
	load1, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, "", false
	}
	cores := runtime.NumCPU()
	if cores <= 0 {
		return 0, "", false
	}

	ratio := load1 / float64(cores)
	if ratio < 0 {
		ratio = 0
	}
	ratio = math.Min(ratio, 1.2)
	// Fallback estimate only. Tuned to cross Warm/Hot bands under sustained load.
	tempC := 35.0 + (ratio * 30.0)
	return tempC, "loadavg_proxy", true
}

func setGlobal(m *Monitor) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalMonitor = m
}

func getGlobal() *Monitor {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalMonitor
}
