# OpenOcta 多源数据与 AI 协同架构任务分解与验收跟踪

> 版本：v0.1  
> 日期：2026-06-05  
> 来源：[ops-multi-source-ai-architecture.md](./ops-multi-source-ai-architecture.md) v0.2  
> 用途：作为 Phase 1 之后研发推进、验收评审和状态跟踪的工作清单。

---

## 0. 状态约定

| 状态 | 含义 |
|------|------|
| ⬜ 待开始 | 尚未进入实现 |
| 🟡 进行中 | 已开始实现或联调 |
| ✅ 已完成 | 已实现并通过本表验收 |
| ⚠️ 阻塞 | 依赖、方案或环境阻塞 |
| ❌ 不通过 | 已验收但不满足标准 |

---

## 1. 总体验收边界

| ID | 验收项 | 标准 | 状态 | 备注 |
|----|--------|------|------|------|
| G-1 | L3 Facts 成为 UI/API 共同事实层 | 驾驶舱和域详情优先读取 `HealthSnapshot`，不直接依赖 Agent 实时推理 | ⬜ 待开始 | |
| G-2 | Cluster 仅作为连接器层 | `Cluster` 不再承载实时健康分；健康事实来自 `HealthSignal` / `HealthSnapshot` | ⬜ 待开始 | |
| G-3 | composite 分诚实展示 | 仅当 `coverage >= minCoverageForScore` 且满足 `requiredAnyOf` 时展示数字分 | ⬜ 待开始 | |
| G-4 | 必需源失败语义统一 | 必需源缺失/失败统一为 `scoreStatus: degraded`，不展示 composite 数字分 | ⬜ 待开始 | |
| G-5 | Chat 不直接写健康分 | 普通对话只能生成草稿 Report；HealthSignal 写入必须满足准入规则 | ⬜ 待开始 | |
| G-6 | GBase 样板闭环 | 无 DSN 诚实 degraded；配置 DSN 后可生成 `gbase_sql` Signal 并参与聚合 | ⬜ 待开始 | |
| G-7 | 结构化巡检替代正则抽分 | `inspection.go` transcript 正则抽分路径退役，改读结构化 `InspectionReport.score` | ⬜ 待开始 | |
| G-8 | 监控 URL 语义统一 | `metricsBaseUrl` / `vmUrlRef` 均为 Prometheus/VM base URL，不含 `/api/v1/query` | ⬜ 待开始 | |

---

## 2. Phase 0：方案冻结

目标：冻结开发边界，避免实现阶段反复变更核心语义。

| ID | 任务 | 交付物 | 验收标准 | 状态 | 备注 |
|----|------|--------|----------|------|------|
| P0-1 | 架构方案 v0.2 确认 | `docs/ops-multi-source-ai-architecture.md` | L3 schema、Policy、Scenario、Chat 准入、alerts 聚合均已定义 | ✅ 已完成 | 待最终签字 |
| P0-2 | 任务分解文档 | 本文档 | 覆盖任务、验收标准、状态跟踪 | ✅ 已完成 | |
| P0-3 | 确认首个样板域 | 评审决议 | 首个样板域为 GBase | ✅ 已完成 | |
| P0-4 | 确认 Scenario 注册方式 | 决议记录 | Phase 1 使用 YAML 或 `ops/scenarios.json`，二期再考虑 UI 可编辑 | ⬜ 待开始 | |
| P0-5 | 确认 Phase 1 开工 | 评审签字 | 明确开工日期、验收人、联调环境 | ⬜ 待开始 | |

---

## 3. Phase 1：事实层骨架

目标：先建立 L3 Facts 的最小可用骨架，让驾驶舱和域详情能读取确定性快照。

