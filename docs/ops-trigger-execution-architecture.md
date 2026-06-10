# OpenOcta 触发与执行架构方案：批量诊断、Cron 与告警统一编排

> 版本：v0.2  
> 日期：2026-06-10  
> 状态：**已冻结，Phase A 实施中**  
> 适用范围：批量对象健康评估、AI 诊断、Cron 定时巡检、告警事件响应、手动/Chat 触发  
> 关联文档：  
> - [ops-multi-source-ai-architecture.md](./ops-multi-source-ai-architecture.md)  
> - [commercialization-architecture-plan.md](./commercialization-architecture-plan.md)  
> - [commercialization-task-breakdown.md](./commercialization-task-breakdown.md)  
> - [inspection-report-schema.md](./inspection-report-schema.md)  
> - [alert-integration.md](./alert-integration.md)  
> - [product-architecture-digital-ops.md](./product-architecture-digital-ops.md)

### 评审结论（v0.2 已确认）

```text
状态：已冻结，Phase A 实施中

已确认默认：
- Work Queue 首期实现：进程内 worker + SQLite 表（work_plans / work_tasks）
- L2 全局并发默认值：8（区间 5～10，留余量）
- Cron 父 Run 完成语义：等待全部子任务完成或超时；超时记 partial
- partial 是否算 cron 成功：lastStatus=partial，且不累加 consecutiveErrors（UI 区分展示）
- Reduce 域级汇总 LLM：默认关闭，仅管理报告场景开启
- 首个批量样板域：Flink 作业（ops-flink-health）

仍需评审拍板（高风险，见 §6.3 / §10 Phase A）：
- 父 Run 等待子任务的实现：轮询子 Run 状态 vs 事件回调
- SQLite 队列的崩溃恢复策略（租约 / reclaim）
- 多租户 scope 是否首期即落 tenantId 列
```

---

## 1. 背景与问题陈述

### 1.1 两类本质不同的工作

平台未来将面对大量**批量对象**场景（例如 500 个 Flink 作业、数百个 Spark 任务、多集群 GBase 实例）。同时，运维响应仍依赖**告警事件**驱动。二者不应共用同一套「直接调 Agent」的执行路径。

| 类型 | 特征 | 典型触发 | 规模示例 |
|------|------|----------|----------|
| **巡检 / 对账** | 全量或抽样，可延迟，需可重复、可对比 | Cron | 500 Flink 作业 / 小时 |
| **事件响应** | 单点或少量对象，要求低延迟、高精度 | 告警 Webhook | 1 个作业反压、1 条慢 SQL |
| **人工追问** | 上下文已明确，按需深度分析 | UI / Chat | 单对象根因 |

若将 Cron 与告警都建模为「起一个 Agent Turn」，在批量规模下会出现：

1. **无并发治理**：到期任务各起 goroutine，配置项 `maxConcurrentRuns` 尚未落地。
2. **完成语义失真**：Cron `Run()` 在 `chat.send` 返回后即记 `ok`，真实 Agent 仍在异步执行。
3. **幂等粗糙**：固定 `idempotencyKey: cron:{jobId}`，长任务期间重触发返回 `in_flight` 或被跳过；且幂等键未绑定**调度计划时刻**，重试时易因 `time.Now()` 漂移生成不同 key。
4. **成本不可控**：500 对象 × 每小时 × LLM = 不可接受的 token 与延迟。
5. **Facts 与 Transcript 割裂**：UI 应读结构化 Facts，而非依赖会话文本抽分。

本方案目标：

> **将「何时触发」与「如何执行」解耦，建立分层诊断与统一 Work Queue，使 Cron（定时对账）与 Alert（事件反应）在同一套编排模型下健壮扩展，AI 仅用于高价值环节。**

### 1.2 与现有资产的关系

本方案**不推翻** [ops-multi-source-ai-architecture.md](./ops-multi-source-ai-architecture.md) 中的 L1–L4 分层、L3 Facts、`OpsScenario`、数字员工定位；而是在其之上补齐**触发适配层**与**批量执行层**，并修正 Cron 与 Agent 的耦合方式。

