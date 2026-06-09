# OpenOcta 商用化任务分解与验收清单

> 日期：2026-06-09  
> 状态：初版任务拆解  
> 来源：[commercialization-architecture-plan.md](./commercialization-architecture-plan.md)  
> 状态口径：  
> - `未开始`：当前代码未形成明确实现。  
> - `部分完成`：已有雏形或局部实现，但未达到商用验收标准。  
> - `已完成`：当前实现已基本满足该任务验收标准。  
> - `阻塞`：需要外部决策、产品边界或基础设施先确认。  

## 1. 总体里程碑

| 阶段 | 目标 | 建议周期 | 完成状态 |
|------|------|----------|----------|
| Phase 0 | 商用化基线冻结，停止新增主业务 JSON store，关闭生产 demo seed | 1-2 周 | 未开始 |
| Phase 1 | 统一数据底座，核心资产、告警、任务、Cron 入库 | 3-5 周 | 部分完成 |
| Phase 2 | 任务、审计、Agent 结构化闭环 | 4-6 周 | 部分完成 |
| Phase 3 | 权限、安全、企业认证 | 3-5 周 | 部分完成 |
| Phase 4 | 企业部署、可运维、备份恢复 | 3-6 周 | 部分完成 |
| Phase 5 | 商业能力、多租户、插件市场、License | 持续迭代 | 未开始 |

## 2. Phase 0：商用化基线冻结

目标：明确商用边界，防止继续以 demo 数据和文件存储扩展核心业务。

| 编号 | 任务 | 优先级 | 依赖 | 交付物 | 验收标准 | 完成状态 |
|------|------|--------|------|--------|----------|----------|
| C0-1 | 冻结商用核心领域对象 | P0 | 无 | 领域对象清单：资产、告警、事件、巡检、任务、审批、审计、会话、通道、配置、密钥 | 文档明确每个对象的 owner、主存储、API 边界；新增功能必须归属到对象模型 | 部分完成 |
| C0-2 | 制定 `openocta.db` schema v1 草案 | P0 | C0-1 | DB schema 文档或 migration 草案 | 覆盖 assets、alerts、tasks、jobs、sessions、audit_logs、approvals；字段预留 tenant/workspace | 未开始 |
| C0-3 | 建立主业务存储准入规则 | P0 | C0-1 | 开发规范文档 | 明确资产、告警、任务、审批、权限、健康结果不得新增 JSON 主写路径 | 未开始 |
| C0-4 | 生产默认关闭 demo seed 数据 | P0 | 无 | `OPENOCTA_SEED_DEMO_DATA` 或等价配置 | 生产/默认启动不自动写入示例集群、示例告警；测试和演示可显式开启 | 未开始 |
| C0-5 | 梳理旧 JSON 数据迁移清单 | P0 | C0-1 | 迁移矩阵 | 包含 `ops/clusters.json`、`ops/alerts.json`、`employee_tasks/*.json`、`cron/jobs.json`、`sessions.json`、transcript jsonl | 部分完成 |
| C0-6 | 定义版本边界 | P1 | 产品决策 | Community / Team / Enterprise 能力矩阵 | 明确 SQLite、Postgres、多租户、OIDC、HA、License 分别属于哪个版本 | 未开始 |
| C0-7 | 定义商用验收门禁 | P0 | C0-1 | 发布检查清单 | 数据、安全、权限、审计、任务、备份、测试均有最低门槛 | 未开始 |

## 3. Phase 1：统一数据底座

目标：核心业务数据从 JSON/分散 SQLite 收敛到可迁移、可备份、可查询的数据层。

### 3.1 数据库迁移框架

| 编号 | 任务 | 优先级 | 依赖 | 交付物 | 验收标准 | 完成状态 |
|------|------|--------|------|--------|----------|----------|
| C1-1 | 建立 `schema_migrations` 表 | P0 | C0-2 | migration runner 与首个 migration | 启动时执行迁移；重复启动幂等；迁移失败服务进入只读或启动失败状态 | 未开始 |
| C1-2 | 新增 `db/migrations` 目录 | P0 | C1-1 | SQL migration 文件结构 | migration 命名、顺序、回滚策略在 README 中说明 | 未开始 |
| C1-3 | 统一 DB 初始化入口 | P0 | C1-1 | `pkg/db` 初始化流程重构 | `openocta.db` 是新表唯一入口；各模块不再自行散落建主业务表 | 部分完成 |
| C1-4 | 设计 SQLite / Postgres 方言边界 | P1 | C1-1 | repository SQL 方言说明 | 新 repository 不依赖 SQLite 特有行为；ID 不暴露自增主键 | 未开始 |

