package detections

import "testing"

func TestResolveAIConfigDefaultsToLocal(t *testing.T) {
	cfg, err := ResolveAIConfig("", "", "")
	if err != nil {
		t.Fatalf("ResolveAIConfig() error = %v", err)
	}

	if cfg.Mode != AIModeLocal {
		t.Fatalf("Mode = %q, want %q", cfg.Mode, AIModeLocal)
	}
	if cfg.DetectorURL != defaultLocalDetectorURL {
		t.Fatalf("DetectorURL = %q, want %q", cfg.DetectorURL, defaultLocalDetectorURL)
	}
}

func TestResolveAIConfigUsesExplicitDetectorOverride(t *testing.T) {
	cfg, err := ResolveAIConfig(AIModeRemote, "http://detector.example:8090", "http://worker.example:8090")
	if err != nil {
		t.Fatalf("ResolveAIConfig() error = %v", err)
	}

	if cfg.DetectorURL != "http://detector.example:8090" {
		t.Fatalf("DetectorURL = %q, want explicit override", cfg.DetectorURL)
	}
}

func TestResolveAIConfigAllowsRemoteModeWithoutWorkerURL(t *testing.T) {
	cfg, err := ResolveAIConfig(AIModeRemote, "", "")
	if err != nil {
		t.Fatalf("ResolveAIConfig() error = %v", err)
	}

	if cfg.Mode != AIModeRemote {
		t.Fatalf("Mode = %q, want %q", cfg.Mode, AIModeRemote)
	}
	if cfg.DetectorURL != "" {
		t.Fatalf("DetectorURL = %q, want empty remote detector", cfg.DetectorURL)
	}
}

func TestResolveAIConfigRejectsInvalidWorkerURL(t *testing.T) {
	if _, err := ResolveAIConfig(AIModeRemote, "", "tcp://worker.example"); err == nil {
		t.Fatal("ResolveAIConfig() error = nil, want invalid URL error")
	}
}

func TestBuildDetectorURLsKeepsRemoteModeStrict(t *testing.T) {
	urls := buildDetectorURLs("http://worker.example:8090", AIModeRemote)
	if len(urls) != 1 || urls[0] != "http://worker.example:8090" {
		t.Fatalf("buildDetectorURLs(remote) = %v, want only explicit remote URL", urls)
	}
}

func TestBuildDetectorURLsAddsLocalFallbacks(t *testing.T) {
	urls := buildDetectorURLs("http://127.0.0.1:8090", AIModeLocal)
	if len(urls) < 2 {
		t.Fatalf("buildDetectorURLs(local) = %v, want fallback aliases", urls)
	}
	if urls[0] != "http://127.0.0.1:8090" {
		t.Fatalf("first local detector URL = %q, want primary local URL", urls[0])
	}
}
