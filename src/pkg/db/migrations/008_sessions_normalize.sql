-- Normalized session metadata (C1-21). Legacy blob table `sessions` is migrated in Go.

CREATE TABLE IF NOT EXISTS sessions_v1 (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  agent_id TEXT NOT NULL DEFAULT 'main',
  session_key TEXT NOT NULL,
  session_id TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  origin TEXT NOT NULL DEFAULT '',
  channel TEXT NOT NULL DEFAULT '',
  owner_id TEXT NOT NULL DEFAULT '',
  store_path TEXT NOT NULL DEFAULT '',
  detail_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_v1_store_key
  ON sessions_v1 (store_path, session_key);

CREATE INDEX IF NOT EXISTS idx_sessions_v1_agent_updated
  ON sessions_v1 (agent_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_sessions_v1_session_id
  ON sessions_v1 (session_id);