| 已有能力 | 当前角色 | 本方案中的位置 |
|----------|----------|----------------|
| `cron.Service` | 定时器 + 直接 `Run()` | 降级为 **Trigger Adapter**，只提交 WorkPlan |
| `ops.RunScenario` | L1 无 LLM 巡检 | **L1 执行器**（可扩展批量参数） |
| `computeFlinkJobAnalysis` 等规则 | API 内本地打分 | **L0 规则引擎**（Collector 输出） |
| 告警合并 + `MatchAlert` | 事件入口 + 路由员工 | **Alert Trigger Adapter** |
| `job_runs` / `run_steps` | 统一审计 | 扩展父子 Run、L0/L1/L2 Step |
| `InspectionReport` / `HealthSignal` | 结构化事实 | L0 写 Signal，L1/L2 写 Report |
| 数字员工 + Skill | L2 推理 | 经 Queue 调度，非 Cron 直连 |

---

## 2. 业界实践与设计理念

参考 Google SRE（**symptom-based alerting + scheduled reconciliation**）、Datadog Watchdog / New Relic AI、PagerDuty Event Orchestration、Kubernetes Controller（**level-triggered 对账 + edge-triggered 事件**）：

| 实践 | 在本平台的映射 |
|------|----------------|
| Scheduled reconciliation | Cron → 全量 L0 采集 + 规则打分，发现「慢变坏」 |
| Event-driven reaction | Alert → 跳过全量，对已定位对象优先 L2 |
| Tiered analysis | L0 规则（便宜）→ L1 工具链（中）→ L2 AI（贵） |
| Fan-out / fan-in | 批量 Map 异常对象，Reduce 域级汇总（可选） |
| Dedup + cooldown | 同对象短时内不重复 L2；告警可覆盖冷却 |

### 2.1 核心原则

1. **Trigger 无关性**：Cron、告警、手动、Chat 只产生统一的 `WorkPlan`，执行逻辑一处维护。
2. **批量默认无 LLM**：全量健康评估走 L0；AI 仅用于异常子集或事件单点。
3. **Facts 优先于 Transcript**：UI / API 高频读 `health_snapshots`；transcript 供审计与追问。
4. **可观测与可追责**：每个对象可回答：谁触发、哪层诊断、证据来源、run_id 链路。
5. **失败语义诚实**：缺配置 / 缺数据源 → `degraded` + `missingSources[]`，禁止静默猜分（与 GBase Skill 一致）。

---

## 3. 目标架构：Trigger → Plan → Execute → Facts

```text
┌─────────────────────────────────────────────────────────────────┐
│                        触发层 Trigger                            │
│   Cron          Alert Webhook       Manual/UI        Chat       │
└────────────┬──────────────┬──────────────┬──────────────┬──────┘
             │              │              │              │
             ▼              ▼              ▼              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     编排层 Planner / Router                      │
│   去重 · 合并 · 冷却 · Scenario 解析 · EscalationPolicy          │
│   输出：WorkPlan（含 scope、tiers、priority、idempotencyKey）     │
└────────────────────────────┬────────────────────────────────────┘
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                   执行层 Executor + Work Queue                   │
│   L0 Collector + RuleEngine   （批量、无 LLM）                    │
│   L1 Scenario Runner          （平台 Tools、无 LLM）              │
│   L2 AI Diagnosis             （数字员工 + Skill + Tools）        │
│   全局/域级并发池 · 优先级 · 超时 · 父子 Run                      │
└────────────────────────────┬────────────────────────────────────┘
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                        事实层 L3 Facts                           │
│   health_signals · health_snapshots · inspection_reports         │
│   job_runs · run_steps · tool_invocations · model_usage          │
└─────────────────────────────────────────────────────────────────┘
```

### 3.1 TriggerEnvelope（统一触发载荷）

所有入口归一化为：

```json
{
  "triggerType": "cron | alert | manual | chat_intent | webhook",
  "triggerRef": "job-inspect-flink | alert-group-uuid | user-123",
  "scenarioKey": "ops-flink-health",
  "scope": {
    "objectType": "job",
    "objectIds": ["*"],
    "clusterId": "prod-a",
    "domain": "hadoop"
  },
  "priority": "low | normal | high",
  "idempotencyKey": "cron:job-inspect-flink:2026-06-10T09"
}
```

