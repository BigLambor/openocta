# OpenOcta 多源数据与 AI 能力协同架构方案

> 版本：v0.2  
> 日期：2026-06-05  
> 状态：**方向已通过，边界与落地约束已补充**（沟通确认后进入 Phase 1 开发）  
> 适用范围：多技术域运维、多数据源接入、MCP / Tools / Skills 协同、驾驶舱以外的全链路能力  
> 关联文档：  
> - [product-architecture-digital-ops.md](./product-architecture-digital-ops.md)  
> - [architecture-paradigm-analysis.md](./architecture-paradigm-analysis.md)  
> - [digital-ops-architecture-review-opinion.md](./digital-ops-architecture-review-opinion.md)  
> - [ops-remediation-plan.md](./ops-remediation-plan.md)  
> - [ops-monitor-labels-checklist.md](./ops-monitor-labels-checklist.md)  
> - [ops-api.md](./ops-api.md)  
> - [src/docs/tools.md](../src/docs/tools.md)  
> - [src/docs/skills.md](../src/docs/skills.md)  
> - [src/docs/mcp-configuration.md](../src/docs/mcp-configuration.md)

### 评审结论（v0.2）

```text
状态：方向通过；v0.2 已补充边界与落地约束，待最终签字后冻结实施

决议：
- 采纳 L1–L4 分层；L3 Facts 为 UI/API 共同事实层
- 首个样板域：GBase（验证多源边界最充分）
- Phase 1 存储：JSON MVP（stateDir/ops/）；触发 SQLite 条件见 §5.6
- composite 数字分：仅当 coverage ≥ minCoverageForScore 且满足域 requiredAnyOf
- 必需源缺失 / 失败：统一为 `scoreStatus: degraded`；可选源不足才是 `partial`
- Chat 默认不写 HealthSignal；写 Facts 走 §5.8 准入规则
- 权重归属 DomainHealthPolicy，不在单条 HealthSignal 上固化
- metricsBaseUrl 语义：Prometheus/VM base URL（不含 /api/v1/query），见 §5.9
- 巡检分数：废弃 transcript 正则抽分，改结构化 InspectionReport → HealthSignal
- Phase 1 开工：评审签字后启动
```

---

## 1. 背景：我们要解决的不是「驾驶舱缺分」

当前平台已具备多条数据与执行通路，但**尚未形成统一协同模型**：

| 通路 | 现状 | 典型问题 |
|------|------|----------|
| 集群登记 | `Cluster` 含 `monitorLabels`、`jmxUrl`、`fiManagerUrl`、`gbaseDsnRef` 等 | 字段是「连接器」，不是「健康证据」 |
| Agent Tools | `query_vm_metrics`、`query_hadoop_jmx`、`query_fi_manager_metrics`、`query_gbase_slow_sql` 等内置 Go Tool | 仅在对话/巡检时调用，结果不结构化回写平台 |
| MCP | 全局 `mcp.servers` + 数字员工 `mcpServers` | 与内置 Ops Tools 边界不清，缺场景级装配规范 |
| Skills | 通用加载机制已有，运维域 Skill 未成体系 | 模型不知道「GBase 应先查 DSN 再查 VM」 |
| 驾驶舱 / API | `vm_health.go` 同步 PromQL + 资产状态兜底 | 其他数据源未进入 L1/L2 事实层 |
| 告警 | 独立 alerts store，`clusterId` 关联资产 | 未与巡检、健康信号统一建模；无聚合算法 |
| 巡检 | `inspection.go` 用正则从 Agent 文本抽「健康得分」 | 与「结构化 Facts」目标冲突，属待清技术债 |
| BCH 深场景 | `bch_service`（Flink/Spark 等） | 与通用域健康体系并行，未纳入信号模型 |

因此，本方案目标不是「给驾驶舱补一个分」，而是：

> **建立跨驾驶舱、巡检、诊断、告警、资产、数字员工的全平台多源数据协同架构，明确 MCP / Tools / Skills 的分工与衔接，并形成可审计、可扩展、AI-Native 但不失控的事实层。**

---

## 2. 产品哲学：结构化骨架 + AI 引擎（混合范式）

