-- Session-level approval whitelist (tool execution bypass until expires_at).

CREATE TABLE IF NOT EXISTS approval_session_whitelist (
  session_id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  expires_at INTEGER NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_approval_session_whitelist_expires
  ON approval_session_whitelist (tenant_id, workspace_id, expires_at);
