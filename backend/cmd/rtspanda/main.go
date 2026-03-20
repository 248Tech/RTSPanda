package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/rtspanda/rtspanda/internal/alerts"
	"github.com/rtspanda/rtspanda/internal/api"
	"github.com/rtspanda/rtspanda/internal/cameras"
	"github.com/rtspanda/rtspanda/internal/db"
	"github.com/rtspanda/rtspanda/internal/detections"
	"github.com/rtspanda/rtspanda/internal/logs"
	"github.com/rtspanda/rtspanda/internal/mode"
	"github.com/rtspanda/rtspanda/internal/notifications"
	"github.com/rtspanda/rtspanda/internal/recordings"
	"github.com/rtspanda/rtspanda/internal/settings"
	"github.com/rtspanda/rtspanda/internal/snapshotai"
	"github.com/rtspanda/rtspanda/internal/streams"
	"github.com/rtspanda/rtspanda/internal/videostorage"
)

func main() {
	// ── Deployment mode ───────────────────────────────────────────────────────
	// Resolved first so that all subsystem setup can respect mode constraints.
	// Set RTSPANDA_MODE=pi|standard|viewer to override auto-detection.
	deployMode := mode.Detect()
	deployMode.LogBanner()

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Database
	database, err := db.Open(dataDir)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	// Camera service
	cameraRepo := cameras.NewRepository(database.DB)
	cameraSvc := cameras.NewService(cameraRepo)

	// Alert service
	alertRepo := alerts.NewRepository(database.DB)
	alertSvc := alerts.NewService(alertRepo)

	// App settings service
	settingsRepo := settings.NewRepository(database.DB)
	settingsSvc := settings.NewService(settingsRepo)

	// Recording service
	recordingSvc := recordings.NewService(dataDir)

	// External video storage sync service
	videoStorageSvc := videostorage.NewService(dataDir, func() (videostorage.Config, error) {
		cfg, err := settingsSvc.GetVideoStorageConfig()
		if err != nil {
			return videostorage.Config{}, err
		}
		return videostorage.Config{
			Enabled:      cfg.Enabled,
			Provider:     cfg.Provider,
			LocalPath:    cfg.LocalPath,
			RemoteName:   cfg.RemoteName,
			RemotePath:   cfg.RemotePath,
			SyncInterval: time.Duration(cfg.SyncIntervalSec) * time.Second,
			MinFileAge:   time.Duration(cfg.MinFileAgeSec) * time.Second,
			RcloneBin:    envOrDefault("RCLONE_BIN", "rclone"),
		}, nil
	})

	// Detection event repository (shared by YOLO pipeline and snapshot AI)
	detectionRepo := detections.NewRepository(database.DB)
	ffmpegBin := envOrDefault("FFMPEG_BIN", "ffmpeg")

	discordNotifier := notifications.NewDiscordNotifier(15*time.Second, notifications.DiscordNotifierConfig{
		FFmpegBin:          ffmpegBin,
		MotionClipDuration: time.Duration(envIntOrDefault("DISCORD_MOTION_CLIP_SECONDS", 4)) * time.Second,
		OpenAIConfigProvider: func() (notifications.OpenAIConfig, error) {
			cfg, err := settingsSvc.GetOpenAIConfig()
			if err != nil {
				return notifications.OpenAIConfig{}, err
			}
			return notifications.OpenAIConfig{
				Enabled: cfg.Enabled,
				APIKey:  cfg.APIKey,
				Model:   cfg.Model,
			}, nil
		},
	})

	// Log buffer for Settings → Logs page
	logBuf := logs.NewBuffer(1000)
	log.SetOutput(io.MultiWriter(os.Stdout, logBuf.Writer()))

	// Stream manager (gracefully disabled if mediamtx binary not found)
	streamMgr := streams.NewManager(dataDir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load all enabled cameras and start mediamtx
	cameraList, err := cameraSvc.List()
	if err != nil {
		log.Fatalf("load cameras: %v", err)
	}
	if err := streamMgr.Start(ctx, cameraList); err != nil {
		log.Fatalf("start streams: %v", err)
	}

	// ── YOLO detection manager ─────────────────────────────────────────────────
	// Standard mode only. Pi mode uses snapshot AI instead.
	var detectionMgr *detections.Manager
	if deployMode.AIInferenceAllowed() {
		aiConfig, err := detections.ResolveAIConfig(
			os.Getenv("AI_MODE"),
			os.Getenv("DETECTOR_URL"),
			os.Getenv("AI_WORKER_URL"),
		)
		if err != nil {
			log.Fatalf("resolve AI detector config: %v", err)
		}
		detectionMgr = detections.NewManager(dataDir, detectionRepo, detections.Config{
			FFmpegBin:             ffmpegBin,
			AIMode:                aiConfig.Mode,
			AIWorkerURL:           aiConfig.AIWorkerURL,
			DetectorURL:           aiConfig.DetectorURL,
			DefaultSampleInterval: time.Duration(envIntOrDefault("DETECTION_SAMPLE_INTERVAL_SECONDS", 30)) * time.Second,
			QueueSize:             envIntOrDefault("DETECTION_QUEUE_SIZE", 128),
			WorkerConcurrency:     envIntOrDefault("DETECTION_WORKERS", 2),
			Notifier:              discordNotifier,
		})
		if err := detectionMgr.Start(ctx, cameraList); err != nil {
			log.Fatalf("start detections: %v", err)
		}
	} else {
		// In Pi/Viewer mode, create a degraded (no-op) detection manager so the
		// API router has a valid handle for health and event endpoints.
		if deployMode == mode.ModePi {
			log.Println("AI inference disabled: Raspberry Pi does not support real-time YOLO inference.")
			log.Println("Use snapshot AI mode instead (SNAPSHOT_AI_ENABLED=true).")
		}
		detectionMgr = detections.NewManager(dataDir, detectionRepo, detections.Config{
			FFmpegBin:             ffmpegBin,
			AIMode:                detections.AIModeRemote,
			DetectorURL:           "", // no detector — degrades gracefully
			DefaultSampleInterval: time.Duration(envIntOrDefault("DETECTION_SAMPLE_INTERVAL_SECONDS", 30)) * time.Second,
			QueueSize:             32,
			WorkerConcurrency:     1,
			Notifier:              discordNotifier,
		})
		if err := detectionMgr.Start(ctx, cameraList); err != nil {
			log.Fatalf("start detections (degraded): %v", err)
		}
	}

	// ── Snapshot Intelligence Engine ───────────────────────────────────────────
	// Pi mode only. Captures frames at intervals and sends them to Claude or
	// OpenAI for structured interpretation. Emits events identical to YOLO output.
	if deployMode.SnapshotAIAllowed() {
		snapCfg := snapshotai.Config{
			Enabled:         envBoolOrDefault("SNAPSHOT_AI_ENABLED", false),
			IntervalSeconds: envIntOrDefault("SNAPSHOT_AI_INTERVAL_SECONDS", 30),
			Provider:        envOrDefault("SNAPSHOT_AI_PROVIDER", "claude"),
			APIKey:          os.Getenv("SNAPSHOT_AI_API_KEY"),
			Prompt:          envOrDefault("SNAPSHOT_AI_PROMPT", "Detect people, vehicles, packages, or other notable activity near a building or property."),
			AlertThreshold:  envOrDefault("SNAPSHOT_AI_THRESHOLD", "medium"),
		}
		snapMgr := snapshotai.NewManager(
			dataDir,
			cameraSvc,
			detectionRepo,
			discordNotifier,
			func(ctx context.Context, cam cameras.Camera, outputPath string) error {
				return detections.CaptureFrameToPath(ctx, ffmpegBin, cam.RTSPURL, outputPath)
			},
			snapCfg,
		)
		snapMgr.Start(ctx)
	}

	videoStorageSvc.Start(ctx)

	// HTTP server
	router := api.NewRouter(cameraSvc, streamMgr, settingsSvc, detectionMgr, discordNotifier, alertSvc, recordingSvc, logBuf, api.DBPingerFunc(database.PingContext))
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("RTSPanda listening on :%s (data: %s, mode: %s)", port, dataDir, deployMode)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	cancel()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
	log.Println("done")
}

func envOrDefault(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func envIntOrDefault(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("invalid %s=%q; using default %d", key, v, fallback)
		return fallback
	}
	return n
}

func envBoolOrDefault(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		log.Printf("invalid %s=%q; using default %v", key, v, fallback)
		return fallback
	}
	return b
}
