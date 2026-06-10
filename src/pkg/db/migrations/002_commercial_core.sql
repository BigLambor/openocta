-- Commercial core schema v1. These tables are the target DB surface for
-- Phase 1+ repository migrations and structured execution/audit trails.

CREATE TABLE IF NOT EXISTS assets (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  type TEXT NOT NULL,
  name TEXT NOT NULL,
  domain TEXT NOT NULL,
  owner TEXT NOT NULL DEFAULT '',
  region TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'unknown',
  attributes_json TEXT NOT NULL DEFAULT '{}',
  deleted_at INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_assets_workspace_domain ON assets (tenant_id, workspace_id, domain);
CREATE INDEX IF NOT EXISTS idx_assets_type_status ON assets (type, status);

CREATE TABLE IF NOT EXISTS clusters (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  asset_id TEXT NOT NULL,
  node_count INTEGER NOT NULL DEFAULT 0,
  components_json TEXT NOT NULL DEFAULT '[]',
  monitor_labels TEXT NOT NULL DEFAULT '',
  vm_url_ref TEXT NOT NULL DEFAULT '',
  metrics_base_url TEXT NOT NULL DEFAULT '',
  jmx_url TEXT NOT NULL DEFAULT '',
  fi_manager_url TEXT NOT NULL DEFAULT '',
  gbase_dsn_ref TEXT NOT NULL DEFAULT '',
  credentials_ref TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  FOREIGN KEY(asset_id) REFERENCES assets(id)
);

CREATE INDEX IF NOT EXISTS idx_clusters_asset ON clusters (asset_id);
CREATE INDEX IF NOT EXISTS idx_clusters_workspace ON clusters (tenant_id, workspace_id);

CREATE TABLE IF NOT EXISTS asset_relations (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  source_asset_id TEXT NOT NULL,
  target_asset_id TEXT NOT NULL,
  relation_type TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_asset_relations_source ON asset_relations (source_asset_id, relation_type);
CREATE INDEX IF NOT EXISTS idx_asset_relations_target ON asset_relations (target_asset_id, relation_type);

CREATE TABLE IF NOT EXISTS alert_groups (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  source TEXT NOT NULL,
  domain TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL,
  severity TEXT NOT NULL,
  status TEXT NOT NULL,
  original_count INTEGER NOT NULL DEFAULT 0,
  reduced_to INTEGER NOT NULL DEFAULT 1,
  session_key TEXT NOT NULL DEFAULT '',
  run_id TEXT NOT NULL DEFAULT '',
  alertname TEXT NOT NULL DEFAULT '',
  service TEXT NOT NULL DEFAULT '',
  instance TEXT NOT NULL DEFAULT '',
  cluster_id TEXT NOT NULL DEFAULT '',
  component TEXT NOT NULL DEFAULT '',
  review_status TEXT NOT NULL DEFAULT 'pending',
  suppression_category TEXT NOT NULL DEFAULT 'none',
  suppression_detail TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_alert_groups_workspace_status ON alert_groups (tenant_id, workspace_id, status);
CREATE INDEX IF NOT EXISTS idx_alert_groups_domain_severity ON alert_groups (domain, severity);

CREATE TABLE IF NOT EXISTS alert_events (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  group_id TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL,
  severity TEXT NOT NULL,
  title TEXT NOT NULL,
  message TEXT NOT NULL DEFAULT '',
  alertname TEXT NOT NULL DEFAULT '',
  service TEXT NOT NULL DEFAULT '',
  instance TEXT NOT NULL DEFAULT '',
  cluster_id TEXT NOT NULL DEFAULT '',
  component TEXT NOT NULL DEFAULT '',
  raw_json TEXT NOT NULL DEFAULT '{}',
  received_at INTEGER NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_alert_events_group ON alert_events (group_id);
CREATE INDEX IF NOT EXISTS idx_alert_events_received ON alert_events (received_at);

CREATE TABLE IF NOT EXISTS incident_timeline (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  subject_type TEXT NOT NULL,
  subject_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  operator_id TEXT NOT NULL DEFAULT '',
  run_id TEXT NOT NULL DEFAULT '',
  message TEXT NOT NULL DEFAULT '',
  evidence_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_incident_timeline_subject ON incident_timeline (subject_type, subject_id, created_at);

CREATE TABLE IF NOT EXISTS tasks (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  employee_id TEXT NOT NULL DEFAULT '',
  domain TEXT NOT NULL DEFAULT '',
  capability TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL,
  status TEXT NOT NULL,
  priority TEXT NOT NULL DEFAULT '',
  trigger_type TEXT NOT NULL DEFAULT '',
  trigger_ref TEXT NOT NULL DEFAULT '',
  workflow_json TEXT NOT NULL DEFAULT '{}',
  evaluation_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tasks_workspace_status ON tasks (tenant_id, workspace_id, status);
CREATE INDEX IF NOT EXISTS idx_tasks_employee_domain ON tasks (employee_id, domain);

CREATE TABLE IF NOT EXISTS jobs (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  kind TEXT NOT NULL,
  name TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  schedule_expr TEXT NOT NULL DEFAULT '',
  agent_id TEXT NOT NULL DEFAULT '',
  session_key TEXT NOT NULL DEFAULT '',
  delivery_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_jobs_workspace_kind ON jobs (tenant_id, workspace_id, kind);

CREATE TABLE IF NOT EXISTS job_runs (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  job_id TEXT NOT NULL DEFAULT '',
  task_id TEXT NOT NULL DEFAULT '',
  trigger_type TEXT NOT NULL DEFAULT '',
  trigger_ref TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  started_at INTEGER NOT NULL DEFAULT 0,
  finished_at INTEGER NOT NULL DEFAULT 0,
  error TEXT NOT NULL DEFAULT '',
  input_json TEXT NOT NULL DEFAULT '{}',
  output_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_job_runs_job ON job_runs (job_id, created_at);
CREATE INDEX IF NOT EXISTS idx_job_runs_status ON job_runs (tenant_id, workspace_id, status);

CREATE TABLE IF NOT EXISTS run_steps (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  run_id TEXT NOT NULL,
  step_order INTEGER NOT NULL,
  kind TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  started_at INTEGER NOT NULL DEFAULT 0,
  finished_at INTEGER NOT NULL DEFAULT 0,
  error TEXT NOT NULL DEFAULT '',
  input_summary TEXT NOT NULL DEFAULT '',
  output_summary TEXT NOT NULL DEFAULT '',
  metadata_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_run_steps_run_order ON run_steps (run_id, step_order);

CREATE TABLE IF NOT EXISTS tool_invocations (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  run_id TEXT NOT NULL DEFAULT '',
  session_id TEXT NOT NULL DEFAULT '',
  step_id TEXT NOT NULL DEFAULT '',
  tool_name TEXT NOT NULL,
  provider TEXT NOT NULL DEFAULT '',
  input_summary TEXT NOT NULL DEFAULT '',
  output_summary TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  duration_ms INTEGER NOT NULL DEFAULT 0,
  error TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tool_invocations_run ON tool_invocations (run_id, created_at);
CREATE INDEX IF NOT EXISTS idx_tool_invocations_session ON tool_invocations (session_id, created_at);

CREATE TABLE IF NOT EXISTS model_usage (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  run_id TEXT NOT NULL DEFAULT '',
  session_id TEXT NOT NULL DEFAULT '',
  provider TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  input_tokens INTEGER NOT NULL DEFAULT 0,
  output_tokens INTEGER NOT NULL DEFAULT 0,
  total_tokens INTEGER NOT NULL DEFAULT 0,
  cost_micros INTEGER NOT NULL DEFAULT 0,
  latency_ms INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_model_usage_run ON model_usage (run_id, created_at);
CREATE INDEX IF NOT EXISTS idx_model_usage_session ON model_usage (session_id, created_at);

CREATE TABLE IF NOT EXISTS approvals (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  subject_type TEXT NOT NULL,
  subject_id TEXT NOT NULL,
  requester_id TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  risk_level TEXT NOT NULL DEFAULT '',
  reason TEXT NOT NULL DEFAULT '',
  request_json TEXT NOT NULL DEFAULT '{}',
  result_json TEXT NOT NULL DEFAULT '{}',
  expires_at INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_approvals_status ON approvals (tenant_id, workspace_id, status);
CREATE INDEX IF NOT EXISTS idx_approvals_subject ON approvals (subject_type, subject_id);

CREATE TABLE IF NOT EXISTS approval_steps (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  approval_id TEXT NOT NULL,
  step_order INTEGER NOT NULL,
  approver_id TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  comment TEXT NOT NULL DEFAULT '',
  decided_at INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_approval_steps_approval ON approval_steps (approval_id, step_order);

CREATE TABLE IF NOT EXISTS audit_logs (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  actor_id TEXT NOT NULL DEFAULT '',
  action TEXT NOT NULL,
  object_type TEXT NOT NULL,
  object_id TEXT NOT NULL DEFAULT '',
  request_id TEXT NOT NULL DEFAULT '',
  run_id TEXT NOT NULL DEFAULT '',
  session_id TEXT NOT NULL DEFAULT '',
  summary TEXT NOT NULL DEFAULT '',
  metadata_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_object ON audit_logs (object_type, object_id, created_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor ON audit_logs (actor_id, created_at);

CREATE TABLE IF NOT EXISTS secrets (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  name TEXT NOT NULL,
  provider TEXT NOT NULL DEFAULT '',
  secret_ref TEXT NOT NULL,
  redacted_value TEXT NOT NULL DEFAULT '',
  metadata_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_secrets_name_scope ON secrets (tenant_id, workspace_id, name);

CREATE TABLE IF NOT EXISTS config_versions (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  config_scope TEXT NOT NULL,
  actor_id TEXT NOT NULL DEFAULT '',
  before_summary TEXT NOT NULL DEFAULT '',
  after_summary TEXT NOT NULL DEFAULT '',
  diff_json TEXT NOT NULL DEFAULT '{}',
  rollback_ref TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_config_versions_scope ON config_versions (tenant_id, workspace_id, config_scope, created_at);