| 字段 | 说明 |
|------|------|
| `triggerType` | 与 `job_runs.trigger_type` 对齐，扩展 `chat_intent` |
| `triggerRef` | 可追溯的外部引用（cron job id、alert group id 等） |
| `scenarioKey` | 映射 `OpsScenario`，决定 tiers 与工具链 |
| `scope` | 批量时 `objectIds: ["*"]` 表示域内全量；告警通常为单对象 |
| `priority` | 队列排序：`alert > manual > cron` |
| `idempotencyKey` | 绑定**调度计划时刻**（非 `time.Now()`），含时间窗口；同一次调度的重试必须复用同一 key，避免长任务 blocking 或重复入队 |

### 3.2 WorkPlan（编排输出）

Planner 根据 `OpsScenario` + `EscalationPolicy` 生成：

```json
{
  "planId": "plan-uuid",
  "parentRunId": "run-uuid",
  "steps": [
    { "tier": "L0", "action": "collect_and_score", "parallelism": 1 },
    { "tier": "L2", "action": "ai_diagnose", "objectIds": ["job_a", "job_b"], "maxConcurrency": 5 }
  ],
  "escalation": {
    "from": "L0",
    "to": "L2",
    "when": "score < 70 OR restarts > 0 OR isBP",
    "cooldownMs": 3600000,
    "maxTargetsPerRun": 50
  }
}
```

---

## 4. 分层诊断模型（L0 / L1 / L2）

每个 `OpsScenario` 显式声明三层能力，**禁止**默认全量走 LLM。

| 层级 | 名称 | 实现 | 成本 | 典型场景 |
|------|------|------|------|----------|
| **L0** | Collect + Score | Go Collector + 批量 PromQL/API + 规则引擎 | 极低 | 500 Flink 作业 / 小时全量打分 |
| **L1** | Structured Inspection | `RunScenario`（现有原生路径，平台 Tools） | 低 | 集群级巡检、单对象补采 |
| **L2** | AI Diagnosis | 数字员工 + Skill + Tools（`chat.send` 或隔离 Run） | 高 | 异常子集根因、告警直达、人工追问 |

### 4.1 L0：批量采集与规则打分

**职责**：拉取指标 / 状态 → 确定性规则 → 写入 `health_signals`（per object）与域级 `health_snapshots`。

**示例（Flink）**：

- 1～N 次批量 PromQL（按 `job_id` label 聚合 lag、restart、反压）。
- 规则引擎：`computeFlinkJobAnalysis`（已有，需从 Mock 接到真实 Collector）。
- 输出：每作业一条 Signal；域内聚合一条 Snapshot（健康分布、Top 风险）。

**不写**完整 `InspectionReport`（除非策略要求）；L0 产出足以驱动驾驶舱与升级决策。

> **对现有 Facts 的影响（需在 schema 演进中处理）**：当前 `HealthObjectCluster = "cluster"` 是唯一对象类型，`AggregateHealthSnapshot` 与 `DomainHealthPolicy` 均强绑 `Cluster`。L0 引入 `objectType=job` 的 **per-object signal**（每作业一条）+ **域级聚合 snapshot**（健康分布、Top 风险）后，需要：
> 1. 新增对象类型常量（如 `HealthObjectJob`），`health_signals` 增加 `(object_type, object_id)` 复合索引；
> 2. 扩展或旁路 `AggregateHealthSnapshot`，支持「object 级聚合 → 域级分布快照」而非仅 cluster 级；
> 3. 避免 500 作业逐条调用 cluster 聚合路径导致的 N 次全表扫描。

### 4.2 L1：结构化巡检（无 LLM）

沿用 `ops.RunScenario`：执行 `platformToolKeys`，校验 `requiredSources` / `optionalSources`，`PersistInspectionFacts`。

适用于：

- 集群级日巡检（Hadoop / FI / GBase）。
- L0 发现异常后的**单对象补采**（例如再查该作业详细指标）。

### 4.3 L2：AI 诊断（有 LLM）

**输入必须是结构化上下文**，而非「请自己去查 500 个对象」：

```json
{
  "objectId": "job_risk_calc",
  "objectType": "job",
  "score": 42,
  "penalties": [{"item": "反压", "deduction": 20}],
  "metricsEvidence": { "lagTrend": 800, "restarts": 1 },
  "triggerType": "alert",
  "question": "根因与处置建议"
}
```

输出：`InspectionReport` / `DiagnosisReport`，校验后写入 Facts，关联 `job_runs`。

### 4.4 OpsScenario 扩展字段（建议）

在现有 [ops-multi-source-ai-architecture.md](./ops-multi-source-ai-architecture.md) §4.5 基础上扩展：

