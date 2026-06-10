-- Preserve full alert group payloads while repository code migrates from JSON
-- files to normalized alert tables.

ALTER TABLE alert_groups ADD COLUMN detail_json TEXT NOT NULL DEFAULT '{}';