采纳 [architecture-paradigm-analysis.md](./architecture-paradigm-analysis.md) 与 [digital-ops-architecture-review-opinion.md](./digital-ops-architecture-review-opinion.md) 的共识：

```text
结构化运维平台（确定性入口、全局视图、工作流、审计）
  + 全局 / 场景内 AI 助手（推理、编排、解释）
  + 自动化引擎（Cron、Runbook、Skill、MCP、Platform Tools）
```

**不采用**「单一对话框解决一切」作为唯一入口——企业运维需要大盘、值班流程、合规追溯。  
**不采用**「每个数据源写一套 Go 硬编码」——扩展性与 AI-Native 接入会受阻。

数字员工定位（评审意见）：

- ✅ 场景助手、自动化执行载体、垂直任务角色  
- ❌ 不建议作为平台一级导航主轴  

---

## 3. 核心设计理念

### 3.1 四层分离

```text
┌─────────────────────────────────────────────────────────────┐
│ L4 体验层：驾驶舱、技术域、能力域页面、对话、工单、IM 推送      │
├─────────────────────────────────────────────────────────────┤
│ L3 事实层（Facts）：HealthSignal、告警、巡检结果、资产状态      │  ← UI / API 主要读这里
├─────────────────────────────────────────────────────────────┤
│ L2 推理层（Reasoning）：数字员工、Cron Agent、诊断链、Runbook   │  ← LLM + Skill 编排
├─────────────────────────────────────────────────────────────┤
│ L1 执行层（Action）：Platform Tools、MCP Tools、Hooks、CMDB   │  ← 真正访问外部系统
└─────────────────────────────────────────────────────────────┘
```

| 层 | 职责 | 关键约束 |
|----|------|----------|
| L1 执行 | 访问 VM、JMX、FI Manager、GBase、工单、日志等 | 必须尊重集群上下文与 RBAC；未配置则明确失败 |
| L2 推理 | 决定查什么、顺序、如何综合、生成报告 | 输出必须尽量结构化；禁止静默猜分 |
| L3 事实 | 存储最近一次可信观测与结论 | 确定性、可缓存、带来源与新鲜度 |
| L4 体验 | 呈现、下钻、触发任务 | 读 L3；仅在「深度分析」时触发 L2 |

**原则 1：UI 不直接绑 Agent 实时推理**  
驾驶舱刷新、域列表、风险 Top N 等高频读路径走 L3，避免每次打开页面都跑 LLM + MCP。

**原则 2：Agent 不直接写 UI 状态**  
Agent 产出写入 L3 的标准 schema，UI 只认 schema，不认自然语言里的「大概 85 分」。  
现有 `ParseInspectionResult` 中的正则抽分（`inspection.go`）应在 Phase 2 退役，改由 `InspectionReport` 结构化字段驱动。

**原则 3：未接入就明确未接入**  
无信号、无配置、查询失败时展示「未知 / 待配置 / 上次失败原因」，与 [ops-remediation-plan.md](./ops-remediation-plan.md) 一致。

**原则 4：多源可并存、来源可解释**  
任何分数/状态必须带 `source`、`signals[]`、`observedAt`、`coverage`。

**原则 5：部分覆盖不冒充综合分**  
`coverage < minCoverageForScore` 或未满足 `requiredAnyOf` 时，**不展示 composite 数字分**，只展示覆盖状态、缺口与非分数事实（告警数、资产态等）。

---

## 4. MCP / Tools / Skills：分工与边界

### 4.1 一句话定义

| 能力类型 | 定义 | 类比 |
|----------|------|------|
| **Platform Tools** | 平台内置、带集群上下文的一等公民工具 | 官方驱动 |
| **MCP** | 标准协议下的可插拔外部工具 | 第三方驱动 / 客户定制 |
| **Skills** | 领域方法论、步骤、输出规范、失败策略 | 岗位手册 |
| **Scenario** | 把 Skill + Tools/MCP + 对象类型 + 触发器绑成可发现场景包 | 工单模板 |
| **Digital Employee** | 执行载体：Prompt + 绑定 Scenario + 可选 MCP + 权限 | 值班员角色 |

### 4.2 Platform Tools（当前 `pkg/agent/tools`）

**适用：**