### 3.2 RBAC 数据收敛

| 编号 | 任务 | 优先级 | 依赖 | 交付物 | 验收标准 | 完成状态 |
|------|------|--------|------|--------|----------|----------|
| C1-5 | 迁移 RBAC 到 `openocta.db` | P0 | C1-1 | RBAC migration 与兼容迁移工具 | 旧 `rbac.db` 用户、角色、权限、token 可迁移；新安装只生成 `openocta.db` | 未开始 |
| C1-6 | RBAC repository 封装 | P0 | C1-5 | `UserRepository`、`RoleRepository`、`TokenRepository` | service 不直接依赖全局 `*sql.DB`；单元测试可使用 fake/in-memory | 未开始 |
| C1-7 | RBAC 数据迁移测试 | P0 | C1-5 | migration tests | 空库、旧库、有默认用户、有自定义角色四类场景通过 | 未开始 |

### 3.3 资产与 Ops 数据入库

| 编号 | 任务 | 优先级 | 依赖 | 交付物 | 验收标准 | 完成状态 |
|------|------|--------|------|--------|----------|----------|
| C1-8 | 设计资产表 | P0 | C0-2 | `assets`、`asset_relations`、`clusters` migration | 支持 domain、owner、region、components、monitor labels、connector refs、created/updated 时间 | 未开始 |
| C1-9 | 实现资产 repository | P0 | C1-8 | SQLite repository + tests | `List/Get/Create/Patch/Delete` 与现有 Ops API 行为一致；并发写测试通过 | 未开始 |
| C1-10 | 迁移 `ops/clusters.json` | P0 | C1-9 | JSON import adapter | 旧 JSON 可迁移并备份；迁移后主读写走 DB；重复迁移不产生重复数据 | 未开始 |
| C1-11 | 移除资产 JSON 主写路径 | P0 | C1-10 | Ops cluster service 重构 | `ops/clusters.json` 不再作为主存储；仅保留导入/备份用途 | 未开始 |

### 3.4 告警与事件入库

| 编号 | 任务 | 优先级 | 依赖 | 交付物 | 验收标准 | 完成状态 |
|------|------|--------|------|--------|----------|----------|
| C1-12 | 设计告警事件 schema | P0 | C0-2 | `alert_events`、`alert_groups`、`incident_timeline` migration | 支持原始告警、合并组、状态流转、ack、resolve、review、evidence | 未开始 |
| C1-13 | 实现告警 repository | P0 | C1-12 | Alert repository + tests | 告警合并、列表过滤、patch、timeline 更新与现有行为一致 | 未开始 |
| C1-14 | 迁移 `ops/alerts.json` | P0 | C1-13 | JSON import adapter | 示例/历史告警可迁移；timeline 不丢失；重复迁移幂等 | 未开始 |
| C1-15 | 关闭告警 demo seed 默认写入 | P0 | C0-4 | seed 开关接入告警初始化 | 默认空库不写示例告警；演示模式显式写入 | 未开始 |

### 3.5 员工任务与 Cron 入库

| 编号 | 任务 | 优先级 | 依赖 | 交付物 | 验收标准 | 完成状态 |
|------|------|--------|------|--------|----------|----------|
| C1-16 | 设计员工任务表 | P0 | C0-2 | `employee_tasks` 或统一 `tasks` migration | 支持 employee、domain、capability、run、workflow、evaluation、状态机字段 | 未开始 |
| C1-17 | 迁移 `employee_tasks/*.json` | P0 | C1-16 | task import adapter | 现有任务文件可迁移；列表排序、加载、删除行为兼容 | 未开始 |
| C1-18 | 设计 Cron job 表 | P1 | C0-2 | `jobs`、`schedules` migration | 支持 cron 表达式、enabled、sessionKey、agent、delivery、created/updated | 未开始 |
| C1-19 | 迁移 `cron/jobs.json` | P1 | C1-18 | Cron repository | Cron list/add/remove/update 主路径走 DB；旧 jobs.json 自动导入 | 未开始 |
| C1-20 | Cron 并发与持久化测试 | P1 | C1-19 | service/repository tests | 并发 add/remove 不丢数据；重启后 next wake 与 enabled 状态正确 | 未开始 |

### 3.6 会话元数据入库