| ID | 任务 | 交付物 | 验收标准 | 状态 | 备注 |
|----|------|--------|----------|------|------|
| P1-1 | 定义 `HealthSignal` schema v1 | `src/pkg/ops/health_signal.go` 或等价文件 | 字段包含 `schemaVersion/id/runId/scenarioKey/objectType/objectId/domain/type/status/score/confidence/source/sourceKind/evidence/error/observedAt/ttlSec` | ⬜ 待开始 | |
| P1-2 | 定义 `HealthSnapshot` schema v1 | `src/pkg/ops/health_snapshot.go` 或等价文件 | 字段包含 `aggregationPolicyVersion/objectType/objectId/domain/score/scoreStatus/coverage/missingSources/presentSources/signals/observedAt` | ⬜ 待开始 | |
| P1-3 | 抽象 `HealthSignalStore` | Store interface + JSON 实现 | 业务逻辑依赖接口，不直接绑死 JSON 文件 | ⬜ 待开始 | Phase 1 JSON MVP |
| P1-4 | 新增 L3 JSON 存储 | `stateDir/ops/health_signals.json`、`health_snapshots.json` | 启动后可加载；写入后可持久化；空文件不崩溃 | ⬜ 待开始 | |
| P1-5 | 定义 `DomainHealthPolicy` | `config/ops/domain_health_policy.yaml` 或内置默认配置 | GBase policy 含 `requiredAnyOf: [gbase_sql]`、weights、`minCoverageForScore`、`coverageScope: configured` | ⬜ 待开始 | |
| P1-6 | 实现 Snapshot 聚合器 | `AggregateHealthSnapshot` | 必需源缺失输出 `degraded`；coverage 不足输出 `partial`；满足条件才输出数字分 | ⬜ 待开始 | |
| P1-7 | 实现 alerts collector | `collectAlertSignal` | `active/analyzing` 告警参与惩罚；`resolved` 不降分；无法归属 cluster 的告警不写 cluster Signal | ⬜ 待开始 | |
| P1-8 | 实现 asset_status collector | `collectAssetStatusSignal` | 资产状态可生成低权重 Signal；不得单独冒充 composite 分 | ⬜ 待开始 | |
| P1-9 | metricsBaseUrl 规范化 | normalize 函数 + 种子数据修正 | 已带 `/api/v1/query` 的旧值会被 strip，避免双拼路径 | ⬜ 待开始 | |
| P1-10 | 驾驶舱读 Snapshot 优先 | Dashboard API / UI | 有 Snapshot 时展示 `composite/partial/degraded`；无 Snapshot 时可走临时 fallback | ⬜ 待开始 | fallback Phase 2 移除 |
| P1-11 | 更新 API 文档 | `docs/ops-api.md` | 新增 L3 查询 API、字段说明、状态语义 | ⬜ 待开始 | |

### Phase 1 验收场景

| 场景 | 输入/前置条件 | 期望结果 | 状态 |
|------|---------------|----------|------|
| V1-1 GBase 无 DSN、无 VM | GBase cluster 无 `gbaseDsnRef`，无 `metricsBaseUrl` | Snapshot 为 `scoreStatus: degraded`，`missingSources` 含 `gbase_sql`，不显示数字分 | ⬜ 待开始 |
| V1-2 GBase 仅有 active 告警 | 无 DSN，有 active/analyzing GBase 告警 | 生成 `alerts` Signal；Snapshot 仍不显示 composite 数字分 | ⬜ 待开始 |
| V1-3 GBase 仅有资产状态 | 无 DSN，仅 cluster status | 生成 `asset_status` Signal；Snapshot 不显示 composite 数字分 | ⬜ 待开始 |
| V1-4 GBase 配置 DSN | `gbaseDsnRef` 可用，`gbase_sql` 采集成功 | 满足 policy 后输出 composite 分，来源为 `composite` | ⬜ 待开始 |
| V1-5 过期 Signal | Signal 超过 `ttlSec` | UI 标注过期或不参与聚合，不静默当作最新健康 | ⬜ 待开始 |

---

## 4. Phase 2：GBase Scenario + Skill 样板

目标：把 GBase 样板跑成完整 Scenario，验证 Skill、Tool、Report、Signal、Snapshot 的链路。

| ID | 任务 | 交付物 | 验收标准 | 状态 | 备注 |
|----|------|--------|----------|------|------|
| P2-1 | 新增 `ops-gbase-health` Scenario | Scenario YAML/JSON 或代码注册 | 声明 `requiredSources: [gbase_sql]`、`optionalSources: [metrics, alerts, inspection]`、tool keys、output schema | ⬜ 待开始 | |
| P2-2 | 新增 `ops-gbase-health` Skill | `skills/ops-gbase-health/SKILL.md` | 明确无 DSN 终止并报错、慢 SQL/连接错误判读、结构化输出要求 | ⬜ 待开始 | |
| P2-3 | Tool evidence 结构化 | `query_gbase_slow_sql` 等 ToolResult 扩展 | Tool 返回原始证据 + 建议解析字段，可被 Normalize 生成 Signal | ⬜ 待开始 | |
| P2-4 | 新增 `InspectionReport` schema | Go model + API 文档 | 包含 `score/status/evidence/toolRuns/errors/reportMarkdown` 等结构化字段 | ⬜ 待开始 | |
| P2-5 | Scenario 执行写 Report | Cron/按钮触发链路 | 执行后落库 `InspectionReport`，并可衍生 `gbase_sql` / `inspection` Signal | ⬜ 待开始 | |
| P2-6 | 退役巡检正则抽分 | `inspection.go` 修改 | 不再依赖 transcript 正则解析健康分；旧路径仅作为兼容或删除 | ⬜ 待开始 | |
| P2-7 | 对话触发同一 Scenario | Chat / 工作台入口 | 对话可触发 Scenario，但默认只产草稿 Report，不自动写 HealthSignal | ⬜ 待开始 | |
| P2-8 | 驾驶舱展示 GBase composite | UI + API | Scenario 跑完后驾驶舱可显示 composite 分、来源、coverage、freshness | ⬜ 待开始 | |

