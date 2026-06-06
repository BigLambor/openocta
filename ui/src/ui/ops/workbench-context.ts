import type { OpsClusterRecord } from "../controllers/ops-clusters.ts";
import type { FlinkJob, SparkJob } from "../controllers/bch-client.ts";
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
export const WORKBENCH_OBJECT_PREFIX_FLINK_JOB = "flink_job:";
export const WORKBENCH_OBJECT_PREFIX_SPARK_JOB = "spark_job:";
export const WORKBENCH_OBJECT_PREFIX_DIRECTORY = "directory:";

export function clusterObjectId(clusterName: string): string {
  return `${WORKBENCH_OBJECT_PREFIX_CLUSTER}${encodeURIComponent(clusterName)}`;
}

export function namespaceObjectId(namespace: string): string {
  return `${WORKBENCH_OBJECT_PREFIX_NAMESPACE}${encodeURIComponent(namespace)}`;
}

export function flinkJobObjectId(jobId: string): string {
  return `${WORKBENCH_OBJECT_PREFIX_FLINK_JOB}${encodeURIComponent(jobId)}`;
}

export function sparkJobObjectId(jobId: string): string {
  return `${WORKBENCH_OBJECT_PREFIX_SPARK_JOB}${encodeURIComponent(jobId)}`;
}

export function parseWorkbenchObjectScope(id: string): {
  kind: "all" | "cluster" | "namespace" | "flink_job" | "spark_job" | "directory" | "custom";
  value: string;
  namespace?: string;
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
  if (id.startsWith(WORKBENCH_OBJECT_PREFIX_FLINK_JOB)) {
    return {
      kind: "flink_job",
      value: decodeURIComponent(id.slice(WORKBENCH_OBJECT_PREFIX_FLINK_JOB.length)),
    };
  }
  if (id.startsWith(WORKBENCH_OBJECT_PREFIX_SPARK_JOB)) {
    return {
      kind: "spark_job",
      value: decodeURIComponent(id.slice(WORKBENCH_OBJECT_PREFIX_SPARK_JOB.length)),
    };
  }
  if (id.startsWith(WORKBENCH_OBJECT_PREFIX_DIRECTORY)) {
    const rest = id.slice(WORKBENCH_OBJECT_PREFIX_DIRECTORY.length);
    const colonIdx = rest.indexOf(":");
    if (colonIdx > 0) {
      return {
        kind: "directory",
        namespace: rest.slice(0, colonIdx),
        value: decodeURIComponent(rest.slice(colonIdx + 1)),
      };
    }
  }
  return { kind: "custom", value: id };
}

export function objectOptionsForScenario(
  scenario: OpsScenario | undefined,
  clusters: OpsClusterRecord[] = [],
  flinkJobs: FlinkJob[] = [],
  sparkJobs: SparkJob[] = [],
): WorkbenchObjectOption[] {
  if (!scenario) {
    return [{ id: WORKBENCH_OBJECT_ALL, label: "全域对象", subtitle: "跨场景汇总" }];
  }
  if (scenario.id === "bch-flink-health") {
    const unique = new Map<string, OpsClusterRecord>();
    for (const cluster of clusters) {
      if (cluster.name) {
        unique.set(cluster.name, cluster);
      }
    }
    return [
      { id: WORKBENCH_OBJECT_ALL, label: "全部集群/作业", subtitle: "Flink 全域" },
      ...Array.from(unique.values()).map((cluster) => ({
        id: clusterObjectId(cluster.name),
        label: cluster.name,
        subtitle: "Flink 集群",
      })),
      ...flinkJobs.map((job) => ({
        id: flinkJobObjectId(job.id),
        label: job.name,
        subtitle: `Flink 作业 · ${job.cluster}`,
      })),
    ];
  }
  if (scenario.id === "bch-spark-tuning") {
    const unique = new Map<string, OpsClusterRecord>();
    for (const cluster of clusters) {
      if (cluster.name) {
        unique.set(cluster.name, cluster);
      }
    }
    return [
      { id: WORKBENCH_OBJECT_ALL, label: "全部集群/作业", subtitle: "Spark 全域" },
      ...Array.from(unique.values()).map((cluster) => ({
        id: clusterObjectId(cluster.name),
        label: cluster.name,
        subtitle: "Spark 集群",
      })),
      ...sparkJobs.map((job) => ({
        id: sparkJobObjectId(job.id),
        label: job.name,
        subtitle: `Spark 作业 · ${job.cluster}`,
      })),
    ];
  }
  if (scenario.id === "bch-hdfs-capacity") {
    const nsOptions: WorkbenchObjectOption[] = [];
    const namespaces = ["NS1", "NS2", "NS3", "NS4", "NS5", "NS6", "NS7", "NS8"];
    
    nsOptions.push({ id: WORKBENCH_OBJECT_ALL, label: "全部 namespace/目录", subtitle: "HDFS 全域" });
    for (const ns of namespaces) {
      nsOptions.push({
        id: namespaceObjectId(ns),
        label: ns,
        subtitle: "HDFS namespace",
      });
      for (const dir of ["/tmp", "/user", "/app"]) {
        nsOptions.push({
          id: `${WORKBENCH_OBJECT_PREFIX_DIRECTORY}${ns}:${encodeURIComponent(dir)}`,
          label: `${ns}:${dir}`,
          subtitle: "HDFS 静态治理热点目录",
        });
      }
    }
    return nsOptions;
  }
  if (scenario.objectTypes.includes("cluster")) {
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
  if (parsed.kind === "directory") {
    return {
      title: `${parsed.namespace}:${parsed.value}`,
      subtitle: "HDFS 静态治理热点目录",
    };
  }
  return { title: parsed.value, subtitle: parsed.kind };
}

export function normalizeWorkbenchTimeRange(raw: string | null | undefined): WorkbenchTimeRange {
  return WORKBENCH_TIME_RANGES.some((item) => item.id === raw) ? (raw as WorkbenchTimeRange) : "24h";
}

export function workbenchTimeRangeLabel(range: WorkbenchTimeRange): string {
  return WORKBENCH_TIME_RANGES.find((item) => item.id === range)?.label ?? range;
}
