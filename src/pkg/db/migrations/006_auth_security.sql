CREATE TABLE IF NOT EXISTS login_lockouts (
  scope TEXT NOT NULL,
  key TEXT NOT NULL,
  fail_count INTEGER NOT NULL DEFAULT 0,
  locked_until INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (scope, key)
);

CREATE INDEX IF NOT EXISTS idx_login_lockouts_locked_until ON login_lockouts (locked_until);

ALTER TABLE user_tokens ADD COLUMN created_at INTEGER NOT NULL DEFAULT 0;