### Phase 2 验收场景

| 场景 | 输入/前置条件 | 期望结果 | 状态 |
|------|---------------|----------|------|
| V2-1 无 DSN 巡检 | GBase cluster 未配置 `gbaseDsnRef` | Scenario 输出明确错误；Snapshot 为 `degraded`；不展示数字分 | ⬜ 待开始 |
| V2-2 DSN 正常 | GBase DSN 可连接 | 生成结构化 `InspectionReport`、`gbase_sql` Signal、composite Snapshot | ⬜ 待开始 |
| V2-3 Tool 失败 | DSN 或 Tool 执行失败 | 记录 `error` 和 `missingSources`；不得用资产状态兜底成健康分 | ⬜ 待开始 |
| V2-4 Chat 草稿 | 用户在 Chat 询问 GBase 健康 | 仅生成聊天卡片或 Report 草稿；未确认前不写 HealthSignal | ⬜ 待开始 |
| V2-5 用户确认写入 | 用户点击确认保存结构化结果 | 通过 schema 校验后写入 Facts，`sourceKind` 非 `chat` | ⬜ 待开始 |

---

## 5. Phase 3：MCP 装配与运行时校验

目标：让 MCP 成为可插拔扩展源，但不破坏 Platform Tools 与 L3 Facts 的确定性。

| ID | 任务 | 交付物 | 验收标准 | 状态 | 备注 |
|----|------|--------|----------|------|------|
| P3-1 | Scenario 绑定 MCP | Scenario 配置支持 `mcpServerKeys` | Scenario 能声明可选 MCP，不影响无 MCP 的 GBase 样板运行 | ⬜ 待开始 | |
| P3-2 | 员工绑定 Scenario | DigitalEmployee model/UI | 员工支持 `scenarioKeys`，样板期可每 Scenario 绑定员工 | ⬜ 待开始 | |
| P3-3 | 工具冲突策略实现 | Tool registry / runtime | Platform Tool 与 MCP 同名时 Platform Tool 优先 | ⬜ 待开始 | |
| P3-4 | 必需/可选源运行时校验 | Scenario runner | 必需源失败为 `degraded`；可选源失败降低 coverage | ⬜ 待开始 | |
| P3-5 | Prometheus MCP 试点 | Prometheus MCP 配置示例 | `metrics` 可由 Platform Tool 或 MCP 采集，并统一 Normalize 为 Signal | ⬜ 待开始 | |
| P3-6 | MCP 失败审计 | Run audit | 记录 `mcpCalled[]`、错误、耗时、missingSources | ⬜ 待开始 | |

### Phase 3 验收场景

| 场景 | 输入/前置条件 | 期望结果 | 状态 |
|------|---------------|----------|------|
| V3-1 MCP 未配置 | Scenario optional MCP 缺失 | 不阻塞执行；coverage 标注缺失源 | ⬜ 待开始 |
| V3-2 必需 MCP 失败 | Scenario required source 只能由 MCP 提供 | Snapshot 为 `degraded`，不展示数字分 | ⬜ 待开始 |
| V3-3 Tool 同名冲突 | Platform Tool 与 MCP 同名 | 实际调用 Platform Tool，并记录冲突处理 | ⬜ 待开始 |

---

## 6. Phase 4：全域推广与 BCH 深场景并入

目标：在 GBase 样板稳定后，把同一模型复制到 BCH、FI、治理、数据 App。

