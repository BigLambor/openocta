# OpenOcta 从 Demo 到商用产品的架构演进方案

> 日期：2026-06-09  
> 状态：初版评估与实施计划  
> 范围：后端架构、数据层、权限安全、任务编排、运维集成、部署运维、工程质量  
> 关联文档：`architecture-paradigm-analysis.md`、`ops-multi-source-ai-architecture.md`、`product-architecture-digital-ops.md`

## 1. 结论摘要

当前项目已经具备商用产品的雏形：Go 单二进制 Gateway、内嵌前端、WebSocket 控制面、Agent Runtime、Channels、Cron、RBAC、Ops 工作台、Skills/MCP/Tools 等关键部件已经存在。但它仍明显处在 demo / MVP 到产品化的中间状态，主要问题不是“有没有功能”，而是“核心数据、执行过程、权限边界、审计和部署形态是否可信、可迁移、可运维、可扩展”。

最关键的架构判断：

1. **不能只把 SQLite 换成 PostgreSQL**。当前更大的问题是数据模型分散：RBAC 有独立 `rbac.db`，统一库 `openocta.db` 已存在，Memory/Health Facts 有 SQLite 能力，但 Cron、Ops 集群、告警、数字员工、员工任务、部分 Session 仍依赖 JSON 文件或 JSONL transcript。
2. **商用版应先建立统一领域数据层**。先用 SQLite 做单机商用形态没有问题，但必须通过 repository 接口、迁移脚本、schema version、审计表和备份恢复机制把数据层收敛；之后再提供 PostgreSQL / MySQL 作为企业部署选项。
3. **产品形态应坚持“结构化运维平台 + AI 增强”**。驾驶舱、资产、告警、巡检、任务、审计必须是确定性系统；AI 负责诊断、解释、编排和建议，不能让 UI 高频读路径依赖实时 LLM 推理。
4. **商用化优先级应从演示功能转向可信闭环**。核心闭环是：资产入库 -> 告警/巡检入库 -> Agent/Runbook 执行 -> 结果结构化入库 -> 审批/处置 -> 审计/复盘 -> 指标报表。

推荐目标架构：

```text
OpenOcta Commercial
├── Experience Layer
│   ├── Control UI / Ops Workbench / ChatOps / Channels
│   └── REST + WebSocket + Webhook + OpenAI-compatible APIs
├── Application Layer
│   ├── Asset Service / Alert Service / Inspection Service / Task Service
│   ├── Agent Orchestrator / Cron Scheduler / Approval Service
│   └── RBAC / Tenant / Audit / Notification / Config Service
├── Domain Facts Layer
│   ├── assets / alerts / incidents / inspections / sessions / tasks
│   ├── health_signals / health_snapshots / run_audits / approvals
│   └── tool_invocations / model_usage / outbound_messages
├── Execution Layer
│   ├── Platform Tools / MCP Tools / Skills / Runbooks
│   └── Channels: WeCom / Feishu / DingTalk / Weixin / Slack / Telegram / ...
├── Storage Layer
│   ├── SQLite for single-node edition
│   ├── PostgreSQL for team / enterprise edition
│   ├── Object storage for transcripts, attachments, reports
│   └── Redis or embedded queue for distributed scheduling and locks
└── Operations Layer
    ├── migrations / backup / restore / metrics / logs / traces
    └── licensing / upgrade / packaging / HA deployment
```

## 2. 当前项目评估

### 2.1 已具备的基础

| 能力 | 当前状态 | 商用价值 |
|------|----------|----------|
| 单二进制 Gateway | Go 后端 + 内嵌前端，便于分发 | 适合私有化交付、试用版、单机版 |
| Control UI | Lit + Vite SPA，已有运维工作台、配置、会话、通道等页面 | 具备产品化入口 |
| Agent Runtime | 已接入 tools、skills、approval queue、sessions | 能承载智能诊断与自动化 |
| Channels | 多 IM 通道插件化雏形 | 适合企业 ChatOps |
| Cron | 定时任务服务已存在 | 可做巡检、日报、自动化触发 |
| RBAC | 已有用户、角色、权限、token 表 | 是商用安全基线的起点 |
| Ops 领域 | 集群、告警、巡检、健康信号、BCH/GBase 场景已有雏形 | 已接近目标产品方向 |
| 文档体系 | 已有产品架构、多源数据、运维 API、实施计划等文档 | 便于沉淀路线图 |