- 需要读取集群登记（`GetClusterConfig`）的能力  
- 平台强依赖、需审计与版本随发版的能力  
- 例：`query_vm_metrics`、`query_hadoop_jmx`、`query_fi_manager_metrics`、`query_gbase_slow_sql`

**规则：**

- 工具名稳定，作为 Scenario 的 `platformToolKeys`  
- 工具返回逐步统一为「原始证据 + 建议解析字段」，供 L2 写 L3  
- 长期可包装为「内置 MCP Server」，但对 Agent 仍保持 Tool 接口

### 4.3 MCP

**适用：**

- 客户环境差异大、需独立部署/升级的系统  
- 已有 MCP 生态的组件（Prometheus、ES、工单、日志平台等）  
- 数字员工专属连接器（`manifest.mcpServers`）

**不适用：**

- 驾驶舱每次加载时的同步健康查询（延迟、成本、不确定性）  
- 替代 L3 事实存储

**装配规则：**

```text
有效工具集 = Builtin Tools ∪ 全局 mcp.servers ∪ 员工 mcpServers（同 key 员工覆盖全局）
```

**工具冲突策略：**

- 平台 Tool 与 MCP 同名时，平台 Tool 优先（安全与上下文注入）  
- MCP 工具须声明 `domainKeys` / `capabilityKeys` 元数据（可在 MCP tool description 或侧车配置）

**降级策略（细化，见 §4.5 `requiredSources`）：**

- 并非所有 MCP 失败都可「仅用 Platform Tool 替代」  
- Scenario 声明的 **必需源** 失败 → `scoreStatus: degraded`，不静默降权、不伪造 composite  
- **可选源** 失败 → 降低 `coverage`，在 UI 标注缺失源

### 4.4 Skills

**适用：**

- 多步巡检剧本（先查可用源 → 再查主信号 → 再查辅信号）  
- 各域判读标准（BCH 看 YARN/HDFS/Flink；GBase 看连接/慢 SQL）  
- 输出 JSON schema 与「缺配置时的诚实报错模板」  
- monitorLabels 对齐、证据引用格式

**不适用：**

- 替代 Tool/MCP 发 HTTP/SQL  
- 存储实时指标（Skill 是知识，不是数据库）

**建议目录结构：**

```text
skills/
├── ops-core/
│   └── SKILL.md                 # 通用：证据优先、禁止猜分、结构化输出
├── ops-bch-inspection/
│   └── SKILL.md
├── ops-fi-inspection/
│   └── SKILL.md
├── ops-gbase-health/            # ← 首个样板 Skill
│   └── SKILL.md
├── ops-governance-metadata/
│   └── SKILL.md
└── ops-dataapp-scheduler/
    └── SKILL.md
```

### 4.5 Scenario（场景包）— 协同的「装配单元」

沿用 [product-architecture-digital-ops.md](./product-architecture-digital-ops.md) 的 `OpsScenario`，扩展为可执行规格：

```text
OpsScenario
├── key                          # 如 ops-gbase-health
├── domainKey                    # gbase
├── capabilityKey                # health-inspection | diagnosis-incident | ...
├── objectType                   # cluster | service | job | db_instance
├── skillIds[]                   # 推理剧本
├── platformToolKeys[]           # 平台工具
├── mcpServerKeys[]              # 可选 MCP 服务
├── requiredSources[]            # 必需信号源，缺则 degraded / 不产出 composite
├── optionalSources[]            # 可选信号源，缺则降低 coverage
├── inputSources[]               # alerts | metrics | inspection | cmdb | ...
├── outputSchema                 # HealthSignal | InspectionReport | DiagnosisReport
├── triggerTypes[]               # manual | cron | alert_hook | chat_intent
└── employeeIds[]                # 推荐执行员工（可选）
```

#### 必需源 / 可选源规则

| 概念 | 含义 | 失败时行为 |
|------|------|------------|
| `requiredSources` | 场景或域策略要求的信号类型（如 `gbase_sql`） | 不写 composite 分；`scoreStatus: degraded`；记录 `missingSources[]` |
| `optionalSources` | 增强覆盖的信号（如 `metrics`、`inspection`） | 不参与 score 加权分母；但仍计入 coverage 期望源，故 `coverage` 下降 |
| `platformToolKeys` / `mcpServerKeys` | L1 执行入口 | 必需源对应工具/MCP 不可用时，等同必需源失败 |

