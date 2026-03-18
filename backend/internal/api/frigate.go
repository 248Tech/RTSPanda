package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rtspanda/rtspanda/internal/cameras"
	"github.com/rtspanda/rtspanda/internal/detections"
)

type frigateWebhookPayload struct {
	Type       string                 `json:"type"`
	EventType  string                 `json:"event_type"`
	Camera     string                 `json:"camera"`
	ID         string                 `json:"id"`
	Label      string                 `json:"label"`
	TopScore   *float64               `json:"top_score"`
	Score      *float64               `json:"score"`
	Confidence *float64               `json:"confidence"`
	Box        []float64              `json:"box"`
	StartTime  *float64               `json:"start_time"`
	Time       *float64               `json:"time"`
	After      *frigateWebhookPayload `json:"after"`
}

// handleFrigateEvent ingests Frigate webhook payloads and forwards matching events
// to camera-level Discord detection alerts configured for provider=frigate.
func (s *server) handleFrigateEvent(w http.ResponseWriter, r *http.Request) {
	if s.notifier == nil {
		writeError(w, http.StatusServiceUnavailable, "discord notifier unavailable")
		return
	}

	var payload frigateWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	eventType := strings.ToLower(strings.TrimSpace(firstNonEmpty(payload.Type, payload.EventType)))
	if eventType != "" && eventType != "new" {
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status": "ignored",
			"reason": fmt.Sprintf("event type %q is ignored", eventType),
		})
		return
	}

	event := payload
	if payload.After != nil {
		event = *payload.After
	}

	incomingCamera := normalizeFrigateCameraName(firstNonEmpty(event.Camera, payload.Camera))
	if incomingCamera == "" {
		writeError(w, http.StatusBadRequest, "camera is required in frigate payload")
		return
	}

	label := strings.TrimSpace(firstNonEmpty(event.Label, payload.Label))
	if label == "" {
		label = "object"
	}
	confidence := clampConfidence(firstNonNilFloat(event.TopScore, event.Score, event.Confidence, payload.TopScore, payload.Score, payload.Confidence))
	occurredAt := firstNonZeroTime(
		unixFloatToTime(firstNonNilFloat(event.StartTime, event.Time)),
		unixFloatToTime(firstNonNilFloat(payload.StartTime, payload.Time)),
		time.Now().UTC(),
	)

	box := event.Box
	if len(box) == 0 {
		box = payload.Box
	}
	bbox := frigateBoxToBBox(box)

	list, err := s.cameras.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	matches := make([]cameras.Camera, 0)
	for _, camera := range list {
		if !camera.Enabled || !camera.DiscordAlertsEnabled || !camera.DiscordTriggerOnDetection {
			continue
		}
		provider := strings.ToLower(strings.TrimSpace(camera.DiscordDetectionProvider))
		if provider == "" {
			provider = string(cameras.DiscordDetectionProviderYOLO)
		}
		if provider != string(cameras.DiscordDetectionProviderFrigate) {
			continue
		}
		expectedName := normalizeFrigateCameraName(firstNonEmpty(camera.FrigateCameraName, camera.Name))
		if expectedName == "" {
			continue
		}
		if expectedName == incomingCamera {
			matches = append(matches, camera)
		}
	}

	if len(matches) == 0 {
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status":        "ignored",
			"reason":        "no camera configured for this Frigate camera name",
			"camera":        incomingCamera,
			"matched_count": 0,
		})
		return
	}

	eventID := strings.TrimSpace(firstNonEmpty(event.ID, payload.ID))
	snapshotPath, cleanupSnapshot, snapshotErr := downloadFrigateSnapshot(r.Context(), eventID)
	if cleanupSnapshot != nil {
		defer cleanupSnapshot()
	}
	if snapshotErr != nil {
		log.Printf("frigate: snapshot download failed event_id=%s err=%v", eventID, snapshotErr)
	}

	sentCameras := make([]string, 0, len(matches))
	sendErrors := make([]string, 0)
	for _, camera := range matches {
		detectionEvent := detections.Event{
			ID:           eventID,
			CameraID:     camera.ID,
			ObjectLabel:  label,
			Confidence:   confidence,
			BBox:         bbox,
			SnapshotPath: snapshotPath,
			CreatedAt:    occurredAt,
		}
		if detectionEvent.ID == "" {
			detectionEvent.ID = uuid.NewString()
		}

		snapshot := detections.Snapshot{
			CameraID:  camera.ID,
			Timestamp: occurredAt,
			Path:      snapshotPath,
		}

		sendCtx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		err := s.notifier.NotifyExternalDetectionEvents(
			sendCtx,
			camera,
			snapshot,
			[]detections.Event{detectionEvent},
			"Frigate",
		)
		cancel()
		if err != nil {
			sendErrors = append(sendErrors, fmt.Sprintf("%s: %v", camera.Name, err))
			continue
		}
		sentCameras = append(sentCameras, camera.Name)
	}

	statusCode := http.StatusOK
	status := "processed"
	if len(sentCameras) == 0 {
		statusCode = http.StatusBadGateway
		status = "failed"
	}

	writeJSON(w, statusCode, map[string]any{
		"status":        status,
		"camera":        incomingCamera,
		"event_id":      eventID,
		"label":         label,
		"matched_count": len(matches),
		"sent_cameras":  sentCameras,
		"errors":        sendErrors,
	})
}

