-- Normalize cron storage onto jobs + job_schedules and retire cron_jobs as primary store.

ALTER TABLE jobs ADD COLUMN detail_json TEXT NOT NULL DEFAULT '{}';

CREATE TABLE IF NOT EXISTS job_schedules (
  job_id TEXT PRIMARY KEY,
  schedule_kind TEXT NOT NULL DEFAULT '',
  schedule_at TEXT NOT NULL DEFAULT '',
  every_ms INTEGER NOT NULL DEFAULT 0,
  anchor_ms INTEGER NOT NULL DEFAULT 0,
  cron_expr TEXT NOT NULL DEFAULT '',
  timezone TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_job_schedules_kind ON job_schedules (schedule_kind);