| 编号 | 任务 | 优先级 | 依赖 | 交付物 | 验收标准 | 完成状态 |
|------|------|--------|------|--------|----------|----------|
| C1-21 | 设计 `sessions` 表 | P1 | C0-2 | sessions migration | 支持 agent、sessionKey、sessionId、title、updatedAt、origin、channel、owner | 部分完成 |
| C1-22 | 完善 session repository | P1 | C1-21 | repository + tests | `sessions.json` 不再是主写；SQLite 查询支持 storePath 兼容；多 agent 列表一致 | 部分完成 |
| C1-23 | transcript 保留归档边界 | P1 | C1-21 | transcript strategy 文档 | 明确 JSONL 短期仍作为归档，DB 作为列表/查询主路径 | 未开始 |

## 4. Phase 2：任务、审计与 Agent 结构化闭环

目标：把 Agent 执行从“聊天输出”升级为“可审计、可回放、可审批、可复盘”的任务系统。

### 4.1 统一任务执行模型

| 编号 | 任务 | 优先级 | 依赖 | 交付物 | 验收标准 | 完成状态 |
|------|------|--------|------|--------|----------|----------|
| C2-1 | 设计统一 `jobs/job_runs/run_steps` | P0 | C1-1 | migration + domain model | 手动、Cron、Webhook、告警触发、IM 触发都能映射到统一模型 | 未开始 |
| C2-2 | 实现 JobRun service | P0 | C2-1 | service + repository | 支持 queued/running/waiting_approval/succeeded/failed/cancelled 状态流转 | 未开始 |
| C2-3 | 接入 Cron run | P0 | C2-2 | Cron -> JobRun 适配 | 每次 Cron 执行都有 run 记录、开始/结束时间、错误原因 | 未开始 |
| C2-4 | 接入手动巡检 run | P0 | C2-2 | inspection -> JobRun 适配 | UI 手动触发巡检可下钻到 run、steps、结果 | 未开始 |
| C2-5 | 接入告警诊断 run | P1 | C2-2 | alert diagnosis -> JobRun 适配 | 告警组发起 AI 诊断后，incident timeline 关联 run_id | 未开始 |

### 4.2 工具、模型、MCP 调用审计

| 编号 | 任务 | 优先级 | 依赖 | 交付物 | 验收标准 | 完成状态 |
|------|------|--------|------|--------|----------|----------|
| C2-6 | 设计 `tool_invocations` 表 | P0 | C2-1 | migration | 记录 tool/mcp 名称、输入摘要、输出摘要、状态、耗时、错误、run_id、session_id | 未开始 |
| C2-7 | 设计 `model_usage` 表 | P0 | C2-1 | migration | 记录 provider、model、tokens、cost、latency、run_id、session_id | 部分完成 |
| C2-8 | Agent runtime 调用链埋点 | P0 | C2-6 | middleware/hook | LLM、Tool、MCP 调用自动写入 run_steps 和 invocation 表 | 未开始 |
| C2-9 | 工具输入输出脱敏 | P0 | C2-6 | redaction policy | 密钥、token、password、DSN 默认不落明文审计 | 未开始 |

### 4.3 结构化诊断与巡检结果

| 编号 | 任务 | 优先级 | 依赖 | 交付物 | 验收标准 | 完成状态 |
|------|------|--------|------|--------|----------|----------|
| C2-10 | 定义 InspectionReport JSON schema | P0 | C1-12 | schema 文档 + Go struct | 包含 status、score、confidence、summary、evidence、risks、recommendedActions、requiresApproval | 部分完成 |
| C2-11 | Agent 输出结构化校验 | P0 | C2-10 | parser/validator | 非法结构化输出进入 failed/degraded，不静默正则猜分 | 未开始 |
| C2-12 | 巡检结果写入 Facts | P0 | C2-10 | inspection_reports + health_signals 写入路径 | UI 高频读走 L3 Facts；报告可追溯 evidence | 部分完成 |
| C2-13 | 退役自然语言正则抽分 | P1 | C2-11 | inspection parsing 重构 | 商用路径不依赖 transcript 文本抽分；保留兼容告警日志 | 未开始 |

### 4.4 审批与处置闭环

