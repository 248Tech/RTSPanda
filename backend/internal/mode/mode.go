// Package mode defines and enforces RTSPanda deployment profiles.
//
// Three modes exist:
//
//   - pi       Raspberry Pi, phones, low-power ARM devices.
//              Viewer + stream relay + snapshot-based cloud AI (Claude/OpenAI).
//              Real-time YOLO inference is explicitly NOT supported.
//
//   - standard Dedicated server (x86, GPU recommended).
//              Full stack: viewer + real-time YOLO detection + all features.
//
//   - viewer   Desktop with no AI dependency.
//              Viewer + stream relay + recordings only.
//
// Set RTSPANDA_MODE to override auto-detection. When unset, ARM architectures
// default to "pi" and x86/other architectures default to "standard".
package mode

import (
	"log"
	"os"
	"runtime"
	"strings"
)

// Mode identifies the RTSPanda deployment profile.
type Mode string

const (
	ModePi       Mode = "pi"
	ModeStandard Mode = "standard"
	ModeViewer   Mode = "viewer"
)

// Detect resolves the deployment mode from RTSPANDA_MODE or CPU architecture.
func Detect() Mode {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("RTSPANDA_MODE")))
	switch raw {
	case string(ModePi):
		return ModePi
	case string(ModeStandard):
		warnIfARM()
		return ModeStandard
	case string(ModeViewer):
		return ModeViewer
	case "":
		arch := runtime.GOARCH
		if arch == "arm" || arch == "arm64" {
			log.Printf(
				"NOTICE: ARM architecture (%s) detected — defaulting to Pi mode "+
					"(viewer + stream relay + snapshot AI only). "+
					"Set RTSPANDA_MODE=standard to override on capable ARM servers.",
				arch,
			)
			return ModePi
		}
		return ModeStandard
	default:
		log.Printf("WARNING: Unknown RTSPANDA_MODE=%q; falling back to standard mode.", raw)
		return ModeStandard
	}
}

// AIInferenceAllowed reports whether real-time YOLO inference is permitted.
// Always false on Pi mode — YOLO on Pi exhausts RAM and CPU on the first camera.
func (m Mode) AIInferenceAllowed() bool {
	return m == ModeStandard
}

// SnapshotAIAllowed reports whether the Snapshot Intelligence Engine may run.
// Only active in Pi mode; standard mode uses the YOLO pipeline instead.
func (m Mode) SnapshotAIAllowed() bool {
	return m == ModePi
}

// LogBanner prints a startup mode summary to the application log.
func (m Mode) LogBanner() {
	switch m {
	case ModePi:
		log.Println("═══════════════════════════════════════════════════════════════")
		log.Println("  RTSPanda — Pi Mode")
		log.Println("  Deployment: viewer + stream relay + snapshot AI alerts")
		log.Println("  ✗ Real-time YOLO inference is DISABLED in Pi mode.")
		log.Println("  ✓ Snapshot AI (Claude / OpenAI) available for interval alerts.")
		log.Println("  → Use RTSPANDA_MODE=standard on a GPU/CPU server for YOLO.")
		log.Println("═══════════════════════════════════════════════════════════════")
	case ModeStandard:
		log.Println("RTSPanda starting in Standard mode (viewer + YOLO detection + full features)")
	case ModeViewer:
		log.Println("RTSPanda starting in Viewer mode (viewer + recordings, AI disabled)")
	}
}

// warnIfARM logs a notice when standard mode is forced on an ARM device.
func warnIfARM() {
	arch := runtime.GOARCH
	if arch == "arm" || arch == "arm64" {
		log.Printf(
			"WARNING: RTSPANDA_MODE=standard on ARM (%s). "+
				"Real-time YOLO inference on Raspberry Pi is NOT supported. "+
				"RAM exhaustion and thermal throttling are expected. "+
				"Set RTSPANDA_MODE=pi to use snapshot AI instead.",
			arch,
		)
	}
}