### 2.2 主要短板

| 短板 | 当前表现 | 商用风险 |
|------|----------|----------|
| 数据层混合 | SQLite、JSON、JSONL、前端 localStorage 同时存在 | 数据不可治理、迁移困难、并发一致性弱 |
| 存储边界不清 | `rbac.db` 与 `openocta.db` 并存，部分模块直接读写文件 | 无法统一备份、租户隔离、审计和升级 |
| demo 数据残留 | Ops store 初始化会写入示例集群和示例告警 | 真实客户环境中容易污染数据和信任 |
| 领域模型未统一 | Alert、Inspection、Task、Session、Health Facts 还没有完整事件生命周期 | 难形成闭环和报表 |
| 安全能力偏 MVP | 默认 admin、SHA256+salt、localStorage token、CORS 警告等 | 不满足企业安全审计 |
| 审计不完整 | 部分 RunAudit 存在，但工具调用、审批、模型调用、外发消息未统一审计 | 高风险操作不可追溯 |
| 任务编排偏轻 | Cron 与 Agent run 存在，但缺可靠队列、重试、幂等、分布式锁 | 自动化执行不稳定 |
| 多租户缺失 | 当前更像单工作区/单组织 | 商用 SaaS 或多团队私有化会受限 |
| 配置与密钥治理不足 | 配置文件与环境变量混用，凭据引用尚未形成统一 Secret 管理 | 容易泄漏或误配置 |
| 发布运维体系不足 | 需要补迁移、备份、升级、可观测、健康探针、许可证、兼容策略 | 难以长期交付 |

## 3. 商用目标与边界

### 3.1 产品版本建议

| 版本 | 目标客户 | 部署形态 | 存储建议 |
|------|----------|----------|----------|
| Community / Trial | 个人、POC、演示 | 单二进制、本机或单服务器 | SQLite + 本地文件附件 |
| Team | 小团队私有化 | 单节点服务 + 反向代理 | SQLite 或 PostgreSQL |
| Enterprise | 大客户、多团队、生产运维 | 多节点、容器化、K8s、统一认证 | PostgreSQL + Redis + 对象存储 |

关键原则：**单机版保留 SQLite，但不能让业务代码绑定 SQLite；企业版必须能切 PostgreSQL。**

### 3.2 商用化验收标准

一个可商用版本至少要满足：

| 维度 | 最低标准 |
|------|----------|
| 数据 | 核心业务数据全部入库；文件仅用于附件、报告、转录归档 |
| 安全 | 首次启动强制改默认密码；密码使用 Argon2id/bcrypt；token 可吊销；敏感配置加密或 Secret 引用 |
| 权限 | API、WebSocket、工具执行、审批、外发消息均可按权限控制 |
| 审计 | 登录、配置变更、工具调用、Agent 执行、审批、处置、外发通知全量记录 |
| 可靠性 | 后台任务支持状态机、重试、超时、取消、幂等、失败原因 |
| 可运维 | 健康检查、结构化日志、指标、trace、备份恢复、数据库迁移 |
| 可升级 | 所有 schema 有版本迁移；旧 JSON 数据可一次性或懒迁移 |
| 可集成 | Prometheus/VM、CMDB、告警源、IM、工单系统具备标准接入协议 |
| 测试 | 核心 API、存储迁移、权限、任务执行、前端关键路径有自动化测试 |

## 4. 目标领域模型

商用化不建议继续按“页面需要什么就存什么”的方式扩展，应建立稳定的领域表。

### 4.1 核心实体

| 领域 | 关键实体 | 说明 |
|------|----------|------|
| 组织与租户 | tenants, workspaces, users, roles, permissions, memberships | 企业版基础 |
| 资产 | assets, asset_relations, clusters, services, components, owners | CMDB 与拓扑基础 |
| 告警事件 | alert_events, alert_groups, incidents, incident_timeline | 告警降噪、事件生命周期 |
| 巡检健康 | inspection_runs, inspection_reports, health_signals, health_snapshots | L3 Facts 层 |
| Agent 会话 | sessions, session_messages, session_artifacts, model_usage | 从 JSONL 走向可查询 |
| 任务执行 | jobs, job_runs, run_steps, tool_invocations, retries | Cron / 手动 / Webhook 统一 |
| 审批与处置 | approvals, approval_steps, remediation_actions, rollback_records | 高风险动作闭环 |
| 通道与通知 | channel_accounts, inbound_messages, outbound_messages, delivery_receipts | ChatOps 和通知追踪 |
| 配置与密钥 | config_versions, secrets, integrations, connector_instances | 配置可审计、密钥可轮换 |
| 审计 | audit_logs, security_events | 合规基础 |