| 编号 | 任务 | 优先级 | 依赖 | 交付物 | 验收标准 | 完成状态 |
|------|------|--------|------|--------|----------|----------|
| C2-14 | 审批队列持久化 | P0 | C1-1 | `approvals`、`approval_steps` migration | 重启不丢审批；支持 approve/reject/timeout/cancel | 部分完成 |
| C2-15 | 高风险动作策略 | P0 | C2-14 | policy 配置 | 执行命令、改配置、重启、回滚、扩容默认需要审批 | 部分完成 |
| C2-16 | 处置动作记录 | P1 | C2-14 | `remediation_actions`、`rollback_records` | 每个处置动作有操作者、审批、输入、输出、回滚信息 | 未开始 |
| C2-17 | UI 下钻执行链路 | P1 | C2-1 | run detail 页面或面板 | 从任务/告警/巡检可查看 steps、tools、evidence、approval、audit | 未开始 |

## 5. Phase 3：权限、安全与企业认证

目标：达到企业内网生产试点的安全基线。

### 5.1 身份认证

| 编号 | 任务 | 优先级 | 依赖 | 交付物 | 验收标准 | 完成状态 |
|------|------|--------|------|--------|----------|----------|
| C3-1 | 首次启动初始化管理员 | P0 | C1-5 | setup wizard 或 CLI init | 无默认 `admin/admin888` 可直接登录；首次必须设置密码 | 未开始 |
| C3-2 | 密码哈希升级 | P0 | C1-5 | Argon2id/bcrypt 实现与迁移 | 新密码使用强哈希；旧 SHA256 登录后自动升级或强制重置 | 未开始 |
| C3-3 | token 存储策略调整 | P0 | C1-6 | httpOnly cookie 或安全 token 模式 | 生产模式不依赖 localStorage token；支持 CSRF/同源策略 | 未开始 |
| C3-4 | 登录失败限制 | P0 | C1-6 | rate limit / lockout | 连续失败触发限制；安全事件写审计 | 未开始 |
| C3-5 | 登出与 token 吊销 | P0 | C1-6 | session management API | 单 token 吊销、全端登出、过期清理可用 | 部分完成 |

### 5.2 授权模型

| 编号 | 任务 | 优先级 | 依赖 | 交付物 | 验收标准 | 完成状态 |
|------|------|--------|------|--------|----------|----------|
| C3-6 | 定义 action/object/scope 权限模型 | P0 | C0-1 | 权限模型文档 + 常量 | 覆盖 read/write/execute/approve/admin 与 domain/asset/job/tool/secret | 部分完成 |
| C3-7 | REST API 权限统一 | P0 | C3-6 | middleware + route policy | Ops、RBAC、config、skills、channels、jobs API 均有权限声明 | 部分完成 |
| C3-8 | WebSocket method 权限统一 | P0 | C3-6 | method policy registry | `sessions.*`、`cron.*`、`config.*`、`skills.*` 等方法权限一致 | 部分完成 |
| C3-9 | Tool 执行权限 | P0 | C3-6 | tool policy gate | 用户无权限时 Agent 不能绕过 UI 直接执行高风险 tool | 未开始 |
| C3-10 | 资产域权限过滤 | P1 | C3-6 | domain/asset scope filter | 非 admin 只能看到授权 domain/asset 的告警、任务、巡检、会话 | 部分完成 |

### 5.3 企业认证与密钥

| 编号 | 任务 | 优先级 | 依赖 | 交付物 | 验收标准 | 完成状态 |
|------|------|--------|------|--------|----------|----------|
| C3-11 | OIDC 接入 | P1 | C3-1 | OIDC login flow | 可对接企业 IdP；用户与角色映射可配置 | 未开始 |
| C3-12 | LDAP 接入 | P2 | C3-1 | LDAP auth provider | 支持基础登录、组到角色映射 | 未开始 |
| C3-13 | Secret 管理模型 | P0 | C1-1 | `secrets` schema + secret_ref | 配置中敏感值使用引用；API 返回脱敏 | 未开始 |
| C3-14 | 配置变更审计 | P0 | C2-6 | `config_versions` + audit | 配置变更记录 before/after 摘要、操作者、时间，可回滚 | 未开始 |

## 6. Phase 4：企业部署与可运维

目标：让系统能在客户环境长期运行、升级、备份、监控。

### 6.1 部署形态

| 编号 | 任务 | 优先级 | 依赖 | 交付物 | 验收标准 | 完成状态 |
|------|------|--------|------|--------|----------|----------|
| C4-1 | 单机 systemd 部署包 | P0 | C1-1 | systemd unit + install doc | 可安装、启动、停止、查看日志、设置 stateDir | 部分完成 |
| C4-2 | Docker Compose 部署 | P0 | C1-1 | compose 文件 | 包含 openocta、可选 postgres、可选 redis；健康检查可用 | 部分完成 |
| C4-3 | Helm Chart | P1 | C4-2 | chart + values | 支持外部 Postgres/Redis/Object Storage；readiness/liveness 配置完整 | 未开始 |
| C4-4 | Postgres repository 验证 | P1 | C1-4 | Postgres CI 或集成测试 | 核心 repository 在 Postgres 下通过迁移和 CRUD 测试 | 未开始 |
| C4-5 | 分布式任务锁 | P1 | C2-2 | DB lock / Redis lock | 多实例部署时同一 job 不会重复执行 | 未开始 |