| ID | 任务 | 交付物 | 验收标准 | 状态 | 备注 |
|----|------|--------|----------|------|------|
| P4-1 | BCH Policy + Scenario | BCH 配置与 Skill | 主信号包含 `metrics`、`jmx`、`bch_workload` | ⬜ 待开始 | |
| P4-2 | `bch_workload` Signal | Flink/Spark 适配器 | BCH 深场景不再并行游离，进入 L3 Facts | ⬜ 待开始 | |
| P4-3 | FI Policy + Scenario | FI 配置与 Skill | 主信号优先 `fi_manager`，metrics/alerts 可选 | ⬜ 待开始 | |
| P4-4 | 治理平台 Scenario | governance 配置与 Skill/MCP | 支持治理 API/MCP 输入，输出结构化 Signal/Report | ⬜ 待开始 | |
| P4-5 | 数据 App Scenario | dataapps 配置与 Skill/MCP | 支持调度成功率、SLA 告警、scheduler_api | ⬜ 待开始 | |
| P4-6 | 告警 Hook → Diagnosis | Diagnosis Scenario | 告警触发诊断并写入 `DiagnosisReport`，不直接写未经校验的分数 | ⬜ 待开始 | |
| P4-7 | 域级 Snapshot 聚合 | DomainHealthSnapshot | 域级概览显示有分集群均值、partial/degraded 数量和来源 | ⬜ 待开始 | |

### Phase 4 验收场景

| 场景 | 输入/前置条件 | 期望结果 | 状态 |
|------|---------------|----------|------|
| V4-1 BCH 多源正常 | VM/JMX/BCH workload 均有数据 | BCH cluster 生成 composite，域级 Snapshot 正常聚合 | ⬜ 待开始 |
| V4-2 FI Manager 缺失 | FI cluster 未配置 FI Manager | 必需源策略按 FI policy 输出 degraded 或 partial，不猜分 | ⬜ 待开始 |
| V4-3 告警触发诊断 | 外部 alert hook 进入 | 写入 DiagnosisReport，保留 run audit 和证据引用 | ⬜ 待开始 |

---

## 7. 横向能力与治理任务

这些任务跨阶段，进入对应阶段时同步推进。

| ID | 任务 | 交付物 | 验收标准 | 状态 | 备注 |
|----|------|--------|----------|------|------|
| X-1 | Run audit | run store / 日志 | 记录 `runId/scenarioKey/employeeId/objectId/toolsCalled/mcpCalled/signalsWritten/missingSources/durationMs/operator` | ⬜ 待开始 | |
| X-2 | RBAC 校验 | API + Scenario runner | L4 读 Facts 按域过滤；L2 触发按 `ops:inspect/ops:diagnose`；L1 按员工权限与 cluster 范围 | ⬜ 待开始 | |
| X-3 | UI 来源与新鲜度展示 | 驾驶舱/域详情组件 | 展示 `source`、`coverage`、`freshness`、`missingSources`，过期清晰标注 | ⬜ 待开始 | |
| X-4 | IM 低分推送改造 | Notify 逻辑 | 仅基于结构化 `score` 触发；`partial/degraded/unknown` 不按数字分推送 | ⬜ 待开始 | |
| X-5 | SQLite 迁移评估 | 技术评审记录 | 满足分页历史、并发写入、runId 审计或 JSON 瓶颈任一条件时启动 SQLite 规划 | ⬜ 待开始 | |
| X-6 | 测试覆盖 | Go tests + UI tests | 聚合器、collector、URL 规范化、状态展示均有测试 | ⬜ 待开始 | |

---

## 8. 当前开放确认项

| ID | 问题 | 当前倾向 | 状态 | 备注 |
|----|------|----------|------|------|
| O-1 | Scenario 注册位置 | Phase 1 用 YAML 或 `ops/scenarios.json` | ⬜ 待确认 | |
| O-2 | 驾驶舱分数语义 | 上次观测分，非实时分，默认 TTL 300s | ✅ 已确认 | |
| O-3 | `vm_health.go` 一期策略 | Snapshot 优先，同步 VM fallback 并行，Phase 2 移除 fallback | ⬜ 待确认 | |
| O-4 | 数字员工粒度 | 样板期每 Scenario 可绑员工，不强制每域一个 | ✅ 已确认 | |
| O-5 | Skill 意图路由 | Phase 2 仅员工/按钮绑定，Phase 4 再做意图路由 | ✅ 已确认 | |
| O-6 | Phase 1 验收人 | 待指定 | ⬜ 待确认 | |

---

## 9. 推进记录

| 日期 | 记录 | 结果 | 下一步 |
|------|------|------|--------|
| 2026-06-05 | 生成任务分解与验收跟踪文档 | 建立推进基线 | 确认 Scenario 注册位置与 Phase 1 开工 |

