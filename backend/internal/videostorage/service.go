package videostorage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	ProviderLocalServer = "local_server"
	ProviderDropbox     = "dropbox"
	ProviderGoogleDrive = "google_drive"
	ProviderOneDrive    = "onedrive"
	ProviderProtonDrive = "proton_drive"
)

type Config struct {
	Enabled      bool
	Provider     string
	LocalPath    string
	RemoteName   string
	RemotePath   string
	SyncInterval time.Duration
	MinFileAge   time.Duration
	RcloneBin    string
}

type ConfigProvider func() (Config, error)

type Service struct {
	recordingsDir string
	configFn      ConfigProvider
}

func NewService(dataDir string, configFn ConfigProvider) *Service {
	return &Service{
		recordingsDir: filepath.Join(dataDir, "recordings"),
		configFn:      configFn,
	}
}

func (s *Service) Start(ctx context.Context) {
	go s.loop(ctx)
}

func (s *Service) loop(ctx context.Context) {
	for {
		delay := 1 * time.Minute

		cfg, err := s.configFn()
		if err != nil {
			log.Printf("video-storage: load config failed: %v", err)
		} else if !cfg.Enabled {
			delay = 1 * time.Minute
		} else {
			if err := s.syncOnce(ctx, cfg); err != nil {
				log.Printf("video-storage: sync failed provider=%s err=%v", cfg.Provider, err)
			}
			if cfg.SyncInterval > 0 {
				delay = cfg.SyncInterval
			}
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (s *Service) syncOnce(ctx context.Context, cfg Config) error {
	if _, err := os.Stat(s.recordingsDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat recordings dir: %w", err)
	}

	switch cfg.Provider {
	case ProviderLocalServer:
		if strings.TrimSpace(cfg.LocalPath) == "" {
			return fmt.Errorf("local_server requires destination path")
		}
		return s.syncToLocalPath(cfg.LocalPath, cfg.MinFileAge)
	case ProviderDropbox, ProviderGoogleDrive, ProviderOneDrive, ProviderProtonDrive:
		if strings.TrimSpace(cfg.RemoteName) == "" {
			return fmt.Errorf("%s requires rclone remote name", cfg.Provider)
		}
		return s.syncWithRclone(ctx, cfg)
	default:
		return fmt.Errorf("unsupported provider %q", cfg.Provider)
	}
}

func (s *Service) syncWithRclone(ctx context.Context, cfg Config) error {
	rcloneBin := strings.TrimSpace(cfg.RcloneBin)
	if rcloneBin == "" {
		rcloneBin = "rclone"
	}

	remoteName := strings.TrimSuffix(strings.TrimSpace(cfg.RemoteName), ":")
	remote := remoteName + ":"
	remotePath := strings.Trim(strings.TrimSpace(cfg.RemotePath), "/\\")
	if remotePath != "" {
		remote += remotePath
	}

	args := []string{
		"copy",
		s.recordingsDir,
		remote,
		"--include",
		"*.mp4",
		"--create-empty-src-dirs",
		"--min-age",
		formatAge(cfg.MinFileAge),
	}
	cmd := exec.CommandContext(ctx, rcloneBin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			return fmt.Errorf("rclone copy failed: %w", err)
		}
		return fmt.Errorf("rclone copy failed: %w: %s", err, msg)
	}

	if out := strings.TrimSpace(string(output)); out != "" {
		log.Printf("video-storage: rclone output provider=%s %s", cfg.Provider, out)
	}
	log.Printf("video-storage: sync complete provider=%s target=%s", cfg.Provider, remote)
	return nil
}

func (s *Service) syncToLocalPath(destination string, minAge time.Duration) error {
	destRoot := filepath.Clean(destination)
	if err := os.MkdirAll(destRoot, 0755); err != nil {
		return fmt.Errorf("create destination path: %w", err)
	}

	now := time.Now()
	copied := 0
	skipped := 0

	err := filepath.WalkDir(s.recordingsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".mp4") {
			return nil
		}

		srcInfo, err := d.Info()
		if err != nil {
			return err
		}

		if minAge > 0 && now.Sub(srcInfo.ModTime()) < minAge {
			skipped++
			return nil
		}

		rel, err := filepath.Rel(s.recordingsDir, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(destRoot, rel)

		if upToDate(dstPath, srcInfo) {
			skipped++
			return nil
		}
		if err := copyFile(path, dstPath, srcInfo); err != nil {
			return err
		}
		copied++
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk recordings: %w", err)
	}

	log.Printf(
		"video-storage: sync complete provider=local_server copied=%d skipped=%d destination=%s",
		copied,
		skipped,
		destRoot,
	)
	return nil
}

func upToDate(dstPath string, srcInfo fs.FileInfo) bool {
	dstInfo, err := os.Stat(dstPath)
	if err != nil {
		return false
	}
	if dstInfo.Size() != srcInfo.Size() {
		return false
	}
	return !dstInfo.ModTime().Before(srcInfo.ModTime())
}

func copyFile(srcPath, dstPath string, srcInfo fs.FileInfo) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("mkdir destination dir: %w", err)
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create destination file: %w", err)
	}

	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		return fmt.Errorf("copy file: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close destination file: %w", closeErr)
	}

	if err := os.Chtimes(dstPath, time.Now(), srcInfo.ModTime()); err != nil {
		return fmt.Errorf("preserve modtime: %w", err)
	}

	return nil
}

func formatAge(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	seconds := int(d.Seconds())
	return fmt.Sprintf("%ds", seconds)
}