### 6.2 可观测性

| 编号 | 任务 | 优先级 | 依赖 | 交付物 | 验收标准 | 完成状态 |
|------|------|--------|------|--------|----------|----------|
| C4-6 | `/healthz` 与 `/readyz` | P0 | C1-1 | HTTP endpoints | healthz 仅检查进程；readyz 检查 DB、migration、queue、关键 connector | 部分完成 |
| C4-7 | `/metrics` | P0 | C2-1 | Prometheus metrics | API 延迟、错误率、job run、tool 调用、token 用量、队列积压可采集 | 未开始 |
| C4-8 | 结构化日志标准 | P0 | 无 | logging guideline + fields | 日志包含 request_id、run_id、session_id、user_id、tenant/workspace | 部分完成 |
| C4-9 | Trace 链路 | P1 | C2-8 | trace middleware | LLM、Tool、MCP、HTTP 外调可串联到 run/session | 部分完成 |
| C4-10 | Connector 健康检查 | P1 | C3-13 | connector status API | VM/Prometheus、CMDB、IM、工单等连接器有 last_check 和失败原因 | 未开始 |

### 6.3 备份恢复与升级

| 编号 | 任务 | 优先级 | 依赖 | 交付物 | 验收标准 | 完成状态 |
|------|------|--------|------|--------|----------|----------|
| C4-11 | SQLite 备份命令 | P0 | C1-1 | `openocta backup` CLI | 在线备份 DB 和附件目录；产物可校验 | 未开始 |
| C4-12 | SQLite 恢复命令 | P0 | C4-11 | `openocta restore` CLI | 可在空环境恢复并启动；恢复前有版本兼容检查 | 未开始 |
| C4-13 | 升级前自动备份 | P1 | C4-11 | upgrade hook | migration 前自动创建恢复点；失败可回滚 | 未开始 |
| C4-14 | Postgres 备份恢复文档 | P1 | C4-4 | ops doc | 明确 pg_dump/PITR 推荐方式和恢复演练步骤 | 未开始 |

## 7. Phase 5：产品化与商业能力

目标：形成可销售、可交付、可持续升级的商业产品。

| 编号 | 任务 | 优先级 | 依赖 | 交付物 | 验收标准 | 完成状态 |
|------|------|--------|------|--------|----------|----------|
| C5-1 | License 能力开关 | P1 | 产品决策 | license verifier + feature flags | 不同版本能力按 license 控制；离线私有化可用 | 未开始 |
| C5-2 | 多 workspace | P1 | C1-1, C3-6 | workspace schema + UI | 资产、告警、任务、会话按 workspace 隔离 | 未开始 |
| C5-3 | 多租户 | P2 | C5-2 | tenant schema + admin UI | 企业版可承载多个组织；数据查询默认带 tenant_id | 未开始 |
| C5-4 | 插件/Skill 签名校验 | P1 | C3-13 | signing + verification | 上传/安装插件可校验来源和完整性 | 未开始 |
| C5-5 | 连接器模板市场 | P2 | C3-13 | connector catalog | Prometheus/VM、CMDB、工单、日志、IM 有标准模板 | 未开始 |
| C5-6 | 客户环境诊断包 | P1 | C4-6 | support bundle command | 一键导出脱敏配置、版本、日志摘要、健康状态、迁移状态 | 未开始 |
| C5-7 | 价值指标报表 | P1 | C2-1 | dashboard/report | 展示降噪率、巡检覆盖率、处置闭环率、节省人时估算、token 成本 | 未开始 |
| C5-8 | 版本兼容矩阵 | P1 | C4-4 | compatibility doc | OpenOcta、DB schema、Skill、MCP、插件版本兼容关系明确 | 未开始 |

## 8. 跨阶段技术债清单

