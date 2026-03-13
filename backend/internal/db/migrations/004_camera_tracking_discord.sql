-- Per-camera YOLO tracking controls.
ALTER TABLE cameras ADD COLUMN tracking_enabled INTEGER NOT NULL DEFAULT 0;
ALTER TABLE cameras ADD COLUMN tracking_min_confidence REAL NOT NULL DEFAULT 0.25;
ALTER TABLE cameras ADD COLUMN tracking_labels TEXT NOT NULL DEFAULT '[]';

-- Per-camera Discord webhook alert controls.
ALTER TABLE cameras ADD COLUMN discord_alerts_enabled INTEGER NOT NULL DEFAULT 0;
ALTER TABLE cameras ADD COLUMN discord_webhook_url TEXT NOT NULL DEFAULT '';
ALTER TABLE cameras ADD COLUMN discord_mention TEXT NOT NULL DEFAULT '';
ALTER TABLE cameras ADD COLUMN discord_cooldown_seconds INTEGER NOT NULL DEFAULT 60;

-- Frame dimensions let the frontend scale overlays correctly across aspect ratios.
ALTER TABLE detection_events ADD COLUMN frame_width INTEGER;
ALTER TABLE detection_events ADD COLUMN frame_height INTEGER;
