import type { OpsClusterRecord } from "../controllers/ops-clusters.ts";
import type { OpsScenario, WorkbenchTimeRange } from "./scenario-registry.ts";
import { WORKBENCH_TIME_RANGES } from "./scenario-registry.ts";

export type WorkbenchObjectOption = {
  id: string;
  label: string;
  subtitle?: string;
};

export const WORKBENCH_OBJECT_ALL = "all";
export const WORKBENCH_OBJECT_PREFIX_CLUSTER = "cluster:";
export const WORKBENCH_OBJECT_PREFIX_NAMESPACE = "namespace:";

export function clusterObjectId(clusterName: string): string {
  return `${WORKBENCH_OBJECT_PREFIX_CLUSTER}${encodeURIComponent(clusterName)}`;
}

export function namespaceObjectId(namespace: string): string {
  return `${WORKBENCH_OBJECT_PREFIX_NAMESPACE}${encodeURIComponent(namespace)}`;
}

export function parseWorkbenchObjectScope(id: string): {
  kind: "all" | "cluster" | "namespace" | "custom";
  value: string;
} {
  if (!id || id === WORKBENCH_OBJECT_ALL) {
    return { kind: "all", value: WORKBENCH_OBJECT_ALL };
  }
  if (id.startsWith(WORKBENCH_OBJECT_PREFIX_CLUSTER)) {
    return {
      kind: "cluster",
      value: decodeURIComponent(id.slice(WORKBENCH_OBJECT_PREFIX_CLUSTER.length)),
    };
  }
  if (id.startsWith(WORKBENCH_OBJECT_PREFIX_NAMESPACE)) {
    return {
      kind: "namespace",
      value: decodeURIComponent(id.slice(WORKBENCH_OBJECT_PREFIX_NAMESPACE.length)),
    };
  }
  return { kind: "custom", value: id };
}

export function objectOptionsForScenario(
  scenario: OpsScenario | undefined,
  clusters: OpsClusterRecord[] = [],
): WorkbenchObjectOption[] {
  if (!scenario) {
    return [{ id: WORKBENCH_OBJECT_ALL, label: "全域对象", subtitle: "跨场景汇总" }];
  }
  if (scenario.objectTypes.some((type) => type.startsWith("hdfs_"))) {
    return [
      { id: WORKBENCH_OBJECT_ALL, label: "全部 namespace", subtitle: "HDFS 全域" },
      ...["NS1", "NS2", "NS3", "NS4", "NS5", "NS6", "NS7", "NS8"].map((ns) => ({
        id: namespaceObjectId(ns),
        label: ns,
        subtitle: "HDFS namespace",
      })),
    ];
  }
  if (scenario.objectTypes.includes("cluster") || scenario.objectTypes.includes("flink_job") || scenario.objectTypes.includes("spark_job")) {
    const unique = new Map<string, OpsClusterRecord>();
    for (const cluster of clusters) {
      if (cluster.name) {
        unique.set(cluster.name, cluster);
      }
    }
    return [
      { id: WORKBENCH_OBJECT_ALL, label: "全部集群", subtitle: "业务域全域" },
      ...Array.from(unique.values()).map((cluster) => ({
        id: clusterObjectId(cluster.name),
        label: cluster.name,
        subtitle: cluster.region ? `${cluster.region} · ${cluster.status}` : cluster.status,
      })),
    ];
  }
  return [{ id: WORKBENCH_OBJECT_ALL, label: "全域对象", subtitle: scenario.objectTypes.join(" / ") }];
}

export function normalizeWorkbenchObjectScope(
  scope: string | null | undefined,
  options: WorkbenchObjectOption[],
): string {
  const raw = scope || WORKBENCH_OBJECT_ALL;
  return options.some((option) => option.id === raw) ? raw : options[0]?.id ?? WORKBENCH_OBJECT_ALL;
}

export function formatWorkbenchObjectScope(scope: string, options: WorkbenchObjectOption[]): {
  title: string;
  subtitle: string;
} {
  const matched = options.find((option) => option.id === scope);
  if (matched) {
    return {
      title: matched.label,
      subtitle: matched.subtitle ?? "当前对象范围",
    };
  }
  const parsed = parseWorkbenchObjectScope(scope);
  return { title: parsed.value, subtitle: parsed.kind };
}

export function normalizeWorkbenchTimeRange(raw: string | null | undefined): WorkbenchTimeRange {
  return WORKBENCH_TIME_RANGES.some((item) => item.id === raw) ? (raw as WorkbenchTimeRange) : "24h";
}

export function workbenchTimeRangeLabel(range: WorkbenchTimeRange): string {
  return WORKBENCH_TIME_RANGES.find((item) => item.id === range)?.label ?? range;
}

