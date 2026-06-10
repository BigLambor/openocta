# OpenOcta 触发与执行架构方案：批量诊断、Cron 与告警统一编排

> 版本：v0.1  
> 日期：2026-06-10  
> 状态：**规划稿，待评审**  
> 适用范围：批量对象健康评估、AI 诊断、Cron 定时巡检、告警事件响应、手动/Chat 触发  
> 关联文档：  
> - [ops-multi-source-ai-architecture.md](./ops-multi-source-ai-architecture.md)  
> - [commercialization-architecture-plan.md](./commercialization-architecture-plan.md)  
> - [commercialization-task-breakdown.md](./commercialization-task-breakdown.md)  
> - [inspection-report-schema.md](./inspection-report-schema.md)  
> - [alert-integration.md](./alert-integration.md)  
> - [product-architecture-digital-ops.md](./product-architecture-digital-ops.md)

### 评审结论（待填写）

```text
状态：规划稿，待产品 + 架构评审

待决议项：
- Work Queue 首期实现：进程内 + SQLite 表 vs 外部队列
- L2 全局并发默认值（建议 5～10）
- Cron 父 Run 完成语义：全部子任务完成 vs 超时后 partial succeeded
- 首个批量样板域：Flink 作业（ops-flink-health）
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
3. **幂等粗糙**：固定 `idempotencyKey: cron:{jobId}`，长任务期间重触发返回 `in_flight` 或被跳过。
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
| `idempotencyKey` | 含时间窗口，避免长任务 blocking 同 key |

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

### 6.2 队列能力（首期建议）

| 能力 | 说明 |
|------|------|
| **全局 L2 并发池** | 默认 5～10，可配置 `cron.maxConcurrentL2Runs` |
| **域级限流** | 如 Flink L2 ≤ 5，GBase L2 ≤ 3 |
| **优先级** | `alert > manual > cron` |
| **幂等** | `idempotencyKey` + 状态机，避免重复入队 |
| **超时** | L0 父任务 30min；L2 单对象 10min；父 Run partial 策略可配置 |
| **持久化** | 首期 SQLite 表 `work_plans` / `work_tasks`；进程内 worker；后期可换 Redis/NATS |

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
- `partial`：L0 成功但部分 L2 失败/超时（可配置是否仍记 cron ok）。
- `failed`：L0 失败。

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
| `work_plans` | 计划快照、状态、父 run_id |
| `work_tasks` | 队列任务：tier、object_id、priority、idempotency_key |
| `job_runs.parent_run_id` | 父子关联（或 `task_id` 复用） |

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
| A2 | SQLite `work_plans` / `work_tasks` + 进程内 worker | 入队、出队、优先级可测 |
| A3 | 实现 `maxConcurrentL2Runs` | 压力测试不超上限 |
| A4 | Cron `lastStatus` 与父 Run 终态绑定 | 不再「send 即 ok」 |
| A5 | 父子 `job_runs` 模型 | UI / API 可查子 Run |

### Phase B — L0 批量样板：Flink（P0）

| 项 | 交付 | 验收 |
|----|------|------|
| B1 | `ops-flink-health` scenario 定义 | 文档 + Go 注册 |
| B2 | `flink_metrics_batch` Collector | 对接 VM/Prom；非 Mock 数据路径 |
| B3 | L0 写 `health_signals` + 域级 `health_snapshots` | 驾驶舱可读 |
| B4 | 单条 Cron `job-inspect-flink` 触发 L0 | 500 级规模压测 < 5min（指标侧依赖环境） |

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

1. **健壮性**：500 对象 Cron 触发不产生无界 goroutine；L2 并发可配置且可观测。
2. **正确性**：Cron `lastStatus` 与父 Run 终态一致；Facts 与 run 可追溯关联。
3. **成本**：全量巡检 LLM 调用量 ≈ 异常对象数，而非对象总数。
4. **事件响应**：告警 P95 入队延迟 < 5s（不含 LLM 执行时间）。
5. **产品**：驾驶舱健康分布来自 L0 Facts；单对象可下钻到 L2 Report 与 evidence。

---

## 12. 开放问题（评审议程）

1. Work Queue 首期是否坚持 SQLite only，还是预留 `queue.backend=redis` 接口？
2. 父 Run `partial` 是否算 Cron 成功（影响 `consecutiveErrors`）？
3. L0 失败时是否仍尝试 L2（告警路径除外）？
4. 多租户下 `scope` 是否必须带 `tenantId`？
5. Reduce 域级汇总 LLM 是否默认开启，还是仅管理报告场景？

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

- `src/pkg/cron/service.go` — Cron 定时与 `Run()`
- `src/pkg/ops/scenario_runner.go` — L1 `RunScenario`
- `src/pkg/ops/alert_jobrun.go` — 告警 JobRun
- `src/pkg/gateway/http/hooks.go` — 告警合并与触发
- `src/pkg/automation/triggers.go` — 告警路由规则
- `src/pkg/jobrun/` — 统一 Run 审计

---

*文档维护：架构 / 平台组。评审通过后更新状态为「已冻结」，并同步任务到 [commercialization-task-breakdown.md](./commercialization-task-breakdown.md)。*
