import { describe, expect, it } from "vitest";
import { findWorkbenchScenario } from "./scenario-registry.ts";
import {
  buildScenarioResult,
  scenarioResultInputText,
  scenarioResultOutputText,
} from "./scenario-results.ts";

describe("scenario results", () => {
  it("builds a consistent result envelope for BCH scenarios", () => {
    for (const id of ["bch-flink-health", "bch-spark-tuning", "bch-hdfs-capacity"]) {
      const scenario = findWorkbenchScenario(id);
      expect(scenario, id).toBeTruthy();
      const result = buildScenarioResult(scenario!, "all", "24h");

      expect(result.scenarioId).toBe(id);
      expect(result.healthSignal).toContain(scenario!.primaryMetric!);
      expect(result.riskEvidence.some((line) => line.includes("对象范围"))).toBe(true);
      expect(result.riskEvidence.some((line) => line.includes("时间范围"))).toBe(true);
      expect(result.recommendedActions.length).toBeGreaterThan(0);
      expect(result.expectedBenefit).toBeTruthy();
      expect(result.artifacts).toContain(`ops-scenario:${id}`);
    }
  });

  it("serializes scenario result into execution-record input and output", () => {
    const scenario = findWorkbenchScenario("bch-spark-tuning")!;
    const result = buildScenarioResult(scenario, "cluster:bch-prod-1", "7d");

    expect(scenarioResultInputText(result)).toContain("健康信号");
    expect(scenarioResultInputText(result)).toContain("近 7 天");
    expect(scenarioResultOutputText(result, "已采纳")).toContain("预期收益");
    expect(scenarioResultOutputText(result, "已采纳")).toContain("闭环备注: 已采纳");
  });
});

