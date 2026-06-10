# InspectionReport JSON Schema (C2-10)

> 状态：商用 P0 结构化巡检报告 schema  
> 实现：`src/pkg/ops/inspection.go`（`InspectionResult`）、`src/pkg/ops/inspection_validator.go`

## 用途

Agent / Scenario Runner 在完成巡检后输出 **InspectionReport**，用于：

1. 写入 `openocta.db.inspection_reports`（可追溯原文与 evidence）
2. 投影到 L3 Facts（`health_signals` / `health_snapshots`）供 UI 高频读取
3. 关联 `job_runs.run_id` 与 tool/model 审计

## 顶层字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | 是 | 会话或 run 标识 |
| `jobId` | string | 推荐 | 巡检任务 ID |
| `domain` | string | 推荐 | 技术域：`hadoop` / `fi` / `gbase` 等 |
| `clusterId` | string | 推荐 | 目标集群 ID |
| `component` | string | 否 | 组件范围 |
| `scenarioKey` | string | 否 | 场景键，如 `ops-gbase-health` |
| `score` | int 0-100 | 条件 | 健康分 |
| `scoreStatus` | string | 条件 | `ok` / `warning` / `critical` / `unknown` / `degraded` |
| `confidence` | string | 推荐 | `high` / `medium` / `low` |
| `summary` | string | 推荐 | 人类可读摘要 |
| `risks` | string[] | 否 | 风险列表 |
| `recommendedActions` | string[] | 否 | 建议动作 |
| `requiresApproval` | bool | 否 | 是否建议进入审批 |
| `toolRuns` | ToolRun[] | 条件 | 工具执行明细 |
| `metricsEvidence` | object | 否 | 指标证据 |
| `validationStatus` | string | 系统 | `valid` / `invalid` / `missing` |
| `validationErrors` | string[] | 系统 | 校验失败原因 |
| `startedAt` / `finishedAt` | int64 ms | 推荐 | 时间范围 |

至少应提供以下之一：`score`、`scoreStatus`、`toolRuns`、`metricsEvidence`、`summary`、`errors`、`risks`、`recommendedActions`。

## ToolRun

```json
{
  "toolName": "query_gbase_slow_sql",
  "success": true,
  "output": "[]",
  "error": ""
}
```

## 校验规则（商用路径）

- 合法 JSON 块 → `validationStatus=valid`，允许写入 Facts
- 非法 JSON → `validationStatus=invalid`，`scoreStatus=degraded`，**禁止**正则抽分（除非 `OPENOCTA_INSPECTION_ALLOW_LEGACY_SCORE=1`）
- 写入路径：`PersistInspectionFacts` → `inspection_reports` + `health_signals`

## 示例

```json
{
  "domain": "gbase",
  "clusterId": "cluster-prod-a",
  "score": 91,
  "scoreStatus": "ok",
  "confidence": "high",
  "summary": "GBase 巡检通过，无慢 SQL",
  "risks": [],
  "recommendedActions": ["继续观察连接池"],
  "requiresApproval": false,
  "toolRuns": [
    {"toolName": "query_gbase_slow_sql", "success": true, "output": "[]"}
  ]
}
```

## 交叉引用

- [commercialization-domain-objects.md](./commercialization-domain-objects.md) — 巡检对象 owner
- [openocta-db-schema-v1.md](./openocta-db-schema-v1.md) — DB 表演进
- Migration `009_inspection_reports.sql`
