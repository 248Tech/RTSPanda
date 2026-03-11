-- Add recording support to cameras
ALTER TABLE cameras ADD COLUMN record_enabled INTEGER NOT NULL DEFAULT 0;

-- Alert rules: user-defined conditions to watch for on a camera
CREATE TABLE IF NOT EXISTS alert_rules (
    id         TEXT PRIMARY KEY,
    camera_id  TEXT NOT NULL REFERENCES cameras(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL CHECK(type IN ('motion','connectivity','object_detection')),
    enabled    INTEGER NOT NULL DEFAULT 1,
    config     TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- Alert events: each triggered occurrence of an alert rule
CREATE TABLE IF NOT EXISTS alert_events (
    id            TEXT PRIMARY KEY,
    rule_id       TEXT NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
    camera_id     TEXT NOT NULL,
    triggered_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    snapshot_path TEXT,
    metadata      TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_alert_rules_camera ON alert_rules(camera_id);
CREATE INDEX IF NOT EXISTS idx_alert_events_rule  ON alert_events(rule_id);
CREATE INDEX IF NOT EXISTS idx_alert_events_camera ON alert_events(camera_id);
