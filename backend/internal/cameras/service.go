package cameras

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List() ([]Camera, error) {
	return s.repo.List()
}

func (s *Service) Get(id string) (Camera, error) {
	return s.repo.GetByID(id)
}

func (s *Service) Create(input CreateInput) (Camera, error) {
	if input.Name == "" {
		return Camera{}, fmt.Errorf("%w: name is required", ErrInvalid)
	}
	if input.RTSPURL == "" {
		return Camera{}, fmt.Errorf("%w: rtsp_url is required", ErrInvalid)
	}
	if !strings.HasPrefix(strings.TrimSpace(input.RTSPURL), "rtsp://") {
		return Camera{}, fmt.Errorf("%w: rtsp_url must start with rtsp://", ErrInvalid)
	}

	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	recordEnabled := false
	if input.RecordEnabled != nil {
		recordEnabled = *input.RecordEnabled
	}
	if input.DetectionSampleSeconds != nil && *input.DetectionSampleSeconds <= 0 {
		return Camera{}, fmt.Errorf("%w: detection_sample_seconds must be > 0", ErrInvalid)
	}

	trackingEnabled := false
	if input.TrackingEnabled != nil {
		trackingEnabled = *input.TrackingEnabled
	}

	trackingMinConfidence := 0.25
	if input.TrackingMinConfidence != nil {
		trackingMinConfidence = *input.TrackingMinConfidence
	}
	if !isValidConfidence(trackingMinConfidence) {
		return Camera{}, fmt.Errorf("%w: tracking_min_confidence must be between 0 and 1", ErrInvalid)
	}

	trackingLabels := normalizeTrackingLabels(input.TrackingLabels)

	discordAlertsEnabled := false
	if input.DiscordAlertsEnabled != nil {
		discordAlertsEnabled = *input.DiscordAlertsEnabled
	}

	discordWebhookURL := ""
	if input.DiscordWebhookURL != nil {
		discordWebhookURL = strings.TrimSpace(*input.DiscordWebhookURL)
	}
	discordMention := ""
	if input.DiscordMention != nil {
		discordMention = strings.TrimSpace(*input.DiscordMention)
	}
	discordCooldown := 60
	if input.DiscordCooldownSeconds != nil {
		discordCooldown = *input.DiscordCooldownSeconds
	}
	if discordCooldown < 0 {
		return Camera{}, fmt.Errorf("%w: discord_cooldown_seconds must be >= 0", ErrInvalid)
	}
	if discordAlertsEnabled && discordWebhookURL == "" {
		return Camera{}, fmt.Errorf("%w: discord_webhook_url is required when discord alerts are enabled", ErrInvalid)
	}
	if discordWebhookURL != "" {
		if err := validateWebhookURL(discordWebhookURL); err != nil {
			return Camera{}, fmt.Errorf("%w: %v", ErrInvalid, err)
		}
	}

	now := time.Now()
	c := Camera{
		ID:                     uuid.New().String(),
		Name:                   input.Name,
		RTSPURL:                input.RTSPURL,
		Enabled:                enabled,
		RecordEnabled:          recordEnabled,
		DetectionSampleSeconds: input.DetectionSampleSeconds,
		TrackingEnabled:        trackingEnabled,
		TrackingMinConfidence:  trackingMinConfidence,
		TrackingLabels:         trackingLabels,
		DiscordAlertsEnabled:   discordAlertsEnabled,
		DiscordWebhookURL:      discordWebhookURL,
		DiscordMention:         discordMention,
		DiscordCooldownSeconds: discordCooldown,
		Position:               0,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	if err := s.repo.Create(c); err != nil {
		return Camera{}, fmt.Errorf("create camera: %w", err)
	}
	return c, nil
}

func (s *Service) Update(id string, input UpdateInput) (Camera, error) {
	c, err := s.repo.GetByID(id)
	if err != nil {
		return Camera{}, err
	}

	if input.Name != nil {
		if *input.Name == "" {
			return Camera{}, fmt.Errorf("%w: name cannot be empty", ErrInvalid)
		}
		c.Name = *input.Name
	}
	if input.RTSPURL != nil {
		if *input.RTSPURL == "" {
			return Camera{}, fmt.Errorf("%w: rtsp_url cannot be empty", ErrInvalid)
		}
		if !strings.HasPrefix(strings.TrimSpace(*input.RTSPURL), "rtsp://") {
			return Camera{}, fmt.Errorf("%w: rtsp_url must start with rtsp://", ErrInvalid)
		}
		c.RTSPURL = *input.RTSPURL
	}
	if input.Enabled != nil {
		c.Enabled = *input.Enabled
	}
	if input.RecordEnabled != nil {
		c.RecordEnabled = *input.RecordEnabled
	}
	if input.DetectionSampleSeconds != nil {
		if *input.DetectionSampleSeconds <= 0 {
			return Camera{}, fmt.Errorf("%w: detection_sample_seconds must be > 0", ErrInvalid)
		}
		c.DetectionSampleSeconds = input.DetectionSampleSeconds
	}
	if input.TrackingEnabled != nil {
		c.TrackingEnabled = *input.TrackingEnabled
	}
	if input.TrackingMinConfidence != nil {
		if !isValidConfidence(*input.TrackingMinConfidence) {
			return Camera{}, fmt.Errorf("%w: tracking_min_confidence must be between 0 and 1", ErrInvalid)
		}
		c.TrackingMinConfidence = *input.TrackingMinConfidence
	}
	if input.TrackingLabels != nil {
		c.TrackingLabels = normalizeTrackingLabels(*input.TrackingLabels)
	}
	if input.DiscordAlertsEnabled != nil {
		c.DiscordAlertsEnabled = *input.DiscordAlertsEnabled
	}
	if input.DiscordWebhookURL != nil {
		webhook := strings.TrimSpace(*input.DiscordWebhookURL)
		if webhook != "" {
			if err := validateWebhookURL(webhook); err != nil {
				return Camera{}, fmt.Errorf("%w: %v", ErrInvalid, err)
			}
		}
		c.DiscordWebhookURL = webhook
	}
	if input.DiscordMention != nil {
		c.DiscordMention = strings.TrimSpace(*input.DiscordMention)
	}
	if input.DiscordCooldownSeconds != nil {
		if *input.DiscordCooldownSeconds < 0 {
			return Camera{}, fmt.Errorf("%w: discord_cooldown_seconds must be >= 0", ErrInvalid)
		}
		c.DiscordCooldownSeconds = *input.DiscordCooldownSeconds
	}

	if c.DiscordAlertsEnabled && strings.TrimSpace(c.DiscordWebhookURL) == "" {
		return Camera{}, fmt.Errorf("%w: discord_webhook_url is required when discord alerts are enabled", ErrInvalid)
	}
	if input.Position != nil {
		c.Position = *input.Position
	}

	if err := s.repo.Update(c); err != nil {
		return Camera{}, fmt.Errorf("update camera: %w", err)
	}
	return c, nil
}

func (s *Service) Delete(id string) error {
	return s.repo.Delete(id)
}

func isValidConfidence(v float64) bool {
	return v >= 0 && v <= 1
}

func normalizeTrackingLabels(labels []string) []string {
	if len(labels) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(labels))
	out := make([]string, 0, len(labels))
	for _, raw := range labels {
		label := strings.ToLower(strings.TrimSpace(raw))
		if label == "" {
			continue
		}
		if _, exists := seen[label]; exists {
			continue
		}
		seen[label] = struct{}{}
		out = append(out, label)
	}
	return out
}

func validateWebhookURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("discord_webhook_url is invalid")
	}
	if u.Scheme != "https" {
		return fmt.Errorf("discord_webhook_url must use https")
	}
	if u.Host == "" {
		return fmt.Errorf("discord_webhook_url host is missing")
	}
	return nil
}