```yaml
key: ops-flink-health
domainKey: hadoop
objectType: job                    # cluster | job | db_instance | queue
batchMode: fan_out                 # single | fan_out | sample
executionTiers:
  L0:
    collector: flink_metrics_batch
    ruleEngine: flink_score_v1
  L1:
    platformToolKeys: [query_vm_metrics]
  L2:
    skillIds: [ops-flink-diagnosis]
    employeeIds: [emp_bch_diagnose]
    maxConcurrency: 5
escalationPolicy:
  L0_to_L2: "score < 70 OR restarts > 0 OR isBP"
  cooldownMs: 3600000
  maxL2PerRun: 50
triggerTypes: [cron, alert_hook, manual, chat_intent]
```

| 字段 | 含义 |
|------|------|
| `batchMode` | `fan_out`：scope 内多对象；`sample`：按比例抽样 |
| `executionTiers` | 各层启用组件；未声明的层不执行 |
| `escalationPolicy` | L0 完成后筛选进入 L2 的条件与上限 |
| `maxL2PerRun` | 单次 Cron 父 Run 最多拉起 L2 子任务数，防止雪崩 |

---

## 5. Cron 与告警：分工与统一入口

### 5.1 对比

| 维度 | Cron（定时对账） | Alert（事件反应） |
|------|------------------|-------------------|
| **目的** | 发现慢变坏、SLA 对账、补漏 | 响应突然变坏 |
| **对象范围** | 常全量 / 抽样（`objectIds: ["*"]`） | 通常已定位单对象或关联组 |
| **默认执行路径** | L0 →（条件）L2 | 轻量 L0 或跳过 → **优先 L2** |
| **延迟容忍** | 分钟～小时 | 秒～分钟 |
| **优先级** | `low` / `normal` | `high` |
| **去重** | schedule 窗口 + `idempotencyKey` | fingerprint + 合并窗口（现有 15s / 20 条） |

### 5.2 Cron 路径（目标行为）

```text
Cron due
  → TriggerRouter 构建 TriggerEnvelope
  → Planner 生成 WorkPlan（通常含 L0；L2 仅 escalation）
  → 入队 Work Queue
  → Worker 执行 L0 批量
  → 按 escalationPolicy 筛选异常 objectIds
  → 为每个异常对象 enqueue L2 子任务（受 maxConcurrency / cooldown 约束）
  → 父 JobRun 在子任务完成或超时后标记终态
  → 更新 cron job state（lastRunAtMs、lastStatus、nextRunAtMs）
```

**修正点**：`lastStatus: ok` 表示 **WorkPlan 终态完成**，而非 `chat.send` 已发出。

### 5.3 告警路径（目标行为）

```text
Alert Webhook
  → fingerprint 合并（现有 hooks 逻辑）
  → MatchAlert / TriggerRule → scenarioKey + employeeId
  → TriggerEnvelope（scope 已含 objectId，priority=high）
  → 若同 object 5min 内已有 running L2：merge 或 skip（可配置）
  → 直接 L2（或 L1+L2）；不扫描全量 500 作业
  → StartAlertDiagnosisRun（已有）+ 子 Run 审计
  → 完成后 SyncAlertDiagnosisAfterChat / Facts 回写
```

告警**可覆盖** L2 冷却时间；Cron 触发的 L2 遵守冷却。

### 5.4 手动与 Chat

- **手动巡检（UI）**：`triggerType=manual`，可指定单对象；默认 L1，用户可选「深度诊断」升 L2。
- **Chat intent**：解析运维上下文后映射 `scenarioKey`；单对象 L2，结果 `PersistInspectionFacts`（现有 chat 路径需与 WorkPlan 对齐）。

---

## 6. 执行引擎：Work Queue 与并发治理

### 6.1 现状缺口

| 缺口 | 现状 | 目标 |
|------|------|------|
| 并发上限 | `maxConcurrentRuns` 仅在 config schema | 实现 L0/L1/L2 分级上限 |
| 任务风暴 | 每个 due job `go Run()` | 入队 + worker 池 |
| 重叠执行 | `RunningAtMs` 仅清除未设防 | `scenarioKey+objectId` 锁或 merge |
| 父子关系 | 扁平 `job_runs` | 父 Run + Task Run（子） |
| 异步完成判定 | `RunCronChat` fire-and-forget，`status` 默认 `ok` | 父 Run 等待子任务终态后落终态（见 §6.3） |
| 崩溃恢复 | 进程重启后 `running` 任务永久悬挂 | `work_tasks` 租约 + reclaim，重启回收 in-flight |

