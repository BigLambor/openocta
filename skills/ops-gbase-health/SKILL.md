# GBase Health Inspection Skill

Use this skill for GBase cluster health inspection, slow SQL diagnosis, and health evidence normalization.

## Scope

- Domain: `gbase`
- Object type: `cluster`
- Scenario key: `ops-gbase-health`
- Required source: `gbase_sql`
- Optional sources: `metrics`, `alerts`, `inspection`

## Required Behavior

1. Resolve the ops context first: `domain`, `clusterId`, and optional `component`.
2. If `clusterId` is missing or equals `all`, state that a concrete GBase cluster is required for DSN-based checks.
3. Check whether the cluster has `gbaseDsnRef` or the environment has `GBASE_DSN`.
4. If no DSN is available, stop the DSN path and report a degraded result. Do not invent slow SQL rows or use asset status as a health score.
5. Call `query_gbase_slow_sql` before optional metrics checks.
6. If metrics are configured, call `query_vm_metrics` for GBase connection/QPS/TPS signals. If metrics are not configured, mark `metrics` as an optional missing source.
7. Produce a structured report with score/status/evidence/errors. Natural-language explanation may be included as `reportMarkdown`, but UI state must use structured fields.

## Slow SQL Scoring Guidance

- 0 slow SQL rows: `gbase_sql` score 100, status `healthy`
- 1-3 slow SQL rows: status `warning`, explain top statements and likely impact
- More than 3 rows: status `warning` or `critical` depending on duration and repetition
- Connection failure or DSN error: status `critical`, no composite score unless another required source is configured by policy

## Output Contract

Return or persist an `InspectionReport` compatible object:

```json
{
  "domain": "gbase",
  "clusterId": "cluster-gbase-prod",
  "score": 82,
  "scoreStatus": "warning",
  "toolRuns": [
    {
      "toolName": "query_gbase_slow_sql",
      "success": true,
      "output": "[{\"sql_text\":\"...\",\"exec_time_sec\":12}]"
    }
  ],
  "metricsEvidence": {
    "gbase_sql": {
      "slowSqlCount": 1
    }
  },
  "errors": [],
  "reportMarkdown": "..."
}
```

## Honesty Rules

- Never silently guess a score when `gbase_sql` is missing.
- Never turn `asset_status` alone into a composite score.
- If `query_gbase_slow_sql` fails, preserve the exact tool error in `errors[]` and evidence.
- If optional metrics fail, keep the GBase SQL evidence and lower coverage instead of failing the whole report.

