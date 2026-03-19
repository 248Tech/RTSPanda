package detections

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	AIModeLocal  = "local"
	AIModeRemote = "remote"

	defaultLocalDetectorURL = "http://127.0.0.1:8090"
)

type AIConfig struct {
	Mode         string
	AIWorkerURL  string
	DetectorURL  string
}

func NormalizeAIMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case AIModeRemote:
		return AIModeRemote
	case AIModeLocal:
		fallthrough
	default:
		return AIModeLocal
	}
}

func ResolveAIConfig(aiMode, detectorURL, aiWorkerURL string) (AIConfig, error) {
	mode := NormalizeAIMode(aiMode)
	config := AIConfig{
		Mode:        mode,
		AIWorkerURL: strings.TrimRight(strings.TrimSpace(aiWorkerURL), "/"),
	}

	if explicit := strings.TrimRight(strings.TrimSpace(detectorURL), "/"); explicit != "" {
		if err := validateDetectorURL(explicit); err != nil {
			return AIConfig{}, fmt.Errorf("DETECTOR_URL: %w", err)
		}
		config.DetectorURL = explicit
		return config, nil
	}

	if config.AIWorkerURL != "" {
		if err := validateDetectorURL(config.AIWorkerURL); err != nil {
			return AIConfig{}, fmt.Errorf("AI_WORKER_URL: %w", err)
		}
	}

	if mode == AIModeRemote {
		config.DetectorURL = config.AIWorkerURL
		return config, nil
	}

	config.DetectorURL = defaultLocalDetectorURL
	return config, nil
}

func validateDetectorURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse URL %q: %w", raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL %q must use http or https", raw)
	}
	if u.Host == "" {
		return fmt.Errorf("URL %q must include a host", raw)
	}
	return nil
}
