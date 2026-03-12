package detections

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("detection event not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateEvents(cameraID string, snapshotPath string, createdAt time.Time, detections []Detection, rawPayload string) ([]Event, error) {
	if len(detections) == 0 {
		return []Event{}, nil
	}

	result := make([]Event, 0, len(detections))
	for _, d := range detections {
		bboxJSON, err := json.Marshal(d.BBox)
		if err != nil {
			return nil, fmt.Errorf("marshal bbox: %w", err)
		}

		id := uuid.New().String()
		raw := nullableString(rawPayload)

		_, err = r.db.Exec(
			`INSERT INTO detection_events (id, camera_id, object_label, confidence, bbox_json, snapshot_path, raw_payload, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			id,
			cameraID,
			d.Label,
			d.Confidence,
			string(bboxJSON),
			snapshotPath,
			raw,
			createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("insert detection event: %w", err)
		}

		result = append(result, Event{
			ID:           id,
			CameraID:     cameraID,
			ObjectLabel:  d.Label,
			Confidence:   d.Confidence,
			BBox:         d.BBox,
			SnapshotPath: snapshotPath,
			RawPayload:   ptrIfNonEmpty(rawPayload),
			CreatedAt:    createdAt,
		})
	}
	return result, nil
}

func (r *Repository) ListRecent(limit int, cameraID string) ([]Event, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	query := `SELECT id, camera_id, object_label, confidence, bbox_json, snapshot_path, raw_payload, created_at
		FROM detection_events`
	args := make([]any, 0, 2)
	if cameraID != "" {
		query += ` WHERE camera_id = ?`
		args = append(args, cameraID)
	}
	query += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list detection events: %w", err)
	}
	defer rows.Close()

	events := make([]Event, 0)
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan detection event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *Repository) GetByID(id string) (Event, error) {
	row := r.db.QueryRow(
		`SELECT id, camera_id, object_label, confidence, bbox_json, snapshot_path, raw_payload, created_at
		FROM detection_events
		WHERE id = ?`,
		id,
	)
	e, err := scanEvent(row)
	if err == sql.ErrNoRows {
		return Event{}, ErrNotFound
	}
	if err != nil {
		return Event{}, fmt.Errorf("get detection event: %w", err)
	}
	return e, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanEvent(s scanner) (Event, error) {
	var e Event
	var bboxJSON string
	var raw sql.NullString
	err := s.Scan(
		&e.ID,
		&e.CameraID,
		&e.ObjectLabel,
		&e.Confidence,
		&bboxJSON,
		&e.SnapshotPath,
		&raw,
		&e.CreatedAt,
	)
	if err != nil {
		return Event{}, err
	}
	if err := json.Unmarshal([]byte(bboxJSON), &e.BBox); err != nil {
		return Event{}, fmt.Errorf("decode bbox: %w", err)
	}
	if raw.Valid && raw.String != "" {
		e.RawPayload = &raw.String
	}
	return e, nil
}

func nullableString(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func ptrIfNonEmpty(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
