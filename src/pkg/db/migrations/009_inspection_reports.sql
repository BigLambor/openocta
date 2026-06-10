-- Structured inspection reports for L3 Facts traceability (C2-10/C2-12).

CREATE TABLE IF NOT EXISTS inspection_reports (
  id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL DEFAULT '',
  job_id TEXT NOT NULL DEFAULT '',
  cluster_id TEXT NOT NULL DEFAULT '',
  domain TEXT NOT NULL DEFAULT '',
  scenario_key TEXT NOT NULL DEFAULT '',
  score INTEGER,
  score_status TEXT NOT NULL DEFAULT '',
  validation_status TEXT NOT NULL DEFAULT '',
  confidence TEXT NOT NULL DEFAULT '',
  summary TEXT NOT NULL DEFAULT '',
  requires_approval INTEGER NOT NULL DEFAULT 0,
  report_json TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_inspection_reports_cluster ON inspection_reports (cluster_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_inspection_reports_run ON inspection_reports (run_id);
