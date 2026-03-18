-- Speeds up the default camera list ordering used by all dashboard loads.
CREATE INDEX IF NOT EXISTS idx_cameras_order ON cameras(position, created_at);
