import { describe, expect, it } from "vitest";
import type { OpsClusterRecord } from "../controllers/ops-clusters.ts";
import { findWorkbenchScenario } from "./scenario-registry.ts";
import {
  clusterObjectId,
  flinkJobObjectId,
  hdfsDirectoryObjectId,
  hdfsNamespaceObjectId,
  namespaceObjectId,
  normalizeWorkbenchObjectScope,
  normalizeWorkbenchTimeRange,
  objectOptionsForScenario,
  parseWorkbenchObjectScope,
  sparkJobObjectId,
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

    expect(options.map((option) => option.label)).toEqual(["全部集群/作业", "bch-prod-1", "bch-prod-2"]);
    expect(parseWorkbenchObjectScope(clusterObjectId("bch-prod-1"))).toEqual({
      kind: "cluster",
      value: "bch-prod-1",
    });
  });

  it("uses job ids rather than names for Flink and Spark object scopes", () => {
    const flinkScenario = findWorkbenchScenario("bch-flink-health");
    const flinkOptions = objectOptionsForScenario(flinkScenario, clusters, [
      { id: "flink-a", name: "同名作业", owner: "ops", cluster: "bch-prod-1", status: "RUNNING", score: 90, sScore: 90, pScore: 90, eScore: 90, metrics: {} as any, penalties: [], diagnosis: "", rootCause: "", rootCauseText: "", actions: [], cotSteps: {} as any },
      { id: "flink-b", name: "同名作业", owner: "ops", cluster: "bch-prod-2", status: "RUNNING", score: 80, sScore: 80, pScore: 80, eScore: 80, metrics: {} as any, penalties: [], diagnosis: "", rootCause: "", rootCauseText: "", actions: [], cotSteps: {} as any },
    ]);

    expect(flinkOptions.map((option) => option.id)).toContain(flinkJobObjectId("flink-a"));
    expect(flinkOptions.map((option) => option.id)).toContain(flinkJobObjectId("flink-b"));
    expect(parseWorkbenchObjectScope(flinkJobObjectId("flink-a"))).toEqual({
      kind: "flink_job",
      value: "flink-a",
    });

    const sparkScenario = findWorkbenchScenario("bch-spark-tuning");
    const sparkOptions = objectOptionsForScenario(sparkScenario, clusters, [], [
      { id: "spark-a", name: "同名作业", owner: "ops", cluster: "bch-prod-1", status: "SUCCEEDED", labels: [], durationSec: 1, metrics: {} as any, tuningAdvice: "" },
      { id: "spark-b", name: "同名作业", owner: "ops", cluster: "bch-prod-2", status: "FAILED", labels: [], durationSec: 2, metrics: {} as any, tuningAdvice: "" },
    ]);

    expect(sparkOptions.map((option) => option.id)).toContain(sparkJobObjectId("spark-a"));
    expect(sparkOptions.map((option) => option.id)).toContain(sparkJobObjectId("spark-b"));
    expect(parseWorkbenchObjectScope(sparkJobObjectId("spark-b"))).toEqual({
      kind: "spark_job",
      value: "spark-b",
    });
  });

  it("builds namespace scope options for HDFS capacity scenario", () => {
    const scenario = findWorkbenchScenario("bch-hdfs-capacity");
    const options = objectOptionsForScenario(scenario, clusters);

    expect(options[0]?.label).toBe("全部集群 / 全部 namespace");
    expect(options.some((option) => option.id === clusterObjectId("bch-prod-1"))).toBe(true);
    expect(options.some((option) => option.id === hdfsNamespaceObjectId("bch-prod-2", "NS8"))).toBe(true);
    expect(options.find((option) => option.id === hdfsDirectoryObjectId("bch-prod-1", "NS1", "/tmp"))?.subtitle).toBe(
      "HDFS 静态治理热点目录",
    );
    expect(parseWorkbenchObjectScope(hdfsNamespaceObjectId("bch-prod-2", "NS8"))).toEqual({
      kind: "namespace",
      cluster: "bch-prod-2",
      value: "NS8",
    });
    expect(parseWorkbenchObjectScope(hdfsDirectoryObjectId("bch-prod-1", "NS1", "/tmp"))).toEqual({
      kind: "directory",
      cluster: "bch-prod-1",
      namespace: "NS1",
      value: "/tmp",
    });
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
