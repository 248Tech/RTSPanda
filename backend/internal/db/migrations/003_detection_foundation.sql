-- Camera-level override for detection frame sample interval (seconds).
-- NULL means "use global DETECTION_SAMPLE_INTERVAL_SECONDS".
ALTER TABLE cameras ADD COLUMN detection_sample_seconds INTEGER;

-- Object detection events emitted by the async detector pipeline.
CREATE TABLE IF NOT EXISTS detection_events (
    id            TEXT PRIMARY KEY,
    camera_id     TEXT NOT NULL REFERENCES cameras(id) ON DELETE CASCADE,
    object_label  TEXT NOT NULL,
    confidence    REAL NOT NULL,
    bbox_json     TEXT NOT NULL,
    snapshot_path TEXT NOT NULL,
    raw_payload   TEXT,
    created_at    DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_detection_events_camera_created
    ON detection_events(camera_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_detection_events_created
    ON detection_events(created_at DESC);