### 6.2 队列能力（首期建议）

| 能力 | 说明 |
|------|------|
| **全局 L2 并发池** | 默认 5～10，可配置 `cron.maxConcurrentL2Runs` |
| **域级限流** | 如 Flink L2 ≤ 5，GBase L2 ≤ 3 |
| **优先级** | `alert > manual > cron` |
| **幂等** | `idempotencyKey` + 状态机，避免重复入队 |
| **超时** | L0 父任务 30min；L2 单对象 10min；父 Run partial 策略可配置 |
| **持久化** | 首期 SQLite 表 `work_plans` / `work_tasks`；进程内 worker；后期可换 Redis/NATS |
| **崩溃恢复** | `work_tasks` 带 `lease_until` 租约；启动时 reclaim 超租约的 `running` 任务；幂等键保证重投不重复执行（**SQLite 方案必做项**） |

### 6.3 JobRun 父子模型

```text
JobRun (父)  id=run-parent, trigger=cron, jobId=job-inspect-flink
  RunStep: L0_collect     status=succeeded, output={ "objects": 500 }
  RunStep: L0_score       status=succeeded, output={ "anomalies": 23 }
  TaskRun (子) × ≤50
    JobRun id=run-child-1, trigger=escalation, objectId=job_risk_calc, tier=L2
    JobRun id=run-child-2, ...
```

父 Run 终态：

- `succeeded`：L0 成功且 L2 子任务全部完成（或 anomalies=0 无需 L2）。
- `partial`：L0 成功但部分 L2 失败/超时（默认 `lastStatus=partial`，不累加 `consecutiveErrors`）。
- `failed`：L0 失败。

#### 异步完成判定（高风险，评审拍板）

现状 `RunCronChat` 是 fire-and-forget，父 Run 要在「所有 L2 子任务终态」后才落终态，**不是简单加队列就能解决**，需显式的子→父状态聚合机制。两种候选：

| 方案 | 机制 | 取舍 |
|------|------|------|
| **A. 轮询子 Run 状态** | 父任务持有 `childRunIds[]`，worker 周期性查 `job_runs` 子状态，全部终态或父超时 → 落终态 | 实现简单、无新依赖；有轮询延迟与额外查询 |
| **B. 事件回调** | 子 Run 完成时发事件/写 `work_tasks.parent_done_signal`，父 Run 监听聚合 | 实时性好；需事件总线或 DB 触发，复杂度高 |

**首期建议方案 A**（轮询间隔 ~5s + 父 Run 超时兜底），避免引入事件总线。无论哪种，必须保证：父超时后对未完成子任务标记 `timeout` 并仍落父 `partial`，不得无限等待。

---

## 7. 批量场景下的 AI 最佳实践

### 7.1 Map-Reduce

| 阶段 | 做法 |
|------|------|
| **Map** | 每个异常对象一次短 L2（输入为 L0 已算好的 score、penalties、metricsEvidence） |
| **Reduce**（可选） | 域级汇总 Run：输入 N 份结构化 Report JSON，输出趋势与 Top 风险（一次 LLM） |

### 7.2 禁止反模式

| 反模式 | 后果 |
|--------|------|
| 单 Prompt 塞 500 对象 | 上下文溢出、工具链爆炸 |
| 500 个 Cron Job 各触发 Agent | goroutine / LLM 风暴 |
| Cron 直连 `chat.send` 表完成 | 状态失真、无法批量审计 |
| 无冷却重复 L2 | 成本翻倍、同对象结论抖动 |

### 7.3 预算与冷却

| 策略 | 建议默认值 |
|------|------------|
| 同对象 L2 冷却（Cron 路径） | 1h |
| 告警覆盖冷却 | 是（`priority=high`） |
| 单次父 Run 最大 L2 数 | 50 |
| 域级每日 L2 预算 | 可配置 `maxAIDiagnosesPerDay` |

---

## 8. 示例：500 Flink 作业每小时评估

