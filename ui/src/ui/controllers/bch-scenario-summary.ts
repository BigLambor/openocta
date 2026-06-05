import {
  fetchBchClustersHealth,
  fetchBchFlinkJobs,
  fetchBchHdfsFsImage,
  fetchBchSparkJobs,
  type BchClusterHealth,
  type FlinkJob,
  type HdfsFsImageStats,
  type SparkJob,
} from "./bch-client.ts";

type BchScenarioSummaryHost = {
  gatewayHttpUrl: string;
  rbacToken: string | null;
  settings: { token: string };
};

export type BchScenarioStatus = "healthy" | "warning" | "critical" | "unknown";

export type BchScenarioCardSummary = {
  id: "flink-health" | "spark-tuning" | "hdfs-storage";
  title: string;
  score: number | null;
  status: BchScenarioStatus;
  primaryMetric: string;
  secondaryMetric: string;
  description: string;
  summary: string;
  primaryActionLabel: string;
  primaryView: "diagnosis" | "governance" | "capacity" | "change";
  secondaryActionLabel?: string;
  secondaryView?: "diagnosis" | "governance" | "capacity" | "change";
  workflowType: string;
  capability: string;
  initialQuestion: string;
};

export type BchDomainScenarioSummary = {
  domain: "hadoop";
  updatedAtMs: number;
  source: "bch-api";
  scenarios: BchScenarioCardSummary[];
  aggregate: {
    jobHealthScore: number | null;
    storageHealthScore: number | null;
    serviceHealthScore: number | null;
  };
  errors: string[];
};

function avg(values: Array<number | null | undefined>): number | null {
  const nums = values.filter((n): n is number => typeof n === "number" && Number.isFinite(n));
  if (nums.length === 0) {
    return null;
  }
  return Math.round(nums.reduce((acc, n) => acc + n, 0) / nums.length);
}

function clampScore(score: number): number {
  return Math.max(0, Math.min(100, Math.round(score)));
}

function statusFromScore(score: number | null): BchScenarioStatus {
  if (score == null) return "unknown";
  if (score >= 90) return "healthy";
  if (score >= 75) return "warning";
  return "critical";
}

function formatScore(score: number | null): string {
  return score == null ? "暂无评分" : `${score}分`;
}

function summarizeFlink(jobs: FlinkJob[]): BchScenarioCardSummary {
  const score = avg(jobs.map((job) => job.score));
  const runningCount = jobs.filter((job) => String(job.status).toUpperCase() === "RUNNING").length;
  const restartIssues = jobs.filter((job) => (job.metrics?.restarts ?? 0) > 0).length;
  const gcIssues = jobs.filter((job) => (job.metrics?.fullGcCount ?? 0) > 0).length;
  const backpressureIssues = jobs.filter((job) => job.metrics?.isBP).length;
  const issueCount = restartIssues + gcIssues + backpressureIssues;
  const issueText =
    issueCount > 0
      ? `${restartIssues} 个 Restart、${gcIssues} 个 GC、${backpressureIssues} 个反压风险`
      : "未发现 Restart/GC/反压风险";
  const effectiveScore = score ?? (jobs.length === 0 ? null : clampScore(100 - issueCount * 8));
  return {
    id: "flink-health",
    title: "Flink 作业健康度",
    score: effectiveScore,
    status: statusFromScore(effectiveScore),
    primaryMetric: `运行中作业: ${runningCount}/${jobs.length}`,
    secondaryMetric: issueText,
    description:
      jobs.length === 0
        ? "尚未接入 Flink 作业运行数据。"
        : issueCount > 0
          ? "作业运行存在稳定性风险，建议进入诊断中心核验证据。"
          : "Flink 作业整体健康，可持续观察延迟、反压和重启趋势。",
    summary: `Flink 作业健康度: ${formatScore(effectiveScore)}，运行中 ${runningCount}/${jobs.length}，${issueText}。`,
    primaryActionLabel: "根因诊断",
    primaryView: "diagnosis",
    secondaryActionLabel: "治理建议",
    secondaryView: "governance",
    workflowType: "diagnosis",
    capability: "observability-alert",
    initialQuestion: "请分析 Flink 作业健康度，重点关注 Restart、GC、反压和延迟风险。",
  };
}

function summarizeSpark(jobs: SparkJob[]): BchScenarioCardSummary {
  const failedCount = jobs.filter((job) => job.status === "FAILED").length;
  const runningCount = jobs.filter((job) => job.status === "RUNNING").length;
  const tuningCandidates = jobs.filter((job) => {
    const m = job.metrics;
    return (
      job.status === "FAILED" ||
      (m?.failedTasks ?? 0) > 0 ||
      (m?.cpuSkewRatio ?? 0) >= 2 ||
      (m?.memorySkewRatio ?? 0) >= 2 ||
      (m?.executorMemoryOverheadMB ?? 0) >= 1024 ||
      Boolean(job.tuningAdvice?.trim())
    );
  }).length;
  const score =
    jobs.length === 0
      ? null
      : clampScore(100 - failedCount * 12 - tuningCandidates * 5 - runningCount * 2);
  const savingPercent = tuningCandidates === 0 ? 0 : Math.min(30, tuningCandidates * 6);
  return {
    id: "spark-tuning",
    title: "Spark 参数调优",
    score,
    status: statusFromScore(score),
    primaryMetric: `调优候选: ${tuningCandidates}/${jobs.length}`,
    secondaryMetric:
      tuningCandidates > 0
        ? `预计资源节省约 ${savingPercent}%`
        : "未发现明显参数调优机会",
    description:
      jobs.length === 0
        ? "尚未接入 Spark 作业运行数据。"
        : tuningCandidates > 0
          ? "存在失败、倾斜或内存开销偏高的作业，建议进入治理中心生成调优处方。"
          : "Spark 作业运行稳定，暂未发现明显资源浪费。",
    summary: `Spark 参数调优: ${formatScore(score)}，调优候选 ${tuningCandidates}/${jobs.length}，失败作业 ${failedCount} 个。`,
    primaryActionLabel: "参数优化",
    primaryView: "governance",
    secondaryActionLabel: "变更护航",
    secondaryView: "change",
    workflowType: "optimization",
    capability: "governance-optimization",
    initialQuestion: "请分析 Spark 参数调优建议，重点关注 executor 内存、并行度、任务倾斜和失败任务。",
  };
}

