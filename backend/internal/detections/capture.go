package detections

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/rtspanda/rtspanda/internal/cameras"
)

func captureFrame(ffmpegBin string, rootDir string, camera cameras.Camera) (Snapshot, error) {
	ts := time.Now().UTC()
	cameraDir := filepath.Join(rootDir, camera.ID)
	if err := os.MkdirAll(cameraDir, 0755); err != nil {
		return Snapshot{}, fmt.Errorf("create snapshot dir: %w", err)
	}

	name := ts.Format("20060102T150405.000000000Z") + ".jpg"
	path := filepath.Join(cameraDir, name)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		ffmpegBin,
		"-hide_banner",
		"-loglevel", "error",
		"-rtsp_transport", "tcp",
		"-rw_timeout", "5000000",
		"-i", camera.RTSPURL,
		"-frames:v", "1",
		"-q:v", "2",
		"-y",
		path,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.Remove(path)
		return Snapshot{}, fmt.Errorf("ffmpeg capture failed: %w (%s)", err, string(output))
	}

	return Snapshot{
		CameraID:  camera.ID,
		Timestamp: ts,
		Path:      path,
	}, nil
}
