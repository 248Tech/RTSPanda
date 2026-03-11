package alerts

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("alert rule not found")
var ErrInvalid = errors.New("invalid input")

// AlertType enumerates supported alert trigger types.
type AlertType string

const (
	AlertTypeMotion          AlertType = "motion"
	AlertTypeConnectivity    AlertType = "connectivity"
	AlertTypeObjectDetection AlertType = "object_detection"
)

// AlertRule defines conditions under which an alert should fire for a camera.
type AlertRule struct {
	ID        string    `json:"id"`
	CameraID  string    `json:"camera_id"`
	Name      string    `json:"name"`
	Type      AlertType `json:"type"`
	Enabled   bool      `json:"enabled"`
	Config    string    `json:"config"` // JSON blob for type-specific settings
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AlertEvent is a recorded occurrence of an alert firing.
type AlertEvent struct {
	ID           string    `json:"id"`
	RuleID       string    `json:"rule_id"`
	CameraID     string    `json:"camera_id"`
	TriggeredAt  time.Time `json:"triggered_at"`
	SnapshotPath string    `json:"snapshot_path,omitempty"`
	Metadata     string    `json:"metadata"` // JSON blob
}

type CreateRuleInput struct {
	CameraID string    `json:"camera_id"`
	Name     string    `json:"name"`
	Type     AlertType `json:"type"`
	Enabled  *bool     `json:"enabled"`
	Config   string    `json:"config"`
}

type UpdateRuleInput struct {
	Name    *string    `json:"name"`
	Type    *AlertType `json:"type"`
	Enabled *bool      `json:"enabled"`
	Config  *string    `json:"config"`
}

type CreateEventInput struct {
	RuleID       string `json:"rule_id"`
	CameraID     string `json:"camera_id"`
	SnapshotPath string `json:"snapshot_path"`
	Metadata     string `json:"metadata"`
}
