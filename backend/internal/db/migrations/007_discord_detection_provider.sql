-- Camera-level alert source selection for Discord detection events.
ALTER TABLE cameras ADD COLUMN discord_detection_provider TEXT NOT NULL DEFAULT 'yolo';
ALTER TABLE cameras ADD COLUMN frigate_camera_name TEXT NOT NULL DEFAULT '';