### 4.2 事件生命周期

```text
AlertEvent
  -> AlertGroup
  -> Incident
  -> Diagnosis Run
  -> Suggested Actions
  -> Approval
  -> Remediation Execution
  -> Verification
  -> Resolution
  -> Review / Report
```

商用版的 UI、API、Agent 都应围绕这个生命周期协同，而不是让告警、会话、任务、报告各自独立存在。

### 4.3 Session 与 Transcript 策略

当前 JSONL transcript 适合本地调试，但商用版需要可查询、可脱敏、可归档：

1. `sessions`：保存会话元数据、归属、状态、标题、最后活跃时间。
2. `session_messages`：保存结构化消息、角色、token、模型、时间、工具调用引用。
3. `session_artifacts`：保存附件、报告、图片、导出文件的对象存储 key。
4. `transcript_archives`：保留 JSONL 归档能力，用于兼容和导出。

迁移策略：短期保留 JSONL 作为原始归档，中期写入 DB 作为主读路径，长期 JSONL 只作为导出格式。

## 5. 数据层演进方案

### 5.1 Repository 接口先行

所有核心模块应从直接读写文件改为接口：

```go
type AlertRepository interface {
    CreateEvents(ctx context.Context, events []AlertEvent) error
    UpsertGroup(ctx context.Context, group AlertGroup) error
    ListGroups(ctx context.Context, filter AlertGroupFilter) ([]AlertGroup, error)
}
```

每个领域至少提供：

| 实现 | 用途 |
|------|------|
| SQLite repository | 单机和默认版本 |
| Postgres repository | 企业版 |
| File import adapter | 只用于迁移旧 JSON |
| In-memory fake | 单元测试 |

### 5.2 统一数据库策略

短期建议：

1. 将 `rbac.db` 合并进 `openocta.db`，或至少提供明确迁移路径。
2. 所有新表只进入 `openocta.db`。
3. 使用 `schema_migrations` 管理版本，不再依赖 `CREATE TABLE IF NOT EXISTS` 隐式变更。
4. 引入 `db/migrations` 目录，迁移文件命名如 `0001_init.sql`、`0002_alerts.sql`。
5. 启动时执行迁移，失败则阻止服务进入可写状态。

中期建议：

1. 引入数据库方言层，支持 `sqlite` 与 `postgres`。
2. SQL 通过明确 repository 封装，避免散落在 handler/service 中。
3. 所有 ID 使用稳定字符串 ID 或 UUID，避免 SQLite 自增 ID 泄漏到外部 API。

### 5.3 JSON 数据迁移顺序

