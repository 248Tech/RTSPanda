package cameras

import (
	"database/sql"
	"fmt"
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
		`SELECT id, name, rtsp_url, enabled, record_enabled, detection_sample_seconds, position, created_at, updated_at
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
		`SELECT id, name, rtsp_url, enabled, record_enabled, detection_sample_seconds, position, created_at, updated_at
		 FROM cameras WHERE id = ?`, id,
	)
	c, err := scanCamera(row)
	if err == sql.ErrNoRows {
		return Camera{}, ErrNotFound
	}
	return c, err
}

func (r *Repository) Create(c Camera) error {
	_, err := r.db.Exec(
		`INSERT INTO cameras (id, name, rtsp_url, enabled, record_enabled, detection_sample_seconds, position, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Name, c.RTSPURL, boolToInt(c.Enabled), boolToInt(c.RecordEnabled), nullableInt(c.DetectionSampleSeconds), c.Position, c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func (r *Repository) Update(c Camera) error {
	res, err := r.db.Exec(
		`UPDATE cameras SET name=?, rtsp_url=?, enabled=?, record_enabled=?, detection_sample_seconds=?, position=?, updated_at=? WHERE id=?`,
		c.Name, c.RTSPURL, boolToInt(c.Enabled), boolToInt(c.RecordEnabled), nullableInt(c.DetectionSampleSeconds), c.Position, time.Now(), c.ID,
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
	var sampleSeconds sql.NullInt64
	err := s.Scan(&c.ID, &c.Name, &c.RTSPURL, &enabled, &recordEnabled, &sampleSeconds, &c.Position, &c.CreatedAt, &c.UpdatedAt)
	c.Enabled = enabled != 0
	c.RecordEnabled = recordEnabled != 0
	if sampleSeconds.Valid {
		v := int(sampleSeconds.Int64)
		c.DetectionSampleSeconds = &v
	}
	return c, err
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