**示例（GBase 样板 Scenario）：**

```yaml
key: ops-gbase-health
domainKey: gbase
objectType: cluster
requiredSources: [gbase_sql]          # 无 DSN → 诚实失败，不跑 composite
optionalSources: [metrics, alerts, inspection]
platformToolKeys: [query_gbase_slow_sql, query_vm_metrics]
mcpServerKeys: []                     # Phase 3 可扩展
outputSchema: InspectionReport
```

**这是 MCP / Tools / Skills 的「交汇点」**：不是三选一，而是 Scenario 声明组合关系与失败语义。

### 4.6 Digital Employee

员工 = **Scenario 的运行时实例**，而非能力集合本身：

```text
DigitalEmployee
├── domainKeys[]
├── capabilityKeys[]
├── scenarioKeys[]               # 推荐显式绑定（样板：ops-gbase-health）
├── skillIds[]
├── mcpServers{}
├── permissionKeys[]
└── scheduleBindings[]
```

---

## 5. 多源数据模型：L3 Facts 与证据链

### 5.1 对象分层

| 对象 | 层级 | 职责 |
|------|------|------|
| `Cluster` | 连接器（已有） | 登记如何连，不表示当前健康 |
| `HealthSignal` | L3 原子 | 单次观测的结构化证据 |
| `HealthSnapshot` | L3 聚合 | 某对象在某时刻的多源快照（cluster / job / db_instance 通用） |
| `InspectionReport` / `DiagnosisReport` | L3 报告 | L2 运行的完整产出，可衍生 HealthSignal |
| `DomainHealthPolicy` | 配置 | 域级权重、必需源、覆盖阈值 |

### 5.2 HealthSignal — 完整字段（schema v1）

单条 Signal **只记录自身证据**，**不固化聚合权重**（权重见 §5.4 DomainHealthPolicy）。

```json
{
  "schemaVersion": "1",
  "id": "sig-uuid",
  "runId": "run-uuid",
  "scenarioKey": "ops-gbase-health",

  "objectType": "cluster",
  "objectId": "cluster-xxx",
  "clusterId": "cluster-xxx",
  "domain": "gbase",

  "tenant": "",
  "env": "prod",
  "region": "哈池",

  "type": "gbase_sql",
  "status": "warning",
  "score": 82,
  "confidence": "high",

  "source": "platform_tool:query_gbase_slow_sql",
  "sourceKind": "platform_tool",
  "evidence": {
    "slowSqlCount": 3,
    "connectionErrors": 0
  },
  "error": "",

  "observedAt": "2026-06-05T10:00:00Z",
  "ttlSec": 300,
  "freshness": "ok"
}
```

| 字段 | 说明 |
|------|------|
| `schemaVersion` | Signal schema 版本，便于迁移 |
| `runId` | 产生该信号的 L2 运行 ID（Cron/巡检/告警触发）；Go 直连采集可为 `collector-run-id` |
| `scenarioKey` | 关联 Scenario；直连采集可为 `system:collector` |
| `objectType` / `objectId` | 通用对象引用；cluster 场景下 `objectId === clusterId` |
| `tenant` / `env` / `region` | 多租户与环境维度（可从 Cluster 登记填充） |
| `type` | 信号类型，见 §5.3 |
| `score` / `status` | 该源独立判读结果；允许某源只有 `status` 无 `score` |
| `confidence` | `high` \| `medium` \| `low` — 来源可信度（Chat 写入通常为 `low`，见 §5.8） |
| `source` / `sourceKind` | 执行来源：`platform_tool` \| `mcp` \| `collector`；告警/资产等 Go 采集统一为 `sourceKind: collector`、`source: collector:alerts` / `collector:asset_status` |
| `evidence` | 原始或半结构化证据，供下钻与审计 |
| `ttlSec` / `freshness` | 新鲜度；过期后 UI 标注，不自动删除历史 |

### 5.3 信号类型注册表