function parseNumericText(value: string | undefined): number | null {
  if (!value) return null;
  const match = /[\d.]+/.exec(value.replace(/,/g, ""));
  return match ? Number(match[0]) : null;
}

function summarizeHdfs(stats: HdfsFsImageStats | null): BchScenarioCardSummary {
  const smallFilePercent =
    stats?.sizeData
      ?.filter((row) => /<\s*1\s*(kb|mb)|小文件/i.test(row.size))
      .reduce((acc, row) => acc + (row.percent || 0), 0) ?? 0;
  const maxDepth = parseNumericText(stats?.maxDepth) ?? 0;
  const zeroByteFiles = stats?.zeroByteFiles ?? 0;
  const trashFiles = stats?.trashFiles ?? 0;
  const riskPenalty = smallFilePercent + Math.max(0, maxDepth - 12) * 2 + Math.min(20, zeroByteFiles / 1000) + Math.min(15, trashFiles / 100000);
  const score = stats ? clampScore(100 - riskPenalty) : null;
  const riskText =
    stats == null
      ? "暂无 FSImage 元数据"
      : `小文件 ${Math.round(smallFilePercent)}%，最大目录深度 ${stats.maxDepth}`;
  return {
    id: "hdfs-storage",
    title: "HDFS 存储与元数据",
    score,
    status: statusFromScore(score),
    primaryMetric: riskText,
    secondaryMetric: stats ? `零字节 ${zeroByteFiles}，回收站 ${trashFiles}` : "等待接入 HDFS FSImage 分析",
    description:
      stats == null
        ? "尚未接入 HDFS FSImage 元数据。"
        : score != null && score < 75
          ? "元数据存在小文件、目录深度或回收站堆积风险，建议进入容量性能中心处理。"
          : "HDFS 元数据风险可控，建议持续巡检小文件和目录深度趋势。",
    summary: `HDFS 存储与元数据: ${formatScore(score)}，${riskText}，零字节文件 ${zeroByteFiles}，回收站文件 ${trashFiles}。`,
    primaryActionLabel: "容量优化",
    primaryView: "capacity",
    workflowType: "capacity",
    capability: "capacity-performance-cost",
    initialQuestion: "请分析 HDFS 存储健康度，评估小文件比例、FSImage、目录深度和 NameNode 内存风险。",
  };
}

function summarizeServiceHealth(clusters: BchClusterHealth[]): number | null {
  return avg(clusters.map((cluster) => cluster.score));
}

export async function fetchBchDomainScenarioSummary(
  host: BchScenarioSummaryHost,
): Promise<BchDomainScenarioSummary> {
  const errors: string[] = [];
  const [clustersResult, flinkResult, sparkResult, hdfsResult] = await Promise.allSettled([
    fetchBchClustersHealth(host),
    fetchBchFlinkJobs(host),
    fetchBchSparkJobs(host),
    fetchBchHdfsFsImage(host, "default"),
  ]);

  const clusters =
    clustersResult.status === "fulfilled"
      ? clustersResult.value
      : (errors.push(clustersResult.reason instanceof Error ? clustersResult.reason.message : String(clustersResult.reason)), []);
  const flinkJobs =
    flinkResult.status === "fulfilled"
      ? flinkResult.value
      : (errors.push(flinkResult.reason instanceof Error ? flinkResult.reason.message : String(flinkResult.reason)), []);
  const sparkJobs =
    sparkResult.status === "fulfilled"
      ? sparkResult.value
      : (errors.push(sparkResult.reason instanceof Error ? sparkResult.reason.message : String(sparkResult.reason)), []);
  const hdfsStats =
    hdfsResult.status === "fulfilled"
      ? hdfsResult.value
      : (errors.push(hdfsResult.reason instanceof Error ? hdfsResult.reason.message : String(hdfsResult.reason)), null);

  const flink = summarizeFlink(flinkJobs);
  const spark = summarizeSpark(sparkJobs);
  const hdfs = summarizeHdfs(hdfsStats);

  return {
    domain: "hadoop",
    updatedAtMs: Date.now(),
    source: "bch-api",
    scenarios: [flink, spark, hdfs],
    aggregate: {
      jobHealthScore: avg([flink.score, spark.score]),
      storageHealthScore: hdfs.score,
      serviceHealthScore: summarizeServiceHealth(clusters),
    },
    errors,
  };
}