```text
08:00 Cron 触发 job-inspect-flink（schedule: 0 * * * * 或 everyMs: 3600000）
  │
  ├─ L0: flink_metrics_batch
  │     PromQL 批量查询 → 500 jobs 指标
  │     flink_score_v1 规则 → 500 health_signals
  │     域级 health_snapshot（healthy: 477, warning: 18, critical: 5）
  │
  ├─ Escalation: 23 jobs 满足 score<70 OR restarts>0 OR isBP
  │     冷却过滤 → 剩余 15 jobs 需 L2
  │     maxL2PerRun=50 → 全部入队
  │
  └─ L2: 15 个子 Run（并发 5）
        输入结构化 penalties + metrics
        输出 InspectionReport → inspection_reports + 更新 signals
        父 Run succeeded（duration ~8min）
```

**告警并行场景**：10:23 某作业反压告警 → 跳过 L0 全量 → 单对象 L2（高优先级，覆盖冷却）→ 与 Cron 子 Run 共享并发池，告警优先。

---

## 9. 与现有代码映射及改造清单

### 9.1 模块改造

| 模块 | 改造项 |
|------|--------|
| `pkg/cron/service.go` | `Run()` 改为提交 WorkPlan；定时循环只负责 Trigger |
| `pkg/gateway/http/server.go` | `RunCronChat` 移至 L2 Executor，由 Queue 调用 |
| `pkg/ops/scenario.go` | 扩展 `batchMode`、`executionTiers`、`escalationPolicy` |
| `pkg/ops/scenario_runner.go` | 拆出 L0 Collector 接口；L1 保留 |
| `pkg/ops/bch_*.go` | `computeFlinkJobAnalysis` → L0 rule engine；接真实 metrics |
| `pkg/gateway/http/hooks.go` | 告警入口走 TriggerRouter，非直接 Agent |
| `pkg/automation/triggers.go` | 增加 `scenarioKey` 映射 |
| `pkg/jobrun` | 父子 Run、TaskRun、`trigger_type=escalation` |
| `pkg/config` | 落地 `maxConcurrentL2Runs`、冷却、预算 |
| UI Cron 页 | 展示父 Run + 子 Run 下钻；lastStatus 语义与后端一致 |

### 9.2 数据表（建议新增）

| 表 | 用途 |
|----|------|
| `work_plans` | 计划快照、状态、父 run_id；预留 `tenant_id`（可空） |
| `work_tasks` | 队列任务：tier、object_id、priority、idempotency_key、`lease_until`（租约/reclaim）、`tenant_id` |
| `job_runs.parent_run_id` | 父子关联（或 `task_id` 复用） |
| `health_signals` | 新增 `object_type=job` 数据；建 `(object_type, object_id)` 复合索引 |

**schema 注意事项**：
- **多租户前置**：`work_plans` / `work_tasks` / Facts 首期即预留 `tenant_id` 列（可空），避免后期二次 migration（对应开放问题 #4）。
- **聚合路径**：`AggregateHealthSnapshot` 当前强绑 `Cluster` + `DomainHealthPolicy`，支持 `objectType=job` 需扩展或旁路（见 §4.1）。

详见 [openocta-db-schema-v1.md](./openocta-db-schema-v1.md) 演进时补充 migration。

### 9.3 配置项（建议）

```json
{
  "cron": {
    "enabled": true,
    "maxConcurrentL2Runs": 8,
    "maxL2PerParentRun": 50,
    "defaultL2CooldownMs": 3600000,
    "parentRunTimeoutMs": 1800000
  },
  "ops": {
    "escalation": {
      "ops-flink-health": {
        "L0_to_L2": "score < 70 OR restarts > 0 OR isBP",
        "maxL2PerRun": 50
      }
    }
  }
}
```

---

## 10. 分阶段落地计划

### Phase A — 统一触发与队列骨架（P0）

| 项 | 交付 | 验收 |
|----|------|------|
| A1 | `TriggerRouter` + `TriggerEnvelope` + `WorkPlan` 类型 | Cron / Manual 可走同一入口 |
| A2 | SQLite `work_plans` / `work_tasks` + 进程内 worker | 入队、出队、优先级可测；**重启后能 reclaim 未完成任务，无僵尸 `running`** |
| A3 | 实现 `maxConcurrentL2Runs` | 压力测试不超上限 |
| A4 | Cron `lastStatus` 与父 Run 终态绑定（**含异步等待子任务，见 §6.3 方案 A**） | 不再「send 即 ok」；父超时落 `partial` 不悬挂 |
| A5 | 父子 `job_runs` 模型 | UI / API 可查子 Run |
| A6 | 幂等键绑定调度计划时刻 | 同次调度重试复用同 key，不重复执行 |