| type | 典型来源 | 主要域 | 采集方式 |
|------|----------|--------|----------|
| `metrics` | VM / Prometheus | 全域 | Platform Tool 或 Prometheus MCP |
| `jmx` | Hadoop JMX HTTP | BCH | Platform Tool |
| `fi_manager` | FI Manager API | FI | Platform Tool |
| `gbase_sql` | GBase DSN SQL | GBase | Platform Tool |
| `governance_api` | 治理平台 API | governance | Platform Tool / MCP |
| `scheduler_api` | Airflow / DS | dataapps | MCP |
| `alerts` | alerts store | 全域 | Go collector（§5.5） |
| `inspection` | 最近巡检结构化结果 | 全域 | L2 写入 |
| `asset_status` | CMDB / 人工登记 | 全域 | Go collector |
| `bch_workload` | Flink / Spark 专有能力 | BCH | bch_service / 未来 Tool |

### 5.4 DomainHealthPolicy — 权重与必需源

权重与覆盖规则**只在此配置**，不在 HealthSignal 上重复。

```yaml
# config/ops/domain_health_policy.yaml（拟议路径）
schemaVersion: "1"

gbase:
  policyVersion: "1"
  requiredAnyOf: [gbase_sql]           # 至少一种主信号成功，才允许 composite
  weights:
    gbase_sql: 0.45
    metrics: 0.25
    alerts: 0.20
    inspection: 0.10
    asset_status: 0.05                 # 兜底权重最低
  minCoverageForScore: 0.5             # 参与加权的源中，成功源占比阈值
  coverageScope: configured            # 期望源 = requiredSources + 已配置 optionalSources
  defaultTtlSec: 300
```

聚合时：

1. 取对象最新一批未过期 Signal（按 `type` 取最新 `observedAt`）  
2. 仅对 **Policy 中有权重、有独立 score、且已成功采集** 的 type 参与 score 加权分子/分母  
3. `coverage = 成功源数 / 期望源数`；默认期望源 = Scenario `requiredSources` + **已配置** `optionalSources` 与 Policy 权重交集（`coverageScope: configured`）  
4. 若未满足 `requiredAnyOf` → **不输出 `score` 数字**，输出 `scoreStatus: degraded` + `missingSources[]` + `coverage`  
5. 若满足 `requiredAnyOf` 但 `coverage < minCoverageForScore` → **不输出 `score` 数字**，输出 `scoreStatus: partial` + `missingSources[]` + `coverage`

### 5.5 告警信号聚合算法（`alerts` type）

告警由 Go collector 从 alerts store 生成 `HealthSignal`，不经过 Agent。

**输入：** 某 `objectId`（优先 `clusterId`）下待处理告警列表。

**归属规则：**

| 条件 | 归属 |
|------|------|
| `alert.clusterId` 非空且匹配资产 `id` | 归入该 cluster |
| `alert.clusterId` 空，但 `service`/`instance` 可映射到域 | 归入域级「未挂载」桶，**不参与** cluster composite |
| 无法归属 | 仅出现在域级告警面板，不写入 cluster Signal |

**去重：** 按 fingerprint（`alertname` + `service` + `instance` + `clusterId` + `component`）合并；同一 fingerprint 只计一次。

**active / analyzing / resolved：**

- 仅 **active / analyzing** 告警参与健康惩罚（与当前 alerts store 状态对齐）  
- `resolved` 在 TTL 窗口内可作为 `evidence.resolvedRecently[]` 记录，**不降低当前分数**

**分数映射（默认，可按域覆盖）：**

```text
base = 100
每条 active/analyzing critical：  -15（下限 35）
每条 active/analyzing warning：   -8
每条 active/analyzing info：      -2
无 active/analyzing 告警：        score = 100, status = healthy
```

**status 映射：**

- 存在 active/analyzing critical → `critical`  
- 仅 warning → `warning`  
- 无 active/analyzing → `healthy`

**confidence：** 固定 `high`（来自结构化 store，非 LLM 推断）。

### 5.6 存储介质与迁移边界

| 阶段 | 介质 | 路径（拟议） | 说明 |
|------|------|--------------|------|
| Phase 1 MVP | JSON 文件 | `stateDir/ops/health_signals.json`、`health_snapshots.json` | 与现有 `clusters.json` 一致 |
| Phase 2+ | 评估 SQLite | `stateDir/ops/ops.db` | 满足下列**任一**触发条件即规划迁移 |