func downloadFrigateSnapshot(ctx context.Context, eventID string) (string, func(), error) {
	frigateBaseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("FRIGATE_BASE_URL")), "/")
	eventID = strings.TrimSpace(eventID)
	if frigateBaseURL == "" || eventID == "" {
		return "", nil, nil
	}

	snapshotURL := fmt.Sprintf("%s/api/events/%s/snapshot.jpg", frigateBaseURL, url.PathEscape(eventID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, snapshotURL, nil)
	if err != nil {
		return "", nil, fmt.Errorf("build snapshot request: %w", err)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("request snapshot: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", nil, fmt.Errorf("frigate snapshot status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	fileName := "rtspanda-frigate-" + sanitizeFileToken(eventID) + "-" + fmt.Sprintf("%d.jpg", time.Now().UTC().UnixNano())
	filePath := filepath.Join(os.TempDir(), fileName)
	out, err := os.Create(filePath)
	if err != nil {
		return "", nil, fmt.Errorf("create snapshot temp file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = os.Remove(filePath)
		return "", nil, fmt.Errorf("write snapshot file: %w", err)
	}

	return filePath, func() { _ = os.Remove(filePath) }, nil
}

func sanitizeFileToken(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return "event"
	}
	var b strings.Builder
	for _, r := range raw {
		isLetter := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if isLetter || isDigit {
			b.WriteRune(r)
			continue
		}
		b.WriteRune('-')
	}
	value := strings.Trim(strings.ReplaceAll(b.String(), "--", "-"), "-")
	if value == "" {
		return "event"
	}
	return value
}

func normalizeFrigateCameraName(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	normalized = strings.ReplaceAll(normalized, " ", "_")
	normalized = strings.ReplaceAll(normalized, "-", "_")
	return normalized
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstNonNilFloat(values ...*float64) *float64 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func clampConfidence(value *float64) float64 {
	if value == nil {
		return 1
	}
	if *value < 0 {
		return 0
	}
	if *value > 1 {
		return 1
	}
	return *value
}

func unixFloatToTime(value *float64) time.Time {
	if value == nil {
		return time.Time{}
	}
	v := *value
	if v <= 0 {
		return time.Time{}
	}
	if v >= 1_000_000_000_000 {
		v = v / 1000
	}
	sec, frac := math.Modf(v)
	return time.Unix(int64(sec), int64(frac*float64(time.Second))).UTC()
}

func firstNonZeroTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}

func frigateBoxToBBox(box []float64) detections.BBox {
	if len(box) < 4 {
		return detections.BBox{}
	}
	x1 := int(math.Round(box[0]))
	y1 := int(math.Round(box[1]))
	x2 := int(math.Round(box[2]))
	y2 := int(math.Round(box[3]))
	if x1 < 0 {
		x1 = 0
	}
	if y1 < 0 {
		y1 = 0
	}
	w := x2 - x1
	h := y2 - y1
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return detections.BBox{
		X:      x1,
		Y:      y1,
		Width:  w,
		Height: h,
	}
}
