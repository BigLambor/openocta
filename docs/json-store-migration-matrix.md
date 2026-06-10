# 旧 JSON 数据迁移矩阵

> 状态：Phase 0 基线清单  
> 目标：明确旧 JSON 文件的目标表、迁移策略和退役条件

| 旧路径 | 当前 owner | 目标主存储 | 优先级 | 导入策略 | 备份策略 | 幂等键 | 退役条件 |
|--------|------------|------------|--------|----------|----------|--------|----------|
| `ops/clusters.json` | `pkg/ops` | `assets`、`clusters`、`asset_relations` | P0 | 启动 migration 读取旧文件，按 cluster ID upsert。 | 导入前复制为 `clusters.json.bak.<timestamp>`。 | `cluster.id` | Ops cluster CRUD 全部走 DB，JSON 仅保留备份。 |
| `ops/alerts.json` | `pkg/ops` | `alert_events`、`alert_groups`、`incident_timeline` | P0 | 按 group ID upsert，timeline 拆入 `incident_timeline`，events 保留原始载荷。 | 导入前复制为 `alerts.json.bak.<timestamp>`。 | `alert_group.id`、`event.alert_id` | 告警列表、状态流转、review 全部走 DB。 |
| `employee_tasks/*.json` | `pkg/employees` | `tasks`、`job_runs` | P0 | 扫描目录，按任务 ID upsert，保留 workflow/evaluation JSON。 | 文件级备份到 `employee_tasks_backup/<timestamp>/`。 | `task.id` | 任务列表、加载、删除全部走 DB。 |
| `cron/jobs.json` | `pkg/cron` | `jobs`、`schedules`、`job_runs` | P1 | 启动 Cron service 前导入，按 job ID upsert。 | 导入前复制为 `jobs.json.bak.<timestamp>`。 | `job.id` | Cron list/add/update/remove 全部走 DB。 |
| `sessions.json` | `pkg/session` | `sessions` | P1 | 按 session key + agent upsert，保留 store path。 | 导入前复制为 `sessions.json.bak.<timestamp>`。 | `agent_id + session_key` | 会话列表和检索主路径走 DB。 |
| `transcripts/*.jsonl` | `pkg/session` | `sessions_v1` 索引；JSONL 暂保留归档 | P1/P2 | 只导入会话元数据、首末消息时间、摘要和用量索引。 | 归档目录按保留策略备份。 | `session_id + file path` | DB 可支撑列表、搜索、脱敏摘要；JSONL 仅用于原文回放。详见 [session-transcript-strategy.md](./session-transcript-strategy.md)。 |

## 迁移通用验收

- 空库启动不写 demo 数据。
- 旧文件存在时自动导入并生成备份。
- 重复迁移不产生重复数据。
- 导入失败时不得删除旧文件。
- 导入完成后所有主读写路径切换到 DB。
