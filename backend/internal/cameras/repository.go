package cameras

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List() ([]Camera, error) {
	rows, err := r.db.Query(
		`SELECT id, name, rtsp_url, enabled, record_enabled, detection_sample_seconds, tracking_enabled, tracking_min_confidence, tracking_labels,
		        discord_alerts_enabled, discord_webhook_url, discord_mention, discord_cooldown_seconds,
				discord_trigger_on_detection, discord_trigger_on_interval, discord_screenshot_interval_seconds,
				discord_include_motion_clip, discord_motion_clip_seconds, discord_record_format, discord_record_duration_seconds,
				position, created_at, updated_at
		 FROM cameras ORDER BY position, created_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("list cameras: %w", err)
	}
	defer rows.Close()

	result := make([]Camera, 0)
	for rows.Next() {
		c, err := scanCamera(rows)
		if err != nil {
			return nil, fmt.Errorf("scan camera: %w", err)
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *Repository) GetByID(id string) (Camera, error) {
	row := r.db.QueryRow(
		`SELECT id, name, rtsp_url, enabled, record_enabled, detection_sample_seconds, tracking_enabled, tracking_min_confidence, tracking_labels,
		        discord_alerts_enabled, discord_webhook_url, discord_mention, discord_cooldown_seconds,
				discord_trigger_on_detection, discord_trigger_on_interval, discord_screenshot_interval_seconds,
				discord_include_motion_clip, discord_motion_clip_seconds, discord_record_format, discord_record_duration_seconds,
				position, created_at, updated_at
		 FROM cameras WHERE id = ?`, id,
	)
	c, err := scanCamera(row)
	if err == sql.ErrNoRows {
		return Camera{}, ErrNotFound
	}
	return c, err
}

func (r *Repository) Create(c Camera) error {
	trackingLabelsJSON, err := json.Marshal(c.TrackingLabels)
	if err != nil {
		return fmt.Errorf("encode tracking labels: %w", err)
	}

	_, err = r.db.Exec(
		`INSERT INTO cameras (id, name, rtsp_url, enabled, record_enabled, detection_sample_seconds, tracking_enabled, tracking_min_confidence, tracking_labels,
		                     discord_alerts_enabled, discord_webhook_url, discord_mention, discord_cooldown_seconds,
							 discord_trigger_on_detection, discord_trigger_on_interval, discord_screenshot_interval_seconds,
							 discord_include_motion_clip, discord_motion_clip_seconds, discord_record_format, discord_record_duration_seconds,
							 position, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID,
		c.Name,
		c.RTSPURL,
		boolToInt(c.Enabled),
		boolToInt(c.RecordEnabled),
		nullableInt(c.DetectionSampleSeconds),
		boolToInt(c.TrackingEnabled),
		c.TrackingMinConfidence,
		string(trackingLabelsJSON),
		boolToInt(c.DiscordAlertsEnabled),
		c.DiscordWebhookURL,
		c.DiscordMention,
		c.DiscordCooldownSeconds,
		boolToInt(c.DiscordTriggerOnDetection),
		boolToInt(c.DiscordTriggerOnInterval),
		c.DiscordScreenshotIntervalSeconds,
		boolToInt(c.DiscordIncludeMotionClip),
		c.DiscordMotionClipSeconds,
		c.DiscordRecordFormat,
		c.DiscordRecordDurationSeconds,
		c.Position,
		c.CreatedAt,
		c.UpdatedAt,
	)
	return err
}

func (r *Repository) Update(c Camera) error {
	trackingLabelsJSON, err := json.Marshal(c.TrackingLabels)
	if err != nil {
		return fmt.Errorf("encode tracking labels: %w", err)
	}

	res, err := r.db.Exec(
		`UPDATE cameras
		 SET name=?, rtsp_url=?, enabled=?, record_enabled=?, detection_sample_seconds=?, tracking_enabled=?, tracking_min_confidence=?, tracking_labels=?,
		     discord_alerts_enabled=?, discord_webhook_url=?, discord_mention=?, discord_cooldown_seconds=?,
			 discord_trigger_on_detection=?, discord_trigger_on_interval=?, discord_screenshot_interval_seconds=?,
			 discord_include_motion_clip=?, discord_motion_clip_seconds=?, discord_record_format=?, discord_record_duration_seconds=?,
			 position=?, updated_at=?
		 WHERE id=?`,
		c.Name,
		c.RTSPURL,
		boolToInt(c.Enabled),
		boolToInt(c.RecordEnabled),
		nullableInt(c.DetectionSampleSeconds),
		boolToInt(c.TrackingEnabled),
		c.TrackingMinConfidence,
		string(trackingLabelsJSON),
		boolToInt(c.DiscordAlertsEnabled),
		c.DiscordWebhookURL,
		c.DiscordMention,
		c.DiscordCooldownSeconds,
		boolToInt(c.DiscordTriggerOnDetection),
		boolToInt(c.DiscordTriggerOnInterval),
		c.DiscordScreenshotIntervalSeconds,
		boolToInt(c.DiscordIncludeMotionClip),
		c.DiscordMotionClipSeconds,
		c.DiscordRecordFormat,
		c.DiscordRecordDurationSeconds,
		c.Position,
		time.Now(),
		c.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) Delete(id string) error {
	res, err := r.db.Exec(`DELETE FROM cameras WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanCamera(s scanner) (Camera, error) {
	var c Camera
	var enabled, recordEnabled int
	var trackingEnabled, discordAlertsEnabled int
	var discordTriggerOnDetection, discordTriggerOnInterval int
	var discordIncludeMotionClip int
	var trackingLabelsRaw string
	var trackingMinConfidence sql.NullFloat64
	var discordWebhookURL, discordMention, discordRecordFormat sql.NullString
	var discordCooldownSeconds sql.NullInt64
	var discordScreenshotIntervalSeconds sql.NullInt64
	var discordMotionClipSeconds sql.NullInt64
	var discordRecordDurationSeconds sql.NullInt64
	var sampleSeconds sql.NullInt64
	err := s.Scan(
		&c.ID,
		&c.Name,
		&c.RTSPURL,
		&enabled,
		&recordEnabled,
		&sampleSeconds,
		&trackingEnabled,
		&trackingMinConfidence,
		&trackingLabelsRaw,
		&discordAlertsEnabled,
		&discordWebhookURL,
		&discordMention,
		&discordCooldownSeconds,
		&discordTriggerOnDetection,
		&discordTriggerOnInterval,
		&discordScreenshotIntervalSeconds,
		&discordIncludeMotionClip,
		&discordMotionClipSeconds,
		&discordRecordFormat,
		&discordRecordDurationSeconds,
		&c.Position,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		return Camera{}, err
	}

	c.Enabled = enabled != 0
	c.RecordEnabled = recordEnabled != 0
	c.TrackingEnabled = trackingEnabled != 0
	c.DiscordAlertsEnabled = discordAlertsEnabled != 0
	c.DiscordTriggerOnDetection = discordTriggerOnDetection != 0
	c.DiscordTriggerOnInterval = discordTriggerOnInterval != 0
	c.DiscordIncludeMotionClip = discordIncludeMotionClip != 0
	if sampleSeconds.Valid {
		v := int(sampleSeconds.Int64)
		c.DetectionSampleSeconds = &v
	}
	if trackingMinConfidence.Valid {
		c.TrackingMinConfidence = trackingMinConfidence.Float64
	}
	if c.TrackingMinConfidence <= 0 {
		c.TrackingMinConfidence = 0.25
	}

	c.TrackingLabels = decodeTrackingLabels(trackingLabelsRaw)
	if discordWebhookURL.Valid {
		c.DiscordWebhookURL = strings.TrimSpace(discordWebhookURL.String)
	}
	if discordMention.Valid {
		c.DiscordMention = strings.TrimSpace(discordMention.String)
	}
	if discordCooldownSeconds.Valid {
		c.DiscordCooldownSeconds = int(discordCooldownSeconds.Int64)
	}
	if c.DiscordCooldownSeconds < 0 {
		c.DiscordCooldownSeconds = 0
	}
	if discordScreenshotIntervalSeconds.Valid {
		c.DiscordScreenshotIntervalSeconds = int(discordScreenshotIntervalSeconds.Int64)
	}
	if c.DiscordScreenshotIntervalSeconds <= 0 {
		c.DiscordScreenshotIntervalSeconds = 300
	}
	if discordMotionClipSeconds.Valid {
		c.DiscordMotionClipSeconds = int(discordMotionClipSeconds.Int64)
	}
	if c.DiscordMotionClipSeconds <= 0 {
		c.DiscordMotionClipSeconds = 4
	}
	if discordRecordDurationSeconds.Valid {
		c.DiscordRecordDurationSeconds = int(discordRecordDurationSeconds.Int64)
	}
	if c.DiscordRecordDurationSeconds <= 0 {
		c.DiscordRecordDurationSeconds = 60
	}
	if discordRecordFormat.Valid {
		c.DiscordRecordFormat = strings.ToLower(strings.TrimSpace(discordRecordFormat.String))
	}
	if c.DiscordRecordFormat == "" {
		c.DiscordRecordFormat = "webp"
	}
	return c, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullableInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func decodeTrackingLabels(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}

	labels := []string{}
	if err := json.Unmarshal([]byte(raw), &labels); err == nil {
		return normalizeTrackingLabels(labels)
	}

	// Backward-compatible fallback for any legacy comma-separated data.
	parts := strings.Split(raw, ",")
	return normalizeTrackingLabels(parts)
}
