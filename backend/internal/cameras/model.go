package cameras

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("camera not found")
var ErrInvalid = errors.New("invalid input")

type Camera struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	RTSPURL       string    `json:"rtsp_url"`
	Enabled       bool      `json:"enabled"`
	RecordEnabled bool      `json:"record_enabled"`
	Position      int       `json:"position"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateInput struct {
	Name          string `json:"name"`
	RTSPURL       string `json:"rtsp_url"`
	Enabled       *bool  `json:"enabled"`
	RecordEnabled *bool  `json:"record_enabled"`
}

type UpdateInput struct {
	Name          *string `json:"name"`
	RTSPURL       *string `json:"rtsp_url"`
	Enabled       *bool   `json:"enabled"`
	RecordEnabled *bool   `json:"record_enabled"`
	Position      *int    `json:"position"`
}
