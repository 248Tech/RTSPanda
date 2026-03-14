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
	"github.com/rtspanda/rtspanda/internal/notifications"
	"github.com/rtspanda/rtspanda/internal/recordings"
	"github.com/rtspanda/rtspanda/internal/streams"
)

func main() {
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

	// Recording service
	recordingSvc := recordings.NewService(dataDir)

	// Detection event repository
	detectionRepo := detections.NewRepository(database.DB)
	ffmpegBin := envOrDefault("FFMPEG_BIN", "ffmpeg")
	discordNotifier := notifications.NewDiscordNotifier(15*time.Second, notifications.DiscordNotifierConfig{
		FFmpegBin:          ffmpegBin,
		MotionClipDuration: time.Duration(envIntOrDefault("DISCORD_MOTION_CLIP_SECONDS", 4)) * time.Second,
	})

	// Log buffer for Settings → Logs page (tee log output)
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

	// Async object detection manager (gracefully degraded if ffmpeg/detector unavailable).
	detectionMgr := detections.NewManager(dataDir, detectionRepo, detections.Config{
		FFmpegBin:             ffmpegBin,
		DetectorURL:           envOrDefault("DETECTOR_URL", "http://127.0.0.1:8090"),
		DefaultSampleInterval: time.Duration(envIntOrDefault("DETECTION_SAMPLE_INTERVAL_SECONDS", 30)) * time.Second,
		QueueSize:             envIntOrDefault("DETECTION_QUEUE_SIZE", 128),
		WorkerConcurrency:     envIntOrDefault("DETECTION_WORKERS", 2),
		Notifier:              discordNotifier,
	})
	if err := detectionMgr.Start(ctx, cameraList); err != nil {
		log.Fatalf("start detections: %v", err)
	}

	// HTTP server
	router := api.NewRouter(cameraSvc, streamMgr, detectionMgr, discordNotifier, alertSvc, recordingSvc, logBuf)
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second, // longer for file downloads
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("RTSPanda listening on :%s (data: %s)", port, dataDir)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	cancel() // stops mediamtx watchdog

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
