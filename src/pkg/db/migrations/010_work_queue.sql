-- Work queue for trigger/execution architecture (Phase A).
-- See docs/ops-trigger-execution-architecture.md

ALTER TABLE job_runs ADD COLUMN parent_run_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_job_runs_parent ON job_runs (parent_run_id);

CREATE TABLE IF NOT EXISTS work_plans (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  trigger_type TEXT NOT NULL DEFAULT '',
  trigger_ref TEXT NOT NULL DEFAULT '',
  scenario_key TEXT NOT NULL DEFAULT '',
  parent_run_id TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'queued',
  priority INTEGER NOT NULL DEFAULT 0,
  idempotency_key TEXT NOT NULL DEFAULT '',
  scheduled_at_ms INTEGER NOT NULL DEFAULT 0,
  envelope_json TEXT NOT NULL DEFAULT '{}',
  plan_json TEXT NOT NULL DEFAULT '{}',
  error TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_work_plans_idempotency
  ON work_plans (idempotency_key) WHERE idempotency_key != '';

CREATE INDEX IF NOT EXISTS idx_work_plans_status ON work_plans (status, updated_at);

CREATE TABLE IF NOT EXISTS work_tasks (
  id TEXT PRIMARY KEY,
  plan_id TEXT NOT NULL,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  tier TEXT NOT NULL DEFAULT '',
  action TEXT NOT NULL DEFAULT '',
  object_type TEXT NOT NULL DEFAULT '',
  object_id TEXT NOT NULL DEFAULT '',
  parent_run_id TEXT NOT NULL DEFAULT '',
  child_run_id TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'queued',
  priority INTEGER NOT NULL DEFAULT 0,
  idempotency_key TEXT NOT NULL DEFAULT '',
  lease_until INTEGER NOT NULL DEFAULT 0,
  worker_id TEXT NOT NULL DEFAULT '',
  input_json TEXT NOT NULL DEFAULT '{}',
  output_json TEXT NOT NULL DEFAULT '{}',
  error TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  FOREIGN KEY (plan_id) REFERENCES work_plans(id)
);

CREATE INDEX IF NOT EXISTS idx_work_tasks_plan ON work_tasks (plan_id, status);
CREATE INDEX IF NOT EXISTS idx_work_tasks_queue ON work_tasks (status, priority DESC, created_at ASC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_work_tasks_idempotency
  ON work_tasks (idempotency_key) WHERE idempotency_key != '';
