package streams

import (
	"encoding/json"
	"testing"
)

func TestParseLegacySourceURL(t *testing.T) {
	t.Run("legacy-string", func(t *testing.T) {
		got := parseLegacySourceURL(json.RawMessage(`"rtsp://cam.local/stream"`))
		if got != "rtsp://cam.local/stream" {
			t.Fatalf("expected legacy source URL, got %q", got)
		}
	})

	t.Run("structured-object", func(t *testing.T) {
		got := parseLegacySourceURL(json.RawMessage(`{"type":"rtspSource","id":"x"}`))
		if got != "" {
			t.Fatalf("expected empty URL for structured source, got %q", got)
		}
	})
}

func TestPathItemReady(t *testing.T) {
	t.Run("ready-true", func(t *testing.T) {
		if !pathItemReady(pathListItem{Ready: true}) {
			t.Fatal("expected ready=true to be treated as ready")
		}
	})

	t.Run("available-fallback", func(t *testing.T) {
		available := true
		if !pathItemReady(pathListItem{Available: &available}) {
			t.Fatal("expected available=true to be treated as ready")
		}
	})
}
