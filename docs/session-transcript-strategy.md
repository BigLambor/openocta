# Session 与 Transcript 存储边界（C1-23）

> 状态：商用化基线 v1（2026-06-10）  
> 关联：`C1-21` / `C1-22`（`sessions_v1`）、[迁移矩阵](./json-store-migration-matrix.md)、[主业务存储准入](./main-business-storage-admission.md)

## 1. 结论（一句话）

**会话元数据以 `openocta.db.sessions_v1` 为唯一主写与列表/检索主路径；Transcript 原文仍以 JSONL 文件归档，仅供回放、用量统计与 Ops 诊断读取，不得作为会话索引或权限过滤的数据源。**

## 2. 双轨职责划分

| 层 | 存储 | 主写 | 主读场景 | 不承担的职责 |
|----|------|------|----------|--------------|
| **会话索引（L2 元数据）** | `openocta.db` → `sessions_v1` | `pkg/session` repository | `sessions.list`、筛选/排序、多 agent 合并、`updatedAt` 触达 | 不存完整消息正文 |
| **Transcript 归档（L3 原文）** | `{OPENOCTA_STATE_DIR}/agents/<agentId>/sessions/*.jsonl` | `pkg/session/transcript.go` append | `chat.history`、Agent 上下文加载、用量/日志 API、告警 Root Cause 抽取 | 不做会话列表、不做 RBAC 过滤 |

```text
用户 / IM / Cron
       │
       ▼
  chat.send / sessions.*
       │
       ├─► sessions_v1 (DB)     title / key / agent / channel / updatedAt / sessionFile 索引
       │
       └─► <sessionId>.jsonl    user/assistant/tool 消息流 + usage 行（append-only）
```

## 3. 路径与命名

### 3.1 状态目录

| 变量 | 默认 |
|------|------|
| `OPENOCTA_STATE_DIR` | `~/.openocta`（Windows：`%APPDATA%\openocta`） |

### 3.2 会话 store（DB 键）

每个 agent 对应一个逻辑 store 路径（**仅作 DB 分区键，不再作为主写 JSON 文件**）：

```text
{stateDir}/agents/{agentId}/sessions/sessions.json
```

- `LoadSessionStore` / `SaveSessionStore` 在 DB 可用时读写 `sessions_v1.store_path = 上述路径`。
- 旧版 `sessions.json` 在首次导入后删除，备份为 `sessions.json.bak.<timestamp>`。

### 3.3 Transcript JSONL

默认路径（`pkg/session/paths.go`）：

```text
{stateDir}/agents/{agentId}/sessions/{sessionId}.jsonl
```

当 `sessions_v1.detail_json.sessionFile` 非空时，优先使用：

```text
{dirname(store_path)}/{sessionFile}
```

解析逻辑见 `handlers/sessions.go` → `resolveSessionTranscriptPath`。

### 3.4 sessionKey 与 sessionId

| 概念 | 示例 | 存储位置 |
|------|------|----------|
| `session_key` | `agent:main:channel:feishu:oc_xxx` | `sessions_v1.session_key` |
| `session_id` | UUID 或 sanitize 后的 id | `sessions_v1.session_id` + JSONL 文件名 |
| `store_path` | `.../agents/main/sessions/sessions.json` | `sessions_v1.store_path` |

## 4. `sessions_v1` 字段映射

| DB 列 | 来源（SessionEntry / 业务） | 用途 |
|-------|----------------------------|------|
| `agent_id` | 从 `session_key` 解析 | 多 agent 列表、隔离 |
| `session_key` | Gateway canonical key | 唯一索引（与 store_path 复合） |
| `session_id` | `SessionEntry.sessionId` | 关联 transcript、JobRun |
| `title` | `SessionEntry.label` | UI 展示 |
| `origin` | `SessionEntry.spawnedBy` | 来源（webhook/cron 等） |
| `channel` | `SessionEntry.channel` | IM 通道 |
| `store_path` | agent sessions.json 逻辑路径 | 多 store 隔离 |
| `detail_json` | thinkingLevel、sessionFile、skillsSnapshot 等扩展 | 非检索字段 blob |
| `created_at` / `updated_at` | 首见 / 最后活跃 | 列表排序、活跃过滤 |

扩展字段** intentionally 留在 `detail_json`**，待后续有检索需求再拆列（Phase 2+）。

## 5. Transcript JSONL 格式（归档契约）

- **编码**：UTF-8，一行一条 JSON（JSONL）。
- **首行**：`type: "session"` 头（`CurrentSessionVersion = 2`）。
- **后续行**：`message` 包装或直接 message 对象；含 `role`、`content[]`、`timestamp`；assistant 行可含 `usage` / `provider` / `model` / `durationMs`。
- **写入方式**：仅 append（`AppendUserMessage`、`AppendAssistantMessage`、`AppendTranscriptLine` 等）。
- **单行上限**：32 MiB（`maxTranscriptLineBytes`），防止 scanner 截断导致整文件不可读。

**禁止**在 transcript 中维护会话列表或反向索引；新增会话必须先写 `sessions_v1`。