**触发 SQLite 规划的条件（满足任一）：**

1. 需要按 `objectId` / `domain` / 时间范围**分页查询**历史 Signal  
2. 需要**并发写入**（多 Cron / 多集群并行采集）  
3. 需要 L2 `runId` 级**审计追溯**与报表导出  
4. 单 JSON 文件体积或读写延迟成为瓶颈  

Phase 1 必须在代码中抽象 `HealthSignalStore` 接口，避免业务逻辑绑死 JSON 实现。

### 5.7 HealthSnapshot — 聚合快照字段

```json
{
  "schemaVersion": "1",
  "aggregationPolicyVersion": "gbase:1",
  "objectType": "cluster",
  "objectId": "cluster-xxx",
  "domain": "gbase",

  "score": null,
  "scoreStatus": "degraded",
  "source": "composite",
  "coverage": 0.33,
  "missingSources": ["gbase_sql", "metrics"],
  "presentSources": ["alerts", "asset_status"],

  "signals": [ /* HealthSignal[] 摘要或引用 */ ],
  "observedAt": "2026-06-05T10:00:00Z"
}
```

| `scoreStatus` | 含义 | UI 行为 |
|---------------|------|---------|
| `ok` | 满足 composite 条件且有分 | 显示数字分 + 「综合」 |
| `warning` / `critical` | 有分且状态差 | 显示数字分 + 色条 |
| `partial` | 有信号但未达 `minCoverageForScore` | **不显示数字分**；显示覆盖度与缺口 |
| `degraded` | 必需源缺失或失败 | **不显示数字分**；显示 degraded 与原因 |
| `unknown` | 无任何有效信号 | 「待评分」 |

域级 `DomainHealthSnapshot` 由同域 cluster 快照聚合（默认：有分集群的加权平均；无分集群计入 `partialCount`）。

### 5.8 Chat 写回 Facts 准入规则

Chat **默认不**将对话结论写入 `HealthSignal` 参与 composite。

| 写入目标 | 默认 | 准入条件 |
|----------|------|----------|
| `DiagnosisReport` 草稿 | ✅ 允许 | schema 校验通过即可；`confidence: low` |
| `InspectionReport` 草稿 | ✅ 允许 | 用户显式触发「保存巡检」或 Cron/按钮触发 Scenario |
| `HealthSignal` | ❌ 默认禁止 | 须**同时**满足：① 输出通过 JSON schema 校验；② `evidence` 含完整工具/MCP/collector 原始引用；③ `sourceKind` 为 `platform_tool` / `mcp` / `collector`（**非** `chat`）；④ 由 **Cron / 告警 Hook / 用户确认按钮** 触发写入，而非纯对话自动落库 |
| Chat 直接写 `score` | ❌ 禁止 | 不得用自然语言分数驱动 UI |

用户对话可生成「建议结论」嵌在聊天卡片；是否进入 L3 聚合由显式动作或系统触发器决定。

### 5.9 连接器字段语义：`metricsBaseUrl`

**冻结语义（v0.2）：**

| 字段 | 语义 | 示例 |
|------|------|------|
| `metricsBaseUrl` | Prometheus / VictoriaMetrics **服务根 URL**（不含 `/api/v1/query`） | `http://vm.example.com:8428` |
| `vmUrlRef` | 环境变量名，解析结果为上述 base URL | `VICTORIAMETRICS_URL_BJ` |
| 全局 `VICTORIAMETRICS_URL` | 同上，域/集群未配置时的默认 base | `http://127.0.0.1:8428` |

代码侧统一：`queryInstant(baseURL, promql)` 内部拼接 `baseURL + "/api/v1/query"`（与当前 `vm_health.go` 一致）。

**迁移说明：** 现有种子数据若写成完整 query endpoint（如 `.../api/v1/query`），应在数据迁移或规范化函数中 strip 后缀，避免双拼路径。

### 5.10 与 monitorLabels 的关系

- `monitorLabels`：仅服务 `metrics` 类信号（PromQL 注入）  
- `gbaseDsnRef`：仅服务 `gbase_sql`  
- 资产 `id`：告警关联、API 主键，**不**自动进入 PromQL  

