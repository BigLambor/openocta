-- Foundation tables managed by the unified openocta.db migration runner.
-- Existing modules may still wrap these tables with compatibility import code,
-- but new table creation belongs here rather than scattered package init code.

CREATE TABLE IF NOT EXISTS sessions (
  store_path TEXT,
  session_key TEXT,
  detail_json TEXT,
  PRIMARY KEY (store_path, session_key)
);

CREATE TABLE IF NOT EXISTS cron_jobs (
  id TEXT PRIMARY KEY,
  detail_json TEXT
);

CREATE TABLE IF NOT EXISTS health_signals (
  object_type TEXT,
  object_id TEXT,
  type TEXT,
  source TEXT,
  detail_json TEXT,
  PRIMARY KEY (object_type, object_id, type, source)
);

CREATE TABLE IF NOT EXISTS health_snapshots (
  object_type TEXT,
  object_id TEXT,
  domain TEXT,
  detail_json TEXT,
  PRIMARY KEY (object_type, object_id)
);