## 6. 读写路径对照（代码）

### 6.1 必须走 DB（`sessions_v1`）

| 入口 | 包/文件 |
|------|---------|
| `sessions.list` / `create` / `ensure` / `patch` / `reset` / `delete` | `handlers/sessions.go` → `LoadSessionStore` / `SaveSessionStore` |
| `UpdateSessionUpdatedAt` | `session/store.go` → `UpsertEntry` |
| 多 agent 合并列表 | `LoadCombinedSessionStore` |

### 6.2 必须走 JSONL（transcript）

| 入口 | 包/文件 |
|------|---------|
| `chat.send` 用户/助手/tool 落盘 | `handlers/chat.go` + `session/transcript.go` |
| `chat.history` | `ReadTranscriptMessages` |
| `sessions.usage` / `usage.timeseries` / `usage.logs` | `session/usage.go`（扫描 jsonl） |
| 告警 Root Cause Markdown | `ops/alerts_service.go`（读 assistant 最后一条） |
| 巡检 toolRuns 补全 | `ops/inspection.go`（读 transcript tool 行，非分数主路径） |
| Agent runtime 历史 | `agent/runtime` session history loader |

### 6.3 已结构化、不应再依赖 transcript 的商用路径

| 场景 | 主路径 |
|------|--------|
| 巡检分数 / 报告 | Agent 输出 JSON → `InspectionReport` 校验（C2-11）→ Facts |
| Token/Tool 审计（任务维度） | `model_usage` / `tool_invocations` + JobRun（C2-6～8） |
| 会话列表 / 权限 | `sessions_v1` + RBAC |

## 7. 生命周期与保留（当前策略）

| 阶段 | 行为 | 实现状态 |
|------|------|----------|
| 创建 | DB upsert + `EnsureTranscriptFile` | 已实现 |
| 活跃 | 每次消息 append JSONL；`updated_at` 触达 DB | 已实现 |
| Reset（/new） | 新 `session_id` + 新 jsonl；旧 jsonl **保留**（审计/复盘） | 已实现 |
| Delete | `sessions.delete` 删 DB 条目；transcript 文件删除策略见 handler | 已实现（文件随删除逻辑） |
| 冷归档 / 压缩 | 将旧 jsonl 迁至 object storage 或 tarball | **未实现**（Phase 2+） |
| 自动 TTL 清理 | 按 `updated_at` 或磁盘配额清理 jsonl | **未实现**；运维可手动删除 |

**短期（MVP）**：JSONL 与 DB 同生命周期共存于 state 目录；备份时两者均需包含。

**中期（规划）**：

1. `session_messages` 表：可检索消息摘要、脱敏预览。
2. `transcript_archives`：冷存 jsonl 路径 + checksum + 保留截止日。
3. 列表/搜索完全走 DB；JSONL 仅按需拉取原文。

## 8. 备份与迁移

| 对象 | 备份方式 | 恢复要点 |
|------|----------|----------|
| `openocta.db` | 含 `sessions_v1` 全量 | 会话列表恢复 |
| `agents/*/sessions/*.jsonl` | 目录级复制 | 与 DB 中 `session_id` / `sessionFile` 一致 |
| 遗留 `sessions.json` | 导入后 `.bak.<ts>` | 不应再作为主写 |

导入幂等键：`store_path + session_key`（`sessions_v1` 行 id 为二者哈希）。

## 9. 准入与禁止事项

### 允许

- 向 JSONL **append** 消息与 usage 行。
- 从 JSONL **只读** 全文、用量、tool 证据（明确标注为归档读路径）。
- 在 `sessions_v1` **upsert** 元数据；`detail_json` 存 UI/Agent 扩展。

### 禁止（新代码 PR 需拒绝）

- 用 JSONL 扫描实现 `sessions.list` 或全局搜索。
- 新增 `sessions.json` 主写（无 DB 时 legacy fallback 除外）。
- 在 transcript 中写入会话索引、权限、tenant 字段并作为主数据源。
- 将 transcript 正则解析作为商用巡检/健康分主路径（见 C2-11 / C2-13）。

## 10. 验收对照（C1-23）

| 验收项 | 文档章节 |
|--------|----------|
| JSONL 短期仍作为归档 | §2、§5、§7 |
| DB 作为列表/查询主路径 | §2、§4、§6.1 |
| 路径与字段边界清晰 | §3、§4 |
| 与迁移矩阵一致 | §8 + [json-store-migration-matrix.md](./json-store-migration-matrix.md) |

## 11. 相关代码索引

| 模块 | 路径 |
|------|------|
| Session repository | `src/pkg/session/repository.go` |
| Store 门面 | `src/pkg/session/store.go` |
| Transcript I/O | `src/pkg/session/transcript.go` |
| 路径解析 | `src/pkg/session/paths.go` |
| 用量扫描 | `src/pkg/session/usage.go` |
| Gateway sessions | `src/pkg/gateway/handlers/sessions.go` |
| Migration | `src/pkg/db/migrations/008_sessions_normalize.sql` |
