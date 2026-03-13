package cameras

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("camera not found")
var ErrInvalid = errors.New("invalid input")

type Camera struct {
	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	RTSPURL                string    `json:"rtsp_url"`
	Enabled                bool      `json:"enabled"`
	RecordEnabled          bool      `json:"record_enabled"`
	DetectionSampleSeconds *int      `json:"detection_sample_seconds,omitempty"`
	TrackingEnabled        bool      `json:"tracking_enabled"`
	TrackingMinConfidence  float64   `json:"tracking_min_confidence"`
	TrackingLabels         []string  `json:"tracking_labels"`
	DiscordAlertsEnabled   bool      `json:"discord_alerts_enabled"`
	DiscordWebhookURL      string    `json:"discord_webhook_url"`
	DiscordMention         string    `json:"discord_mention"`
	DiscordCooldownSeconds int       `json:"discord_cooldown_seconds"`
	Position               int       `json:"position"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type CreateInput struct {
	Name                   string   `json:"name"`
	RTSPURL                string   `json:"rtsp_url"`
	Enabled                *bool    `json:"enabled"`
	RecordEnabled          *bool    `json:"record_enabled"`
	DetectionSampleSeconds *int     `json:"detection_sample_seconds"`
	TrackingEnabled        *bool    `json:"tracking_enabled"`
	TrackingMinConfidence  *float64 `json:"tracking_min_confidence"`
	TrackingLabels         []string `json:"tracking_labels"`
	DiscordAlertsEnabled   *bool    `json:"discord_alerts_enabled"`
	DiscordWebhookURL      *string  `json:"discord_webhook_url"`
	DiscordMention         *string  `json:"discord_mention"`
	DiscordCooldownSeconds *int     `json:"discord_cooldown_seconds"`
}

type UpdateInput struct {
	Name                   *string   `json:"name"`
	RTSPURL                *string   `json:"rtsp_url"`
	Enabled                *bool     `json:"enabled"`
	RecordEnabled          *bool     `json:"record_enabled"`
	DetectionSampleSeconds *int      `json:"detection_sample_seconds"`
	TrackingEnabled        *bool     `json:"tracking_enabled"`
	TrackingMinConfidence  *float64  `json:"tracking_min_confidence"`
	TrackingLabels         *[]string `json:"tracking_labels"`
	DiscordAlertsEnabled   *bool     `json:"discord_alerts_enabled"`
	DiscordWebhookURL      *string   `json:"discord_webhook_url"`
	DiscordMention         *string   `json:"discord_mention"`
	DiscordCooldownSeconds *int      `json:"discord_cooldown_seconds"`
	Position               *int      `json:"position"`
}