| 编号 | 技术债 | 影响 | 建议处理阶段 | 完成状态 |
|------|--------|------|--------------|----------|
| D-1 | `rbac.db` 与 `openocta.db` 并存 | 备份、迁移、审计割裂 | Phase 1 | 未开始 |
| D-2 | Ops clusters / alerts 使用 JSON 主存储 | 并发一致性、查询、审计不足 | Phase 1 | 未开始 |
| D-3 | Cron jobs 使用 JSON 主存储 | 调度可靠性不足 | Phase 1 | 未开始 |
| D-4 | employee tasks 使用文件存储 | 任务看板、过滤、审计不足 | Phase 1 | 未开始 |
| D-5 | transcript JSONL 是会话事实主来源 | 查询、脱敏、用量统计困难 | Phase 1 / 2 | 部分完成 |
| D-6 | 部分 demo seed 默认写入 | 污染生产数据 | Phase 0 | 未开始 |
| D-7 | 密码哈希强度不足 | 安全审计风险 | Phase 3 | 未开始 |
| D-8 | token 存在 localStorage | XSS 风险 | Phase 3 | 未开始 |
| D-9 | Agent 结果仍有文本解析路径 | 结果不稳定、不可验证 | Phase 2 | 部分完成 |
| D-10 | 工具/MCP 调用审计不统一 | 高风险动作不可追溯 | Phase 2 | 未开始 |
| D-11 | 配置与密钥混用 env/json | 凭据泄漏风险 | Phase 3 | 未开始 |
| D-12 | 缺备份恢复命令 | 生产不可运维 | Phase 4 | 未开始 |

## 9. 最小商用版本任务范围

若目标是尽快形成可对外试点的最小商用版本，建议只纳入以下任务：

| 范围 | 必选任务 |
|------|----------|
| 数据底座 | C1-1 到 C1-3、C1-8 到 C1-17、C1-21 到 C1-23 |
| demo 风险 | C0-4、C1-15 |
| 任务闭环 | C2-1 到 C2-4、C2-10 到 C2-12、C2-14 到 C2-15 |
| 安全基线 | C3-1 到 C3-7、C3-13 到 C3-14 |
| 运维基线 | C4-1、C4-2、C4-6 到 C4-8、C4-11 到 C4-12 |
| 测试门禁 | C1-7、C1-20、权限越权测试、E2E 闭环测试 |

最小商用版本不建议纳入：

| 暂缓项 | 原因 |
|--------|------|
| 多租户 | 会显著扩大数据模型和权限复杂度 |
| Helm + HA | 可在单节点私有化验证后推进 |
| 插件市场 | 依赖签名、版本兼容和安全审查 |
| License | 可在试点阶段先用交付约束替代 |
| LDAP + OIDC 双实现 | 先落一个企业认证方式即可 |

## 10. 验收测试矩阵

| 测试项 | 覆盖任务 | 验收标准 | 完成状态 |
|--------|----------|----------|----------|
| JSON 迁移测试 | C1-10、C1-14、C1-17、C1-19 | 旧数据自动导入、备份、幂等、无重复 | 未开始 |
| DB 并发写测试 | C1-9、C1-13、C1-20 | 并发创建/更新/删除无数据丢失 | 未开始 |
| 权限越权测试 | C3-7、C3-8、C3-9、C3-10 | 非授权用户无法读写、执行、审批越权资源 | 部分完成 |
| 登录安全测试 | C3-1 到 C3-5 | 默认账号不可用、失败限制、token 吊销生效 | 未开始 |
| Agent 执行链路测试 | C2-1 到 C2-12 | 一次巡检可追溯 run、steps、tool、model、evidence、facts | 未开始 |
| 审批测试 | C2-14 到 C2-16 | 高风险动作等待审批；拒绝后不执行；重启不丢审批 | 部分完成 |
| 备份恢复测试 | C4-11、C4-12 | 备份包可恢复到空环境并通过 readyz | 未开始 |
| E2E 商用闭环 | MVP 范围 | 登录 -> 建资产 -> 接告警 -> AI 诊断 -> 审批 -> 处置 -> 审计 -> 报表 | 部分完成 |

## 11. 当前状态判断

基于当前代码和文档观察，项目处于：

```text
Demo/MVP 后期
  -> 已有产品骨架、Ops 场景、RBAC、SQLite 雏形、测试雏形
  -> 尚未完成核心数据统一、审计闭环、安全基线、备份恢复、企业部署
```

最建议优先推进的 5 个任务：

1. C1-1：建立 `schema_migrations`。
2. C0-4：关闭生产默认 demo seed。
3. C1-8 到 C1-15：资产与告警入库。
4. C2-1 到 C2-8：统一任务执行和工具/模型调用审计。
5. C3-1 到 C3-5：默认账号、密码哈希、token 安全。

