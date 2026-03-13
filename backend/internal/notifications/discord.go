package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rtspanda/rtspanda/internal/cameras"
	"github.com/rtspanda/rtspanda/internal/detections"
)

type DiscordNotifier struct {
	httpClient *http.Client

	mu       sync.Mutex
	lastSent map[string]time.Time
}

func NewDiscordNotifier(timeout time.Duration) *DiscordNotifier {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &DiscordNotifier{
		httpClient: &http.Client{Timeout: timeout},
		lastSent:   make(map[string]time.Time),
	}
}

func (n *DiscordNotifier) NotifyDetectionEvents(
	ctx context.Context,
	camera cameras.Camera,
	snapshot detections.Snapshot,
	events []detections.Event,
) error {
	if len(events) == 0 {
		return nil
	}
	if !camera.DiscordAlertsEnabled {
		return nil
	}

	webhookURL := strings.TrimSpace(camera.DiscordWebhookURL)
	if webhookURL == "" {
		return nil
	}

	if !n.allowSend(camera.ID, camera.DiscordCooldownSeconds) {
		return nil
	}

	if err := n.sendWebhook(ctx, webhookURL, camera, snapshot, events); err != nil {
		return err
	}
	n.markSent(camera.ID)
	return nil
}

func (n *DiscordNotifier) allowSend(cameraID string, cooldownSeconds int) bool {
	if cooldownSeconds <= 0 {
		return true
	}

	cooldown := time.Duration(cooldownSeconds) * time.Second
	now := time.Now()

	n.mu.Lock()
	defer n.mu.Unlock()

	last, ok := n.lastSent[cameraID]
	if !ok {
		return true
	}
	return now.Sub(last) >= cooldown
}

func (n *DiscordNotifier) markSent(cameraID string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.lastSent[cameraID] = time.Now()
}

func (n *DiscordNotifier) sendWebhook(
	ctx context.Context,
	webhookURL string,
	camera cameras.Camera,
	snapshot detections.Snapshot,
	events []detections.Event,
) error {
	mention := strings.TrimSpace(camera.DiscordMention)
	attachmentName := filepath.Base(snapshot.Path)
	if attachmentName == "" {
		attachmentName = "snapshot.jpg"
	}

	payload := discordWebhookPayload{
		Content: strings.TrimSpace(strings.Join([]string{
			mention,
			fmt.Sprintf(
				"Camera **%s** detected %d object(s): %s",
				camera.Name,
				len(events),
				summarizeDetections(events),
			),
		}, "\n")),
		Embeds: []discordEmbed{
			{
				Title:       "YOLOv8 Detection Event",
				Description: "Per-camera AI tracking reported one or more detections.",
				Color:       5793266,
				Timestamp:   events[0].CreatedAt.UTC().Format(time.RFC3339),
				Fields:      buildFields(events),
				Footer:      &discordFooter{Text: "RTSPanda"},
				Image:       &discordImage{URL: "attachment://" + attachmentName},
			},
		},
		AllowedMentions: &discordAllowedMentions{Parse: allowedMentionParse(mention)},
	}

	imageFile, err := os.Open(snapshot.Path)
	if err != nil {
		// Fall back to text-only embed when snapshot is missing.
		payload.Embeds[0].Image = nil
		return n.sendJSONWebhook(ctx, webhookURL, payload)
	}
	defer imageFile.Close()

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode discord payload: %w", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("payload_json", string(payloadJSON)); err != nil {
		return fmt.Errorf("write payload_json: %w", err)
	}

	part, err := writer.CreateFormFile("files[0]", attachmentName)
	if err != nil {
		return fmt.Errorf("create multipart file: %w", err)
	}
	if _, err := io.Copy(part, imageFile); err != nil {
		return fmt.Errorf("copy snapshot attachment: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, &body)
	if err != nil {
		return fmt.Errorf("create discord request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("post discord webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord webhook returned %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}

func (n *DiscordNotifier) sendJSONWebhook(
	ctx context.Context,
	webhookURL string,
	payload discordWebhookPayload,
) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode discord payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("create discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("post discord webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord webhook returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func summarizeDetections(events []detections.Event) string {
	if len(events) == 0 {
		return "none"
	}

	parts := make([]string, 0, len(events))
	for _, event := range events {
		parts = append(parts, fmt.Sprintf("%s (%.0f%%)", event.ObjectLabel, event.Confidence*100))
	}
	return strings.Join(parts, ", ")
}

func buildFields(events []detections.Event) []discordField {
	max := len(events)
	if max > 10 {
		max = 10
	}

	fields := make([]discordField, 0, max)
	for i := 0; i < max; i++ {
		event := events[i]
		fields = append(fields, discordField{
			Name:   fmt.Sprintf("Detection %d", i+1),
			Value:  fmt.Sprintf("%s\nConfidence: %.1f%%\nBBox: x=%d y=%d w=%d h=%d", event.ObjectLabel, event.Confidence*100, event.BBox.X, event.BBox.Y, event.BBox.Width, event.BBox.Height),
			Inline: true,
		})
	}
	return fields
}

func allowedMentionParse(mention string) []string {
	if strings.TrimSpace(mention) == "" {
		return []string{}
	}
	return []string{"users", "roles", "everyone"}
}

type discordWebhookPayload struct {
	Content         string                  `json:"content,omitempty"`
	Embeds          []discordEmbed          `json:"embeds,omitempty"`
	AllowedMentions *discordAllowedMentions `json:"allowed_mentions,omitempty"`
}

type discordAllowedMentions struct {
	Parse []string `json:"parse,omitempty"`
}

type discordEmbed struct {
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	Color       int            `json:"color,omitempty"`
	Timestamp   string         `json:"timestamp,omitempty"`
	Fields      []discordField `json:"fields,omitempty"`
	Image       *discordImage  `json:"image,omitempty"`
	Footer      *discordFooter `json:"footer,omitempty"`
}

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type discordImage struct {
	URL string `json:"url"`
}

type discordFooter struct {
	Text string `json:"text"`
}
