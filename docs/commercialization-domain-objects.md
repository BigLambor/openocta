# OpenOcta 商用核心领域对象边界

> 状态：商用化基线 v1  
> 适用范围：新增主业务能力、数据库 schema、API 与审计设计

## 领域对象清单

| 对象 | Owner 模块 | 主存储 | API 边界 | 说明 |
|------|------------|--------|----------|------|
| 资产 | `pkg/ops` | `openocta.db.assets`、`clusters`、`asset_relations` | `/api/ops/clusters`、Ops WebSocket method | 运维对象锚点，包含 domain、owner、region、components、monitor labels、connector refs。 |
| 告警 | `pkg/ops` | `openocta.db.alert_events`、`alert_groups`、`incident_timeline` | `/hooks/alert`、`/api/ops/alerts` | 原始告警、合并组、状态流转、证据与处置时间线。 |
| 事件 | `pkg/ops` | `openocta.db.incident_timeline`、`audit_logs` | 告警、巡检、任务 API 的 timeline 字段 | 业务事件必须可关联来源对象、操作者、时间和证据。 |
| 巡检 | `pkg/ops` | `openocta.db.inspection_reports`、`health_signals`、`job_runs` | `/api/ops/inspection`、Ops workbench | 巡检结果使用结构化报告，摘要可展示，原始证据可追溯。 |
| 任务 | `pkg/employees`、`pkg/cron`、`pkg/ops` | `openocta.db.tasks`、`jobs`、`job_runs`、`run_steps` | employee tasks、Cron、manual run API | 手动、Cron、告警触发、IM 触发最终归一到任务与执行记录。 |
| 审批 | `pkg/security` | `openocta.db.approvals`、`approval_steps` | approvals HTTP/WebSocket API | 高风险动作、工具执行、配置变更必须可审批、拒绝、超时和审计。 |
| 审计 | `pkg/security`、`pkg/agent/runtime` | `openocta.db.audit_logs`、`tool_invocations`、`model_usage` | audit query API | 记录用户、会话、任务、工具、模型和配置变更摘要，不落敏感明文。 |
| 会话 | `pkg/session` | `openocta.db.sessions_v1`，transcript JSONL 仅归档 | sessions HTTP/WebSocket API | DB 是列表、检索、用量和权限过滤主路径，JSONL 保留为短期 transcript 归档。见 [session-transcript-strategy.md](./session-transcript-strategy.md)。 |
| 通道 | `pkg/channels` | `openocta.db.channels`、`channel_sessions` | channels API、channel runtime | IM/机器人通道配置、运行状态、外部会话映射。 |
| 配置 | `pkg/config` | `openocta.db.config_versions`，配置文件为启动输入 | config API | 配置变更需要版本化、差异摘要、操作者和回滚点。 |
| 密钥 | `pkg/config`、`pkg/security` | `openocta.db.secrets` | secrets/config API | 业务配置使用 `secret_ref`，API 返回脱敏值，不返回密钥明文。 |

## 新增功能归属规则

1. 新增能力必须先归属到上表某个对象；无法归属时先扩展对象模型文档，再实现代码。
2. 涉及资产、告警、任务、审批、权限、健康结果的主写路径必须进入 `openocta.db` schema，不得新增 JSON 主存储。
3. JSON 文件只能作为兼容导入、备份导出、transcript 归档或测试 fixture。
4. 所有跨对象关系使用业务 ID，不暴露 SQLite 自增主键到 API。
5. 新表必须预留 `tenant_id`、`workspace_id`、`created_at`、`updated_at`，即使 Community 单机版暂时只写默认值。