详见 [ops-monitor-labels-checklist.md](./ops-monitor-labels-checklist.md)。

---

## 6. 全链路协同：不仅是驾驶舱

### 6.1 场景矩阵

| 用户旅程 | L4 入口 | L2 触发 | L1 执行 | L3 写入 | L4 读取 |
|----------|---------|---------|---------|---------|---------|
| 全局驾驶舱 | 运维大屏 | Cron 周期采集 | Collector + Tool | DomainHealthSnapshot | 分数或 partial 态 |
| 技术域概览 | GBase / BCH / FI 页 | 域巡检 Scenario | 域 Skill 编排 | HealthSnapshot | 分布条、Top 风险 |
| 一键巡检 | 集群「巡检」 | Inspection Scenario | Tools 按序 | InspectionReport → Signal | 巡检历史 |
| 告警处置 | 告警详情 | Diagnosis Scenario | 告警+指标 MCP | DiagnosisReport | 处置建议 |
| 对话分析 | Chat | 意图 → Scenario | Skill 选 Tool | 草稿 Report；Signal 需 §5.8 | 聊天卡片 |
| CMDB 同步 | 资产页 | 无 Agent | CMDB HTTP | Cluster 登记 | 资产表 |

### 6.2 标准流水线

```text
Trigger → Resolve Context → Load Scenario（含 required/optional Sources）
  → Apply Skill → Execute L1（Tools/MCP/Collector）
  → Normalize HealthSignal → Aggregate HealthSnapshot（Policy 加权）
  → Persist L3 → Notify → Render L4
```

### 6.3 Cron 与告警的分工

| 触发器 | 职责 |
|--------|------|
| **Cron** | 周期刷新 L3（collector + Scenario），保证大盘有「上次观测」 |
| **告警 Hook** | 启动 Diagnosis Scenario；刷新 `alerts` Signal |
| **Chat** | 深度推理；默认只产 Report 草稿 |

---

## 7. 各技术域落地蓝图

### 7.1 首个样板域：GBase（已选定）

| 层级 | 内容 |
|------|------|
| 验证目标 | 连接器≠事实；无 DSN 诚实失败；metrics 可选；仅有 alerts/asset 不出 composite |
| Skills | `ops-gbase-health` |
| Platform Tools | `query_gbase_slow_sql`；（可选）`query_vm_metrics` |
| Scenario | `ops-gbase-health`：`requiredSources: [gbase_sql]` |
| Policy | 见 §5.4 示例 |
| Phase 1 验收 | 无 VM、无 DSN 时：展示 degraded + 缺口，**无 composite 数字分** |

### 7.2 BCH（hadoop）— Phase 4

Skills `ops-bch-inspection`；主信号 `metrics` + `jmx` + `bch_workload`；与 GBase 样板解耦，避免特有能力干扰架构验证。

### 7.3 FI / 治理 / 数据 App

按同一模板扩展 Scenario + Policy；治理/数据 App 以 MCP 优先。

---

## 8. 与现有代码的映射

| 现状 | 目标态 | 迁移动作 |
|------|--------|----------|
| `vm_health.go` 同步算分 | Collector 写 `metrics` Signal；UI 读 Snapshot | 抽象 `collectMetricsSignal`；规范化 base URL |
| `inspection.go` 正则抽分 | `InspectionReport.score` 结构化 | 废弃正则路径 |
| `pkg/agent/tools/*.go` | L1 执行器 + 统一 evidence | 扩展 ToolResult |
| alerts store | §5.5 告警 collector | `collectAlertSignal` |
| `service.go` JSON store | `HealthSignalStore` 接口 | Phase 1 JSON 实现 |
| Skills 无运维包 | `skills/ops-gbase-health` | Phase 2 |

---

## 9. 治理与安全

### 9.1 RBAC

L4 读 Facts 按域过滤；L2 触发按 `ops:inspect` / `ops:diagnose`；L1 按员工权限与 cluster 范围。

### 9.2 审计

每次 L2 运行：`runId, scenarioKey, employeeId, objectId, toolsCalled[], mcpCalled[], signalsWritten[], missingSources[], durationMs`。

### 9.3 成本与稳定性