| 优先级 | 当前数据 | 目标表 | 原因 |
|--------|----------|--------|------|
| P0 | ops/clusters.json | assets, clusters | 资产是一切运维数据的锚点 |
| P0 | ops/alerts.json | alert_events, alert_groups, incident_timeline | 告警必须可追溯 |
| P0 | employee_tasks/*.json | tasks, task_runs | 任务闭环和看板需要 |
| P1 | cron/jobs.json | jobs, schedules, job_runs | 自动化可靠性需要 |
| P1 | sessions.json | sessions | 会话列表和权限过滤需要 |
| P1 | transcript jsonl | session_messages, model_usage | 用量、审计、复盘需要 |
| P2 | employees/manifest.json | employees, employee_versions | 员工市场和版本管理需要 |
| P2 | config json | config_versions, integrations | 配置审计和回滚需要 |

### 5.4 文件仍然保留的边界

可以继续使用文件系统的内容：

| 类型 | 原因 |
|------|------|
| 附件、图片、导出报告 | 适合对象存储或本地 blob |
| Skill 源文件 | 类似插件包，适合文件/仓库管理 |
| Prompt 模板 | 可版本化为文件，但发布后应有 DB 元数据 |
| JSONL 导出 | 作为可移植审计归档格式 |

不应继续作为主存储的内容：

| 类型 | 原因 |
|------|------|
| 资产、告警、任务、审批、权限 | 需要查询、事务、审计、迁移 |
| Cron 状态 | 需要并发锁、运行记录、重试 |
| 会话元数据 | 需要按用户、租户、员工、场景过滤 |
| 健康分和诊断结果 | 需要趋势、报表、下钻 |

## 6. 应用层重构方向

### 6.1 从 Handler 直连存储转为 Service + Repository

目标分层：

```text
HTTP / WebSocket Handler
  -> Application Service
  -> Domain Policy / Validator
  -> Repository
  -> DB / Object Storage / External Connector
```

每个 Handler 只负责：

1. 参数解析。
2. 鉴权上下文注入。
3. 调用 application service。
4. 返回统一错误格式。

业务规则必须下沉到 service / domain policy，避免 REST 和 WebSocket 两套入口行为不一致。

### 6.2 后台任务统一模型

Cron、手动巡检、告警触发诊断、Webhook 触发 Agent，本质都是 Job Run。

建议统一为：

```text
jobs
├── id
├── type: cron | manual | webhook | alert | channel
├── trigger
├── schedule
├── status
└── policy

job_runs
├── id
├── job_id
├── status: queued | running | waiting_approval | succeeded | failed | cancelled
├── started_at / finished_at
├── actor / tenant / workspace
└── error

run_steps
├── run_id
├── step_type: llm | tool | mcp | approval | notification
├── input_ref / output_ref
├── status / duration / retry_count
└── audit_ref
```

单机版可以用 DB polling + SQLite lock；企业版切换到 Redis / Postgres advisory lock / 队列系统。

### 6.3 Agent 输出结构化

商用版必须减少从自然语言中正则抽取结论。推荐所有运维场景输出：

```json
{
  "status": "degraded",
  "score": 72,
  "confidence": 0.81,
  "summary": "...",
  "evidence": [
    {
      "source": "victoriametrics",
      "query": "...",
      "observedAt": 1780999200000,
      "value": "..."
    }
  ],
  "risks": [],
  "recommendedActions": [],
  "requiresApproval": true
}
```

自然语言报告可以由结构化结果渲染生成，而不是反过来。

## 7. 权限、安全与合规

### 7.1 身份认证

短期必须完成：

1. 首次启动强制初始化 admin 密码，不再默认 `admin/admin888` 可直接使用。
2. 密码哈希从 SHA256+salt 升级为 Argon2id 或 bcrypt。
3. token 使用 httpOnly cookie 或可配置的 Authorization token；前端 localStorage token 仅保留开发模式。
4. 增加 token 轮换、全端登出、会话列表、过期清理。
5. 登录失败锁定、审计和速率限制。

中期企业能力：

1. OIDC / SAML / LDAP。
2. 多租户 workspace membership。
3. SCIM 用户同步。
4. 细粒度 API key，用于 webhook 和系统集成。

### 7.2 授权模型

推荐权限维度：

```text
subject = user / service_account / channel_bot
action  = read / write / execute / approve / admin
object  = tenant / workspace / domain / asset / incident / job / tool / secret
scope   = all / owned / domain / asset_tag
```

高风险动作必须走策略：

| 动作 | 策略 |
|------|------|
| 执行命令、修改配置、重启服务、回滚、扩容 | 默认需要审批 |
| 查询监控、读取资产、生成报告 | 按域权限控制 |
| 外发 IM、创建工单 | 记录审计，可配置审批 |
| 修改密钥、模型配置、MCP 配置 | 仅管理员，强审计 |

### 7.3 审计模型

所有商用关键行为写 `audit_logs`：

| 字段 | 说明 |
|------|------|
| actor_type / actor_id | 用户、系统、通道机器人 |
| tenant_id / workspace_id | 租户与空间 |
| action | login、config.update、tool.execute、approval.approve 等 |
| resource_type / resource_id | 目标对象 |
| request_id / run_id / session_id | 串联调用链 |
| before / after | 配置类变更差异 |
| ip / user_agent / channel | 来源 |
| result / error | 成功或失败原因 |

## 8. 外部系统集成架构

### 8.1 Connector 注册模型

把 VM/Prometheus、CMDB、工单、日志、IM、数据库等统一成 connector instance：

```text
connector_definitions
  - key: prometheus
  - capabilities: metrics.query

connector_instances
  - id
  - tenant_id
  - key
  - name
  - endpoint
  - secret_ref
  - status
  - last_check_at
```

Platform Tools 和 MCP 都通过 connector instance 获取配置，不再直接散落读取环境变量。

### 8.2 接入分层

| 层级 | 作用 | 示例 |
|------|------|------|
| Connector | 管连接、认证、健康检查 | Prometheus、CMDB、Jira |
| Tool | 管一次动作 | query_vm_metrics、create_ticket |
| Scenario | 管业务编排 | GBase 健康巡检、Flink 诊断 |
| Agent / Runbook | 管推理和自动化步骤 | 根因分析、处置建议 |
| Facts | 管结果持久化 | health_signal、incident |

## 9. 部署与运维

### 9.1 推荐部署形态

| 形态 | 说明 |
|------|------|
| Desktop / Trial | 当前 Wails/launcher + 单机 SQLite，适合试用 |
| Single Server | systemd / Docker Compose，SQLite 或 Postgres |
| Enterprise HA | K8s + Postgres + Redis + 对象存储 + Ingress |

### 9.2 可观测性要求

必须暴露：

| 类型 | 内容 |
|------|------|
| `/healthz` | 进程存活 |
| `/readyz` | DB、队列、迁移、关键 connector 状态 |
| `/metrics` | Prometheus 指标 |
| structured logs | JSON 日志，含 request_id、run_id、tenant_id |
| traces | LLM、tool、MCP、HTTP 外调耗时链路 |

关键指标：

1. API 延迟、错误率。
2. Agent run 成功率、失败原因、平均耗时。
3. Tool/MCP 调用成功率。
4. 队列积压、重试次数。
5. 模型 token 和费用。
6. 告警降噪率、巡检覆盖率、处置闭环率。

### 9.3 备份恢复

单机版：

1. SQLite online backup。
2. `stateDir` 附件目录打包。
3. 配置和密钥导出需脱敏。

企业版：

1. Postgres PITR。
2. 对象存储版本化。
3. 定期恢复演练。
4. 升级前自动备份和回滚点。

## 10. 工程质量与发布体系

### 10.1 测试分层

| 层级 | 重点 |
|------|------|
| Unit | domain policy、repository、validator、parser |
| Integration | DB migration、API、RBAC、connector fake |
| E2E | 登录 -> 资产 -> 巡检 -> 告警 -> 诊断 -> 审批 -> 处置 |
| Browser | 关键 UI 流程、权限菜单、错误状态 |
| Migration | 旧 JSON / SQLite 数据升级 |
| Security | 认证绕过、权限越权、敏感信息泄漏 |

### 10.2 发布要求

1. 版本号遵循 semver。
2. 每个版本包含 migration、rollback note、breaking changes。
3. 配置 schema 有版本和兼容策略。
4. 企业版支持 license 文件或 license server。
5. 插件/Skill/MCP 需要兼容矩阵。

## 11. 分阶段实施计划

### Phase 0：商用化基线冻结，1-2 周

目标：停止继续堆 demo 功能，明确边界和基线。

交付：

1. 冻结核心领域对象：资产、告警、巡检、任务、会话、审批、审计。
2. 定义 `openocta.db` schema v1 草案。
3. 制定 JSON 数据迁移清单。
4. 标记所有 demo seed 数据，增加 `OPENOCTA_SEED_DEMO_DATA` 开关，生产默认关闭。
5. 明确 Community / Team / Enterprise 版本边界。

验收：

1. 新功能不得新增主业务 JSON store。
2. 所有新增 API 必须有权限和审计设计。

### Phase 1：统一数据底座，3-5 周

目标：核心业务数据入库，建立迁移机制。

交付：

1. 引入 `schema_migrations` 和 migrations 目录。
2. 合并或迁移 RBAC 到 `openocta.db`。
3. 资产、告警、员工任务、Cron jobs 入库。
4. 保留 JSON import adapter，启动时可迁移旧数据并备份。
5. repository 接口覆盖上述领域。

验收：

1. 删除/禁用 JSON 主写路径。
2. 重启后数据一致。
3. 并发写入测试通过。
4. 旧数据迁移测试通过。

### Phase 2：任务、审计与 Agent 结构化闭环，4-6 周

目标：让 AI 执行从“会话输出”变成“可审计任务”。

交付：

1. 统一 `jobs`、`job_runs`、`run_steps`。
2. 工具调用、MCP 调用、模型调用写入 `tool_invocations` / `model_usage`。
3. 巡检结果使用结构化 `inspection_reports`。
4. 告警诊断写入 incident timeline。
5. 审批队列持久化，支持批准、拒绝、超时、取消。

验收：

1. 任意一次巡检/诊断可从 UI 下钻到输入、工具、证据、输出、审批和审计。
2. 失败任务有明确状态和错误原因。
3. 重试不会重复写脏数据。

### Phase 3：安全与企业认证，3-5 周

目标：达到企业内网生产试点安全要求。

交付：

1. 首次启动初始化管理员。
2. 密码哈希升级。
3. token/cookie 策略调整。
4. OIDC / LDAP 至少落一个。
5. 权限从菜单级扩展到 API、工具、资产域。
6. Secret 管理与敏感配置脱敏。

验收：

1. 前后端越权测试通过。
2. 默认凭据和明文密码不能进入生产启动。
3. 审计能覆盖登录、配置、执行、审批、外发。

### Phase 4：企业部署与可运维，3-6 周

目标：能在客户生产环境长期运行。

交付：

1. Docker Compose 与 K8s Helm Chart。
2. Postgres repository 或至少 Postgres-ready SQL 方言验证。
3. Redis/队列或 DB lock 的任务调度策略。
4. `/readyz`、`/metrics`、结构化日志、trace。
5. 备份恢复命令。
6. 升级/回滚文档。

验收：

1. 单节点和容器部署都可自动迁移。
2. 备份恢复演练成功。
3. 关键指标可被 Prometheus 采集。

### Phase 5：产品化与商业能力，持续迭代

目标：形成可销售、可交付、可运营的产品。

交付：

1. License 机制和版本能力开关。
2. 插件/Skill 市场与签名校验。
3. 客户环境诊断包。
4. 使用量、价值指标、报表。
5. 工单系统、CMDB、监控系统的标准连接器模板。
6. 多租户/多 workspace。

验收：

1. 能支撑多个客户环境独立交付。
2. 能输出稳定性、降噪、巡检、处置效率报表。
3. 能进行插件升级和版本兼容管理。

## 12. 推荐近期任务清单

### P0 必做

1. 建立 `schema_migrations`。
2. 将 Ops clusters、alerts、employee tasks 从 JSON 主存储迁入 DB。
3. 生产默认关闭 demo seed 数据。
4. 首次启动强制修改 admin 默认密码。
5. 增加统一 `audit_logs` 表并接入登录、配置变更、工具执行。
6. 为 API 和 WebSocket 统一 request_id / session_id / run_id。

### P1 必做

1. Cron jobs 和 job_runs 入库。
2. Session 元数据入库，JSONL 转为归档。
3. Agent 输出结构化 schema。
4. Tool/MCP 调用记录入库。
5. 密钥引用和配置脱敏。
6. `/readyz`、`/metrics`。

### P2 必做

1. Postgres 适配。
2. OIDC/LDAP。
3. 多 workspace。
4. 对象存储附件。
5. Helm Chart。
6. License 和企业能力开关。

## 13. 风险与取舍

| 风险 | 建议 |
|------|------|
| 一次性大重构拖慢进度 | 先以 repository 接口包住现有逻辑，再逐领域迁移 |
| SQLite 被认为“不商用” | 明确 SQLite 是单机版能力，企业版提供 Postgres；不要让业务代码绑定 SQLite |
| AI 输出不稳定 | 所有业务闭环依赖结构化 schema 和 Facts，不依赖自然语言 |
| 权限补晚导致返工 | 从 Phase 1 开始所有表都带 tenant/workspace/owner 字段预留 |
| demo 数据污染生产 | seed 数据必须显式开关，生产默认关闭 |
| 通道/工具太多难治理 | 统一 connector instance、secret_ref、health_check、audit |

## 14. 最小商用版本定义

如果要尽快形成一个可对外试点的商用版本，建议将 MVP 商用版限定为：

1. 单租户、单 workspace。
2. SQLite 主库，但所有核心数据入 `openocta.db`。
3. 支持资产、告警、巡检、任务、审批、审计的完整闭环。
4. 支持 Prometheus/VM、CMDB CSV/API、企业微信或飞书至少一种 IM。
5. 支持管理员、运维员、只读三类角色。
6. 支持备份恢复和升级迁移。
7. 禁止默认账号密码进入生产。

这个范围足以支撑私有化 POC 和小规模生产试点；Postgres、多租户、HA、OIDC、插件市场可以作为后续企业增强。