### Phase B — L0 批量样板：Flink（P0）

| 项 | 交付 | 验收 |
|----|------|------|
| B1 | `ops-flink-health` scenario 定义 | 文档 + Go 注册 |
| B2 | `flink_metrics_batch` Collector | 对接 VM/Prom；非 Mock 数据路径。**明确：指标命名、按 `job_id` label 聚合维度、500 作业单次批量查询还是分批** |
| B3 | L0 写 `health_signals`（per job）+ 域级 `health_snapshots` | 驾驶舱可读；新增 `object_type=job` 与索引 |
| B4 | 单条 Cron `job-inspect-flink` 触发 L0 | 500 级规模压测 < 5min（指标侧依赖环境）；聚合无 N 次全表扫描 |

### Phase C — 条件升级 L2（P1）

| 项 | 交付 | 验收 |
|----|------|------|
| C1 | `escalationPolicy` 配置化 | Cron 仅升级异常子集 |
| C2 | L2 冷却 + 告警覆盖 | 同对象 1h 内不重复；告警可打断 |
| C3 | 告警路径接入 TriggerRouter | 与 Cron 共享 Queue，告警优先 |
| C4 | Map-Reduce 可选域级汇总 | 23 份 Report → 1 份域摘要 |

### Phase D — 多场景复制（P1）

| 场景 | objectType | 说明 |
|------|------------|------|
| Spark 作业 | `job` | 复用 L0 批量 + L2 升级 |
| YARN 队列 | `queue` | 队列级 L0，异常队列 L2 |
| GBase 实例 | `db_instance` | 连接/慢 SQL 批量 |
| DataApp 管道 | `pipeline` | 调度失败批量 |

---

## 11. 成功标准（Release Gates）

1. **健壮性**：500 对象 Cron 触发不产生无界 goroutine；L2 并发可配置且可观测；**进程重启后无悬挂 `running` 任务（reclaim 生效）**。
2. **正确性**：Cron `lastStatus` 与父 Run 终态一致；Facts 与 run 可追溯关联。
3. **成本**：全量巡检 LLM 调用量 ≈ 异常对象数，而非对象总数。
4. **事件响应**：告警 P95 入队延迟 < 5s（不含 LLM 执行时间）。
5. **产品**：驾驶舱健康分布来自 L0 Facts；单对象可下钻到 L2 Report 与 evidence。

---

## 12. 开放问题（评审议程）

| # | 议题 | 默认建议 | 待评审确认点 |
|---|------|----------|--------------|
| 1 | Work Queue 首期 SQLite only vs 预留 `queue.backend=redis` | SQLite only，但 worker/queue 以接口抽象，便于后期替换 | 是否值得首期就抽象接口成本 |
| 2 | 父 Run `partial` 是否算 Cron 成功 | `lastStatus=partial`，不累加 `consecutiveErrors`，UI 区分 | 告警/SLA 统计是否按 partial 计失败 |
| 3 | L0 失败时是否仍尝试 L2（告警路径除外） | 否（L0 失败 → 父 Run `failed`，不升级 L2） | 是否需要「L0 失败仍对已知异常对象兜底 L2」 |
| 4 | 多租户 `scope` 是否必须带 `tenantId` | 首期表即预留 `tenant_id` 列（可空），逻辑后置 | 单租户部署是否完全省略 |
| 5 | Reduce 域级汇总 LLM 是否默认开启 | 默认关闭，仅管理报告场景开启 | 驾驶舱是否需要每次汇总摘要 |
| 6 | 父 Run 等待子任务实现（§6.3） | 方案 A 轮询（~5s + 超时兜底） | 实时性要求是否需上事件回调 |

---

## 13. 交叉引用与术语

| 术语 | 定义 |
|------|------|
| **Trigger** | 产生 `TriggerEnvelope` 的入口，不含执行逻辑 |
| **WorkPlan** | Planner 输出的分层执行计划 |
| **L0 / L1 / L2** | 采集规则层 / 工具巡检层 / AI 诊断层 |
| **Escalation** | L0 结果筛选后进入 L2 的策略 |
| **Facts** | L3 `health_signals`、`health_snapshots`、`inspection_reports` |

相关实现文件（当前基线）：

