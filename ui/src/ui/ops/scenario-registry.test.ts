import { describe, expect, it } from "vitest";
import {
  OPS_SCENARIOS,
  defaultScenarioForWorkbench,
  filterWorkbenchScenarios,
  findWorkbenchScenario,
  scenarioCatalogStats,
  scenariosForWorkbench,
} from "./scenario-registry.ts";

describe("ops scenario registry", () => {
  it("returns center-specific scenarios for all domains without treating all as BCH detail", () => {
    const diagnosis = scenariosForWorkbench("all", "diagnosis");
    expect(diagnosis.map((scenario) => scenario.id)).toEqual([
      "bch-flink-health",
      "gbase-slow-sql",
      "gbase-lock-wait",
      "fi-component-diagnosis",
      "dataapps-schedule-failure",
    ]);
    expect(diagnosis.map((scenario) => scenario.domain)).toEqual([
      "hadoop",
      "gbase",
      "gbase",
      "fi",
      "dataapps",
    ]);
  });

  it("filters BCH scenarios by workbench center", () => {
    expect(scenariosForWorkbench("hadoop", "diagnosis").map((scenario) => scenario.id)).toEqual([
      "bch-flink-health",
    ]);
    expect(scenariosForWorkbench("hadoop", "governance").map((scenario) => scenario.id)).toEqual([
      "bch-spark-tuning",
    ]);
    expect(scenariosForWorkbench("hadoop", "capacity").map((scenario) => scenario.id)).toEqual([
      "bch-hdfs-capacity",
      "bch-yarn-capacity",
    ]);
  });

  it("does not leak BCH scenarios into other technical domains", () => {
    expect(scenariosForWorkbench("gbase", "diagnosis").map((scenario) => scenario.id)).toEqual([
      "gbase-slow-sql",
      "gbase-lock-wait",
    ]);
    expect(scenariosForWorkbench("gbase", "governance")).toEqual([]);
    expect(defaultScenarioForWorkbench("fi", "capacity")).toBeUndefined();
    expect(findWorkbenchScenario("bch-spark-tuning")?.center).toBe("governance");
  });

  it("registers first scenarios for non-BCH domains", () => {
    expect(scenariosForWorkbench("gbase", "capacity").map((scenario) => scenario.id)).toEqual([
      "gbase-capacity-watermark",
    ]);
    expect(scenariosForWorkbench("fi", "inspection").map((scenario) => scenario.id)).toEqual([
      "fi-service-health",
    ]);
    expect(scenariosForWorkbench("governance", "governance").map((scenario) => scenario.id)).toEqual([
      "governance-metadata-lineage",
      "governance-config-compliance",
    ]);
    expect(scenariosForWorkbench("dataapps", "change").map((scenario) => scenario.id)).toEqual([
      "dataapps-sla-escort",
    ]);
    expect(scenariosForWorkbench("fi", "diagnosis").map((scenario) => scenario.id)).toEqual([
      "fi-component-diagnosis",
    ]);
    expect(scenariosForWorkbench("dataapps", "diagnosis").map((scenario) => scenario.id)).toEqual([
      "dataapps-schedule-failure",
    ]);
  });

  it("summarizes scenario coverage by domain, center, and maturity", () => {
    const allStats = scenarioCatalogStats("all");
    expect(allStats.total).toBe(13);
    expect(allStats.domains).toMatchObject({
      hadoop: 4,
      gbase: 3,
      fi: 2,
      governance: 2,
      dataapps: 2,
    });
    expect(allStats.centers).toMatchObject({
      diagnosis: 5,
      governance: 3,
      capacity: 3,
      inspection: 1,
      change: 1,
    });
    expect(allStats.maturity).toMatchObject({
      planned: 9,
      beta: 3,
      connected: 0,
      automated: 1,
    });
    expect(scenarioCatalogStats("gbase").domains).toEqual({ gbase: 3 });
  });

  it("filters scenarios by keyword and maturity", () => {
    const diagnosis = scenariosForWorkbench("all", "diagnosis");
    expect(filterWorkbenchScenarios(diagnosis, "长事务", "all").map((scenario) => scenario.id)).toEqual([
      "gbase-lock-wait",
    ]);
    expect(filterWorkbenchScenarios(diagnosis, "SLA", "planned").map((scenario) => scenario.id)).toEqual([
      "dataapps-schedule-failure",
    ]);
    expect(filterWorkbenchScenarios(diagnosis, "checkpoint", "beta").map((scenario) => scenario.id)).toEqual([
      "bch-flink-health",
    ]);
    expect(filterWorkbenchScenarios(diagnosis, "", "planned").length).toBe(4);
  });

  it("requires operational metadata for every registered scenario", () => {
    for (const scenario of OPS_SCENARIOS) {
      expect(scenario.objectTypes.length, scenario.id).toBeGreaterThan(0);
      expect(scenario.triggers.length, scenario.id).toBeGreaterThan(0);
      expect(scenario.inputs.length, scenario.id).toBeGreaterThan(0);
      expect(scenario.outputs.length, scenario.id).toBeGreaterThan(0);
      expect(scenario.recommendedActions.length, scenario.id).toBeGreaterThan(0);
      expect(scenario.runbooks.length, scenario.id).toBeGreaterThan(0);
      expect(scenario.owner, scenario.id).toBeTruthy();
      expect(scenario.maturity, scenario.id).toMatch(/^(planned|beta|connected|automated)$/);
      expect(scenario.automationLevel, scenario.id).toMatch(/^(manual|recommendation|approval|closed-loop)$/);
    }
  });
});
