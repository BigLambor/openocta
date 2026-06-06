import { describe, expect, it } from "vitest";
import {
  buildInspectionDetailBullets,
  extractInspectionExecutiveSummary,
  normalizeInspectionReportMarkdown,
  resolveInspectionSummaries,
} from "./inspection-report.ts";

const SAMPLE_REPORT = `# BCH 生态深度健康巡检报告

**巡检时间**：2026-06-06
**综合健康分**：86 / 100

## 执行摘要

本次巡检覆盖 YARN、HDFS、Flink 流作业与 Spark 批作业链路。整体可用性良好，但存在 2 个 Flink 背压作业。

## 健康度评分

| 维度 | 状态 | 说明 |
|------|------|------|
| 流计算健康 | 亚健康 | 2 个作业持续背压 |

## 风险项与处置建议

### P1 · Flink 背压（prod-b / risk-realtime-calc）
- **建议**：提升并行度

## 二、关键指标

1. YARN 活跃节点 48 台
`;

describe("inspection-report", () => {
  it("extracts executive summary lead only", () => {
    const summary = extractInspectionExecutiveSummary(SAMPLE_REPORT);
    expect(summary).toContain("Flink 背压作业");
    expect(summary).not.toContain("健康度评分");
    expect(summary).not.toContain("YARN 活跃节点");
  });

  it("strips json fences from report body", () => {
    const raw = '```json\n{"score":86}\n```\n\n# Report\n\n## 执行摘要\n\n简短总结。';
    expect(normalizeInspectionReportMarkdown(raw)).toBe("# Report\n\n## 执行摘要\n\n简短总结。");
  });

  it("builds structured detail bullets", () => {
    const bullets = buildInspectionDetailBullets(
      { status: "ok", result: { score: 86, scoreStatus: "warning", reportMarkdown: SAMPLE_REPORT } },
      86,
      SAMPLE_REPORT,
    );
    expect(bullets[0]).toContain("86/100");
    expect(bullets.some((line) => line.includes("Flink 背压作业"))).toBe(true);
    expect(bullets.some((line) => line.includes("P1"))).toBe(true);
  });

  it("does not use full report as summary", () => {
    const { reportSummary, reportMarkdown } = resolveInspectionSummaries(
      {
        status: "ok",
        result: { score: 86, scoreStatus: "warning", reportMarkdown: SAMPLE_REPORT },
      },
      86,
    );
    expect(reportSummary).toContain("86/100");
    expect(reportMarkdown).toContain("## 二、关键指标");
    expect(reportSummary).not.toContain("YARN 活跃节点 48 台");
  });
});
