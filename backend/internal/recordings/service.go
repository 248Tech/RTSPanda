// Package recordings lists and manages video files written by mediamtx.
// Files live at {dataDir}/recordings/camera-{id}/*.mp4
package recordings

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var ErrNotFound = errors.New("recording not found")

type Recording struct {
	Filename  string    `json:"filename"`
	CameraID  string    `json:"camera_id"`
	Size      int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

type Service struct {
	dataDir string
}

func NewService(dataDir string) *Service {
	return &Service{dataDir: dataDir}
}

func (s *Service) recordingsDir(cameraID string) string {
	return filepath.Join(s.dataDir, "recordings", "camera-"+cameraID)
}

// List returns all recordings for the given camera, newest first.
func (s *Service) List(cameraID string) ([]Recording, error) {
	dir := s.recordingsDir(cameraID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Recording{}, nil
		}
		return nil, fmt.Errorf("read recordings dir: %w", err)
	}

	result := make([]Recording, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".mp4") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		result = append(result, Recording{
			Filename:  name,
			CameraID:  cameraID,
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
		})
	}

	// Newest first
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

// FilePath returns the absolute path to a recording file, or ErrNotFound.
func (s *Service) FilePath(cameraID, filename string) (string, error) {
	// Sanitise: reject any path traversal
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") || strings.Contains(filename, "..") {
		return "", ErrNotFound
	}
	if !strings.HasSuffix(filename, ".mp4") {
		return "", ErrNotFound
	}
	path := filepath.Join(s.recordingsDir(cameraID), filename)
	if _, err := os.Stat(path); err != nil {
		return "", ErrNotFound
	}
	return path, nil
}

// Delete removes a recording file.
func (s *Service) Delete(cameraID, filename string) error {
	path, err := s.FilePath(cameraID, filename)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete recording: %w", err)
	}
	return nil
}
