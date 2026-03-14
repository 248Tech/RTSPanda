package detections

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-i", camera.RTSPURL,
		"-frames:v", "1",
		"-q:v", "2",
		"-y",
		path,
	}
	output, err := runRTSPFFmpegCommand(ctx, ffmpegBin, args)
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

func runRTSPFFmpegCommand(ctx context.Context, ffmpegBin string, args []string) ([]byte, error) {
	withRWTimeout := append(
		[]string{"-rtsp_transport", "tcp", "-rw_timeout", "5000000"},
		args...,
	)
	output, err := runFFmpegCommand(ctx, ffmpegBin, withRWTimeout)
	if err == nil {
		return output, nil
	}
	if !isMissingFFmpegOption(output, "rw_timeout") {
		return output, err
	}

	withTimeout := append(
		[]string{"-rtsp_transport", "tcp", "-timeout", "5000000"},
		args...,
	)
	output, err = runFFmpegCommand(ctx, ffmpegBin, withTimeout)
	if err == nil {
		return output, nil
	}
	if !isMissingFFmpegOption(output, "timeout") {
		return output, err
	}

	withoutTimeout := append([]string{"-rtsp_transport", "tcp"}, args...)
	return runFFmpegCommand(ctx, ffmpegBin, withoutTimeout)
}

func runFFmpegCommand(ctx context.Context, ffmpegBin string, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, ffmpegBin, args...)
	return cmd.CombinedOutput()
}

func isMissingFFmpegOption(output []byte, option string) bool {
	msg := strings.ToLower(string(output))
	return strings.Contains(msg, "option "+strings.ToLower(option)+" not found")
}
