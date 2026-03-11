package alerts

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

// --- Alert Rules ---

func (r *Repository) ListRules(cameraID string) ([]AlertRule, error) {
	rows, err := r.db.Query(
		`SELECT id, camera_id, name, type, enabled, config, created_at, updated_at
		 FROM alert_rules WHERE camera_id = ? ORDER BY created_at`,
		cameraID,
	)
	if err != nil {
		return nil, fmt.Errorf("list alert rules: %w", err)
	}
	defer rows.Close()

	result := make([]AlertRule, 0)
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, fmt.Errorf("scan rule: %w", err)
		}
		result = append(result, rule)
	}
	return result, rows.Err()
}

func (r *Repository) GetRule(id string) (AlertRule, error) {
	row := r.db.QueryRow(
		`SELECT id, camera_id, name, type, enabled, config, created_at, updated_at
		 FROM alert_rules WHERE id = ?`, id,
	)
	rule, err := scanRule(row)
	if err == sql.ErrNoRows {
		return AlertRule{}, ErrNotFound
	}
	return rule, err
}

func (r *Repository) CreateRule(rule AlertRule) error {
	_, err := r.db.Exec(
		`INSERT INTO alert_rules (id, camera_id, name, type, enabled, config, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.ID, rule.CameraID, rule.Name, string(rule.Type),
		boolToInt(rule.Enabled), rule.Config, rule.CreatedAt, rule.UpdatedAt,
	)
	return err
}

func (r *Repository) UpdateRule(rule AlertRule) error {
	res, err := r.db.Exec(
		`UPDATE alert_rules SET name=?, type=?, enabled=?, config=?, updated_at=? WHERE id=?`,
		rule.Name, string(rule.Type), boolToInt(rule.Enabled), rule.Config, time.Now(), rule.ID,
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

func (r *Repository) DeleteRule(id string) error {
	res, err := r.db.Exec(`DELETE FROM alert_rules WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Alert Events ---

func (r *Repository) ListEvents(ruleID string, limit int) ([]AlertEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(
		`SELECT id, rule_id, camera_id, triggered_at, COALESCE(snapshot_path,''), metadata
		 FROM alert_events WHERE rule_id = ? ORDER BY triggered_at DESC LIMIT ?`,
		ruleID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list alert events: %w", err)
	}
	defer rows.Close()

	result := make([]AlertEvent, 0)
	for rows.Next() {
		var e AlertEvent
		if err := rows.Scan(&e.ID, &e.RuleID, &e.CameraID, &e.TriggeredAt, &e.SnapshotPath, &e.Metadata); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (r *Repository) ListEventsByCamera(cameraID string, limit int) ([]AlertEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.Query(
		`SELECT id, rule_id, camera_id, triggered_at, COALESCE(snapshot_path,''), metadata
		 FROM alert_events WHERE camera_id = ? ORDER BY triggered_at DESC LIMIT ?`,
		cameraID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list camera alert events: %w", err)
	}
	defer rows.Close()

	result := make([]AlertEvent, 0)
	for rows.Next() {
		var e AlertEvent
		if err := rows.Scan(&e.ID, &e.RuleID, &e.CameraID, &e.TriggeredAt, &e.SnapshotPath, &e.Metadata); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (r *Repository) CreateEvent(e AlertEvent) error {
	var snapshotPath *string
	if e.SnapshotPath != "" {
		snapshotPath = &e.SnapshotPath
	}
	_, err := r.db.Exec(
		`INSERT INTO alert_events (id, rule_id, camera_id, triggered_at, snapshot_path, metadata)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		e.ID, e.RuleID, e.CameraID, e.TriggeredAt, snapshotPath, e.Metadata,
	)
	return err
}

// --- helpers ---

type scanner interface {
	Scan(dest ...any) error
}

func scanRule(s scanner) (AlertRule, error) {
	var rule AlertRule
	var enabled int
	var ruleType string
	err := s.Scan(&rule.ID, &rule.CameraID, &rule.Name, &ruleType, &enabled, &rule.Config, &rule.CreatedAt, &rule.UpdatedAt)
	rule.Type = AlertType(ruleType)
	rule.Enabled = enabled != 0
	return rule, err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