域级/员工级并发上限；Signal TTL 过期标注；必需源失败不阻塞其他对象的采集。

### 9.4 诚实原则

无分 → `unknown` / `partial` / `degraded`；工具失败写 `error`，必需源失败标 `degraded`；IM 低分基于结构化 `score`，禁止正则猜分。

---

## 10. 分阶段实施路线

### Phase 0：方案冻结 ✅（v0.2）

- [x] 方向通过  
- [x] L3 完整 schema、聚合算法、Scenario 源规则、Chat 准入、metricsBaseUrl 语义  
- [x] 首个样板域：**GBase**  
- [ ] 评审签字 → 启动 Phase 1  

### Phase 1：事实层骨架（2–3 周）

| 项 | 交付 |
|----|------|
| P1-1 | `HealthSignal` / `HealthSnapshot` schema v1 + `docs/ops-api.md` |
| P1-2 | `HealthSignalStore` 接口 + JSON 实现 |
| P1-3 | Collector：`alerts`（§5.5）、`asset_status`；`metrics` 可选 |
| P1-4 | GBase 域 Policy 配置；Snapshot 聚合（§5.4） |
| P1-5 | 驾驶舱读 Snapshot：`composite` / `partial` / `degraded` 展示 |
| P1-6 | `metricsBaseUrl` 规范化与种子数据修正 |

**验收（GBase，无 VM、无 DSN）：**

- [ ] 展示 `scoreStatus: degraded`、`coverage`、`missingSources`（含 `gbase_sql`）  
- [ ] 若有 active 告警，展示 `alerts` 信号与告警数，**不展示 composite 数字分**  
- [ ] 若有 `asset_status`，展示资产态，**不展示 composite 数字分**  
- [ ] 配置 DSN + 采集成功后，`coverage ≥ minCoverageForScore` 时才出现 composite 分  

### Phase 2：GBase Scenario + Skill（2–3 周）

| 项 | 交付 |
|----|------|
| P2-1 | `ops-gbase-health` Scenario + SKILL.md |
| P2-2 | Cron 执行 → `InspectionReport` → `gbase_sql` Signal |
| P2-3 | 退役 `inspection.go` 正则抽分路径 |
| P2-4 | 驾驶舱在达标时显示 `composite` 分 |

### Phase 3：MCP 装配（2 周）

Scenario `mcpServerKeys`、员工 `scenarioKeys`、必需/可选源运行时校验、Prometheus MCP 试点。

### Phase 4：全域推广

BCH / FI / governance / dataapps；`bch_workload`；告警 Hook → Diagnosis。

---

## 11. 仍待签字确认项

| # | 问题 | v0.2 倾向 |
|---|------|-----------|
| 1 | Scenario 注册存放位置 | Phase 1 用 YAML/`ops/scenarios.json`；二期再考虑 UI 可编辑 |
| 2 | 驾驶舱分数语义 | **上次观测分**（非实时）；默认 TTL 300s |
| 3 | `vm_health.go` 一期策略 | 与 L3 **并行**，读 Snapshot 优先，VM 同步作 fallback，Phase 2 移除 fallback |
| 4 | 数字员工粒度 | 样板期 **每 Scenario 可绑员工**，不强制每域一个 |
| 5 | Skill 意图路由 | Phase 2 仅员工/按钮绑定；意图路由放 Phase 4 |

---

## 12. 总结

OpenOcta 作为 AI-Native 运维平台：

```text
Skills   → 怎么查、怎么判、怎么输出
Tools    → 平台官方、带集群上下文
MCP      → 客户与生态扩展
Scenario → 装配 + 必需/可选源 + 失败语义
Policy   → 权重与 composite 准入
Facts    → UI/API/告警共用确定性层
```

**Cluster 是连接配置，不是健康事实。**  
**Agent 写 schema，UI 不读自然语言。**  
**部分覆盖不冒充综合分。**

驾驶舱是 L4 消费者之一；巡检、诊断、告警、资产、Cron、对话共享 L3，才是多源数据的整体解。

---

*确认实施后请同步更新 [ops-roadmap.md](./ops-roadmap.md) 与 [ops-remediation-plan.md](./ops-remediation-plan.md)。*
