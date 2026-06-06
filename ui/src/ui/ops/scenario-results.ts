import type { OpsScenario, WorkbenchTimeRange } from "./scenario-registry.ts";
import { workbenchTimeRangeLabel } from "./workbench-context.ts";

export type OpsScenarioResult = {
  scenarioId: string;
  title: string;
  healthSignal: string;
  riskEvidence: string[];
  recommendedActions: string[];
  expectedBenefit: string;
  outputs: string[];
  runbooks: string[];
  artifacts: string[];
};

function benefitForScenario(scenario: OpsScenario): string {
  switch (scenario.id) {
    case "bch-flink-health":
      return "降低实时作业背压、重启和 Checkpoint 失败带来的 SLA 风险。";
    case "bch-spark-tuning":
      return "释放闲置计算资源，减少慢作业和数据倾斜导致的批处理延迟。";
    case "bch-hdfs-capacity":
      return "减少 namespace 元数据压力，降低小文件和容量水位带来的稳定性风险。";
    default:
      return "沉淀可复用治理动作，提升专项处置的一致性和可追溯性。";
  }
}

export function buildScenarioResult(
  scenario: OpsScenario,
  objectScope: string,
  timeRange: WorkbenchTimeRange,
): OpsScenarioResult {
  const timeLabel = workbenchTimeRangeLabel(timeRange);
  return {
    scenarioId: scenario.id,
    title: scenario.title,
    healthSignal: `${scenario.primaryMetric ?? scenario.title} · ${scenario.secondaryMetric ?? timeLabel}`,
    riskEvidence: [
      `对象范围: ${objectScope || "all"}`,
      `时间范围: ${timeLabel}`,
      ...scenario.inputs.map((input) => `输入证据: ${input}`),
    ],
    recommendedActions: scenario.recommendedActions,
    expectedBenefit: benefitForScenario(scenario),
    outputs: scenario.outputs,
    runbooks: scenario.runbooks,
    artifacts: [`ops-scenario:${scenario.id}`, `object-scope:${objectScope || "all"}`, `time-range:${timeRange}`],
  };
}

export function scenarioResultInputText(result: OpsScenarioResult): string {
  return [`场景: ${result.title}`, `健康信号: ${result.healthSignal}`, ...result.riskEvidence].join("\n");
}

export function scenarioResultOutputText(result: OpsScenarioResult, note: string): string {
  return [
    `输出成果: ${result.outputs.join(" / ")}`,
    `建议动作: ${result.recommendedActions.join(" / ")}`,
    `预期收益: ${result.expectedBenefit}`,
    `闭环备注: ${note}`,
  ].join("\n");
}

