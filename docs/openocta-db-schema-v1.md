# openocta.db Schema v1 草案

> 状态：Phase 0 草案，供 Phase 1 migration 实现使用  
> 目标：覆盖商用最小闭环的资产、告警、任务、会话、审批、审计，并预留 tenant/workspace

## 基础约定

- 主键使用文本业务 ID，格式由业务模块生成。
- 所有核心表包含 `tenant_id`、`workspace_id`、`created_at`、`updated_at`。
- 时间字段使用 Unix milliseconds，避免不同方言 timestamp 行为差异。
- JSON 字段仅存放扩展属性、证据摘要或兼容载荷，不作为可查询核心字段的唯一来源。
- 软删除字段统一为 `deleted_at`，默认 `0`。

## Migration 管理

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  checksum TEXT NOT NULL,
  applied_at INTEGER NOT NULL
);
```

## 核心表草案

```sql
CREATE TABLE assets (
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

CREATE TABLE clusters (
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

CREATE TABLE asset_relations (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  source_asset_id TEXT NOT NULL,
  target_asset_id TEXT NOT NULL,
  relation_type TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE alert_events (
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

CREATE TABLE alert_groups (
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
  review_status TEXT NOT NULL DEFAULT 'pending',
  suppression_category TEXT NOT NULL DEFAULT 'none',
  suppression_detail TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE incident_timeline (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  subject_type TEXT NOT NULL,
  subject_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  operator_id TEXT NOT NULL DEFAULT '',
  message TEXT NOT NULL DEFAULT '',
  evidence_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE tasks (
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

CREATE TABLE jobs (
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

CREATE TABLE job_runs (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  job_id TEXT NOT NULL DEFAULT '',
  task_id TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  started_at INTEGER NOT NULL DEFAULT 0,
  finished_at INTEGER NOT NULL DEFAULT 0,
  error TEXT NOT NULL DEFAULT '',
  input_json TEXT NOT NULL DEFAULT '{}',
  output_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE sessions (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  agent_id TEXT NOT NULL DEFAULT '',
  session_key TEXT NOT NULL,
  session_id TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  origin TEXT NOT NULL DEFAULT '',
  channel TEXT NOT NULL DEFAULT '',
  owner_id TEXT NOT NULL DEFAULT '',
  store_path TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE approvals (
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
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE approval_steps (
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

CREATE TABLE audit_logs (
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
```

## 后续 Phase 1 必须补齐

- SQL migration 文件与 checksum 校验。
- JSON 导入 adapter 和备份策略。
- SQLite repository 单元测试与并发写测试。
- Postgres 方言约束说明。