- `src/pkg/cron/service.go` — Cron 定时与 `Run()`；默认 `job-inspect-flink`
- `src/pkg/ops/flink_collector.go` — Flink L0 批量采集与 Facts 写入
- `src/pkg/workqueue/planner.go` / `executor.go` — L0-only plan 与 `RunFlinkHealthL0`
- `src/pkg/ops/escalation.go` — Flink L0→L2 升级策略与结构化 L2 上下文
- `src/pkg/workqueue/escalation.go` — L0 完成后条件入队 L2 子任务
- `src/pkg/workqueue/trigger_alert.go` — 告警 `TriggerEnvelope`（`priority=high`）
- `src/pkg/ops/domain_reduce.go` — Map-Reduce 域级汇总（规则默认，可选 LLM）
- `src/pkg/workqueue/reduce.go` — L2 完成后条件入队 `domain_reduce` 任务
- `src/pkg/ops/scenario_runner.go` — L1 `RunScenario`
- `src/pkg/ops/alert_jobrun.go` — 告警 JobRun
- `src/pkg/gateway/http/hooks.go` — 告警合并与触发
- `src/pkg/automation/triggers.go` — 告警路由规则
- `src/pkg/jobrun/` — 统一 Run 审计

---

## 14. Phase A 实施进度（代码）

| 项 | 状态 | 实现位置 |
|----|------|----------|
| A1 TriggerEnvelope / WorkPlan / Planner | 已完成 | `pkg/workqueue/types.go`, `planner.go`, `pkg/cron/trigger.go` |
| A2 work_plans / work_tasks + worker + reclaim | 已完成 | migration `010_work_queue.sql`, `pkg/workqueue/repository.go`, `service.go` |
| A3 maxConcurrentL2Runs | 已完成 | `pkg/workqueue/config.go`, claim 时 L2 计数 |
| A4 Cron lastStatus 绑定父 Run（含 L2 等待） | 已完成 | `pkg/cron/service_workqueue.go`, `runnotify`, `chat.go` defer |
| A5 parent_run_id 父子 Run | 已完成 | migration `010`, `pkg/jobrun` |
| A6 幂等键绑定调度时刻 | 已完成 | `cron:{jobId}:{scheduledAtMs}` |
| B1 ops-flink-health scenario | 已完成 | `pkg/ops/scenario.go`, `inspection_facts.go` |
| B2 flink_metrics_batch Collector | 已完成 | `pkg/ops/flink_collector.go`, `flink_metrics_vm.go`, `flink_score.go` |
| B3 per-job signals + 域快照 | 已完成 | `object_type=job`, `hadoop:flink` domain snapshot |
| B4 job-inspect-flink Cron（每小时） | 已完成 | `pkg/cron/service.go` `ensureDefaultFlinkJob` |
| B5 WorkQueue L0 执行路径 | 已完成 | `pkg/workqueue/planner.go`, `executor.go` |
| B6 BCH API 读 L3 Facts | 已完成 | `gateway/http/bch_api.go` → `ListFlinkJobsHealth` |
| C1 escalationPolicy（Flink L0→L2） | 已完成 | `pkg/ops/escalation.go`, `workqueue/escalation.go` |
| C2 L2 冷却（1h，告警可绕过） | 已完成 | `workqueue/repository.go` `lastSuccessfulL2At`, `config.DefaultL2CooldownMs` |
| C3 告警接入 Work Queue | 已完成 | `gateway/http/hooks.go`, `workqueue/trigger_alert.go`, `Submit` |
| C4 Map-Reduce 域级汇总 | 已完成 | `domain_reduce.go`, `workqueue/reduce.go`；默认关闭 |
| D1 Spark 作业 L0 批量 | 已完成 | `spark_collector.go`, `job-inspect-spark` |
| D2 YARN 队列 L0 批量 | 已完成 | `yarn_collector.go`, `job-inspect-yarn` |
| D3 GBase 实例 L0 批量 | 已完成 | `gbase_instance_collector.go`, `db_instance` |
| D4 DataApp 管道 L0 批量 | 已完成 | `dataapps_pipeline_collector.go`, `pipeline` |
| D5 批量场景统一框架 | 已完成 | `batch_scenarios.go`, `batch_l0_runner.go`, 通用 escalation |

---

*文档维护：架构 / 平台组。任务条目同步见 [commercialization-task-breakdown.md](./commercialization-task-breakdown.md) §7。*
