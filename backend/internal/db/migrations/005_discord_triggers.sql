-- Expanded Discord alert controls for YOLO-native triggers and media behavior.
ALTER TABLE cameras ADD COLUMN discord_trigger_on_detection INTEGER NOT NULL DEFAULT 1;
ALTER TABLE cameras ADD COLUMN discord_trigger_on_interval INTEGER NOT NULL DEFAULT 0;
ALTER TABLE cameras ADD COLUMN discord_screenshot_interval_seconds INTEGER NOT NULL DEFAULT 300;
ALTER TABLE cameras ADD COLUMN discord_include_motion_clip INTEGER NOT NULL DEFAULT 1;
ALTER TABLE cameras ADD COLUMN discord_motion_clip_seconds INTEGER NOT NULL DEFAULT 4;
ALTER TABLE cameras ADD COLUMN discord_record_format TEXT NOT NULL DEFAULT 'webp';
ALTER TABLE cameras ADD COLUMN discord_record_duration_seconds INTEGER NOT NULL DEFAULT 60;
