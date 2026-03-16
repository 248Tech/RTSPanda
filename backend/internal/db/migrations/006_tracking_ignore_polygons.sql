-- Per-camera ignore polygons for filtering out known noisy regions.
-- Coordinates are normalized [0..1] in image space.
ALTER TABLE cameras ADD COLUMN tracking_ignore_polygons TEXT NOT NULL DEFAULT '[]';
