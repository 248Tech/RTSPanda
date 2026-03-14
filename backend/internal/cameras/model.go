package cameras

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("camera not found")
var ErrInvalid = errors.New("invalid input")

type Camera struct {
	ID                               string    `json:"id"`
	Name                             string    `json:"name"`
	RTSPURL                          string    `json:"rtsp_url"`
	Enabled                          bool      `json:"enabled"`
	RecordEnabled                    bool      `json:"record_enabled"`
	DetectionSampleSeconds           *int      `json:"detection_sample_seconds,omitempty"`
	TrackingEnabled                  bool      `json:"tracking_enabled"`
	TrackingMinConfidence            float64   `json:"tracking_min_confidence"`
	TrackingLabels                   []string  `json:"tracking_labels"`
	DiscordAlertsEnabled             bool      `json:"discord_alerts_enabled"`
	DiscordWebhookURL                string    `json:"discord_webhook_url"`
	DiscordMention                   string    `json:"discord_mention"`
	DiscordCooldownSeconds           int       `json:"discord_cooldown_seconds"`
	DiscordTriggerOnDetection        bool      `json:"discord_trigger_on_detection"`
	DiscordTriggerOnInterval         bool      `json:"discord_trigger_on_interval"`
	DiscordScreenshotIntervalSeconds int       `json:"discord_screenshot_interval_seconds"`
	DiscordIncludeMotionClip         bool      `json:"discord_include_motion_clip"`
	DiscordMotionClipSeconds         int       `json:"discord_motion_clip_seconds"`
	DiscordRecordFormat              string    `json:"discord_record_format"`
	DiscordRecordDurationSeconds     int       `json:"discord_record_duration_seconds"`
	Position                         int       `json:"position"`
	CreatedAt                        time.Time `json:"created_at"`
	UpdatedAt                        time.Time `json:"updated_at"`
}

type CreateInput struct {
	Name                             string   `json:"name"`
	RTSPURL                          string   `json:"rtsp_url"`
	Enabled                          *bool    `json:"enabled"`
	RecordEnabled                    *bool    `json:"record_enabled"`
	DetectionSampleSeconds           *int     `json:"detection_sample_seconds"`
	TrackingEnabled                  *bool    `json:"tracking_enabled"`
	TrackingMinConfidence            *float64 `json:"tracking_min_confidence"`
	TrackingLabels                   []string `json:"tracking_labels"`
	DiscordAlertsEnabled             *bool    `json:"discord_alerts_enabled"`
	DiscordWebhookURL                *string  `json:"discord_webhook_url"`
	DiscordMention                   *string  `json:"discord_mention"`
	DiscordCooldownSeconds           *int     `json:"discord_cooldown_seconds"`
	DiscordTriggerOnDetection        *bool    `json:"discord_trigger_on_detection"`
	DiscordTriggerOnInterval         *bool    `json:"discord_trigger_on_interval"`
	DiscordScreenshotIntervalSeconds *int     `json:"discord_screenshot_interval_seconds"`
	DiscordIncludeMotionClip         *bool    `json:"discord_include_motion_clip"`
	DiscordMotionClipSeconds         *int     `json:"discord_motion_clip_seconds"`
	DiscordRecordFormat              *string  `json:"discord_record_format"`
	DiscordRecordDurationSeconds     *int     `json:"discord_record_duration_seconds"`
}

type UpdateInput struct {
	Name                             *string   `json:"name"`
	RTSPURL                          *string   `json:"rtsp_url"`
	Enabled                          *bool     `json:"enabled"`
	RecordEnabled                    *bool     `json:"record_enabled"`
	DetectionSampleSeconds           *int      `json:"detection_sample_seconds"`
	TrackingEnabled                  *bool     `json:"tracking_enabled"`
	TrackingMinConfidence            *float64  `json:"tracking_min_confidence"`
	TrackingLabels                   *[]string `json:"tracking_labels"`
	DiscordAlertsEnabled             *bool     `json:"discord_alerts_enabled"`
	DiscordWebhookURL                *string   `json:"discord_webhook_url"`
	DiscordMention                   *string   `json:"discord_mention"`
	DiscordCooldownSeconds           *int      `json:"discord_cooldown_seconds"`
	DiscordTriggerOnDetection        *bool     `json:"discord_trigger_on_detection"`
	DiscordTriggerOnInterval         *bool     `json:"discord_trigger_on_interval"`
	DiscordScreenshotIntervalSeconds *int      `json:"discord_screenshot_interval_seconds"`
	DiscordIncludeMotionClip         *bool     `json:"discord_include_motion_clip"`
	DiscordMotionClipSeconds         *int      `json:"discord_motion_clip_seconds"`
	DiscordRecordFormat              *string   `json:"discord_record_format"`
	DiscordRecordDurationSeconds     *int      `json:"discord_record_duration_seconds"`
	Position                         *int      `json:"position"`
}
