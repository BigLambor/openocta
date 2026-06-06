import { describe, expect, it } from "vitest";
import type { OpsClusterRecord } from "../controllers/ops-clusters.ts";
import { findWorkbenchScenario } from "./scenario-registry.ts";
import {
  clusterObjectId,
  namespaceObjectId,
  normalizeWorkbenchObjectScope,
  normalizeWorkbenchTimeRange,
  objectOptionsForScenario,
  parseWorkbenchObjectScope,
  workbenchTimeRangeLabel,
} from "./workbench-context.ts";

const clusters: OpsClusterRecord[] = [
  {
    id: "c1",
    name: "bch-prod-1",
    domain: "hadoop",
    region: "上海",
    nodeCount: 10,
    components: ["HDFS"],
    owner: "ops",
    status: "healthy",
    createdAtMs: 1,
    updatedAtMs: 1,
  },
  {
    id: "c2",
    name: "bch-prod-2",
    domain: "hadoop",
    region: "北京",
    nodeCount: 8,
    components: ["YARN"],
    owner: "ops",
    status: "warning",
    createdAtMs: 1,
    updatedAtMs: 1,
  },
];

describe("workbench context", () => {
  it("builds cluster scope options for Flink and Spark scenarios", () => {
    const scenario = findWorkbenchScenario("bch-flink-health");
    const options = objectOptionsForScenario(scenario, clusters);

    expect(options.map((option) => option.label)).toEqual(["全部集群", "bch-prod-1", "bch-prod-2"]);
    expect(parseWorkbenchObjectScope(clusterObjectId("bch-prod-1"))).toEqual({
      kind: "cluster",
      value: "bch-prod-1",
    });
  });

  it("builds namespace scope options for HDFS capacity scenario", () => {
    const scenario = findWorkbenchScenario("bch-hdfs-capacity");
    const options = objectOptionsForScenario(scenario, clusters);

    expect(options[0]?.label).toBe("全部 namespace");
    expect(options.some((option) => option.id === namespaceObjectId("NS8"))).toBe(true);
    expect(parseWorkbenchObjectScope(namespaceObjectId("NS2"))).toEqual({
      kind: "namespace",
      value: "NS2",
    });
  });

  it("normalizes invalid object scope and time ranges", () => {
    const scenario = findWorkbenchScenario("bch-spark-tuning");
    const options = objectOptionsForScenario(scenario, clusters);

    expect(normalizeWorkbenchObjectScope("missing", options)).toBe("all");
    expect(normalizeWorkbenchTimeRange("bad")).toBe("24h");
    expect(workbenchTimeRangeLabel("7d")).toBe("近 7 天");
  });
});

