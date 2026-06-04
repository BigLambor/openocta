export interface BchClusterHealth {
  id: string;
  name: string;
  region: string;
  status: "healthy" | "warning" | "critical";
  score: number;
  nodeCount: number;
  activeAlerts: number;
  cpuUsedPercent: number;
  memUsedPercent: number;
  dfsUsedPercent: number;
  metrics: Record<string, any>;
}

export interface FlinkJobMetric {
  lagTrend: number;
  maxLag: number;
  avgLag: number;
  isBP: boolean;
  cpuMax: number;
  cpuAvg: number;
  heapMax: number;
  fullGcCount: number;
  restarts: number;
}

export interface FlinkPenalty {
  item: string;
  deduction: number;
  type: "fatal" | "warning" | "info";
}

export interface FlinkCotStep {
  text: string;
  state: "active" | "warning" | "critical";
}

export interface FlinkCotSteps {
  step1: FlinkCotStep;
  step2: FlinkCotStep;
  step3: FlinkCotStep;
}

export interface FlinkJob {
  id: string;
  name: string;
  owner: string;
  cluster: string;
  status: string;
  score: number;
  sScore: number;
  pScore: number;
  eScore: number;
  metrics: FlinkJobMetric;
  penalties: FlinkPenalty[];
  diagnosis: string;
  rootCause: string;
  rootCauseText: string;
  actions: string[];
  cotSteps: FlinkCotSteps;
}

export interface SparkJobMetric {
  executorMemoryOverheadMB: number;
  maxTaskDurationSec: number;
  avgTaskDurationSec: number;
  totalTasks: number;
  failedTasks: number;
  cpuSkewRatio: number;
  memorySkewRatio: number;
  inputBytes: number;
  shuffleReadBytes: number;
  shuffleWriteBytes: number;
}

export interface SparkJob {
  id: string;
  name: string;
  owner: string;
  cluster: string;
  status: "SUCCEEDED" | "RUNNING" | "FAILED";
  labels: string[];
  durationSec: number;
  metrics: SparkJobMetric;
  tuningAdvice: string;
}

export interface HdfsFsImageDepthStats {
  depth: string;
  count: number;
  percent: number;
}

export interface HdfsFsImageSizeStats {
  size: string;
  count: number;
  percent: number;
}

export interface HdfsFsImageUserStats {
  user: string;
  files: number;
  percent: number;
  size: string;
}

export interface HdfsFsImageTimeStats {
  period: string;
  count: number;
  percent: number;
}

export interface HdfsFsImageFileTypeStats {
  ext: string;
  count: number;
  percent: number;
}

export interface HdfsFsImagePathPattern {
  path: string;
  count: number;
  percent: number;
}

export interface HdfsFsImageStats {
  namespace: string;
  totalRecords: string;
  totalFiles: string;
  totalDirs: string;
  totalSize: string;
  avgFileSize: string;
  maxDepth: string;
  processingTime: string;
  processingSpeed: string;
  depthData: HdfsFsImageDepthStats[];
  sizeData: HdfsFsImageSizeStats[];
  userData: HdfsFsImageUserStats[];
  modifyData: HdfsFsImageTimeStats[];
  accessData: HdfsFsImageTimeStats[];
  fileTypeData: HdfsFsImageFileTypeStats[];
  pathData: HdfsFsImagePathPattern[];
  zeroByteFiles: number;
  trashFiles: number;
}

export interface BchEmployeeTask {
  time: string;
  task: string;
  result: string;
}

export interface BchEmployee {
  id: string;
  name: string;
  status: "idle" | "working";
  statusDesc: string;
  description: string;
  skills: string[];
  tools: string[];
  recentTasks: BchEmployeeTask[];
}

interface BchClientHost {
  gatewayHttpUrl: string;
  rbacToken: string | null;
  settings: { token: string };
}

function authHeaders(host: BchClientHost): Record<string, string> {
  const headers: Record<string, string> = { Accept: "application/json" };
  if (host.rbacToken) {
    headers.Authorization = `Bearer ${host.rbacToken}`;
  } else if (host.settings.token.trim()) {
    headers.Authorization = `Bearer ${host.settings.token.trim()}`;
  }
  return headers;
}

function baseUrl(host: BchClientHost): string {
  return host.gatewayHttpUrl.replace(/\/$/, "");
}

export async function fetchBchClustersHealth(host: BchClientHost): Promise<BchClusterHealth[]> {
  const res = await fetch(`${baseUrl(host)}/api/ops/bch/clusters/health`, {
    headers: authHeaders(host),
  });
  if (!res.ok) {
    throw new Error(`加载 BCH 集群健康状态失败 (${res.status})`);
  }
  const data = await res.json();
  return Array.isArray(data) ? data : [];
}

export async function fetchBchFlinkJobs(host: BchClientHost): Promise<FlinkJob[]> {
  const res = await fetch(`${baseUrl(host)}/api/ops/bch/flink/jobs`, {
    headers: authHeaders(host),
  });
  if (!res.ok) {
    throw new Error(`加载 Flink 作业列表失败 (${res.status})`);
  }
  const data = await res.json();
  return Array.isArray(data) ? data : [];
}

export async function fetchBchFlinkJobConfig(host: BchClientHost, id: string): Promise<any> {
  const res = await fetch(`${baseUrl(host)}/api/ops/bch/flink/jobs/${encodeURIComponent(id)}/config`, {
    headers: authHeaders(host),
  });
  if (!res.ok) {
    throw new Error(`提取 Flink 作业运行配置失败 (${res.status})`);
  }
  return res.json();
}

export async function diagnoseBchFlinkJob(host: BchClientHost, id: string): Promise<FlinkJob> {
  const res = await fetch(`${baseUrl(host)}/api/ops/bch/flink/jobs/${encodeURIComponent(id)}/diagnose`, {
    method: "POST",
    headers: authHeaders(host),
  });
  if (!res.ok) {
    throw new Error(`智能诊断 Flink 作业失败 (${res.status})`);
  }
  return res.json();
}

export async function fetchBchSparkJobs(host: BchClientHost): Promise<SparkJob[]> {
  const res = await fetch(`${baseUrl(host)}/api/ops/bch/spark/jobs`, {
    headers: authHeaders(host),
  });
  if (!res.ok) {
    throw new Error(`加载 Spark 作业列表失败 (${res.status})`);
  }
  const data = await res.json();
  return Array.isArray(data) ? data : [];
}

export async function tuneBchSparkJob(host: BchClientHost, id: string): Promise<SparkJob> {
  const res = await fetch(`${baseUrl(host)}/api/ops/bch/spark/jobs/${encodeURIComponent(id)}/tune`, {
    method: "POST",
    headers: authHeaders(host),
  });
  if (!res.ok) {
    throw new Error(`智能调优 Spark 作业失败 (${res.status})`);
  }
  return res.json();
}

export async function fetchBchHdfsFsImage(host: BchClientHost, namespace: string): Promise<HdfsFsImageStats> {
  const res = await fetch(`${baseUrl(host)}/api/ops/bch/hdfs/fsimage?namespace=${encodeURIComponent(namespace)}`, {
    headers: authHeaders(host),
  });
  if (!res.ok) {
    throw new Error(`加载 HDFS FSImage 分析失败 (${res.status})`);
  }
  const data = await res.json();
  return {
    namespace: data.namespace || namespace,
    totalRecords: data.totalRecords || "0",
    totalFiles: data.totalFiles || "0",
    totalDirs: data.totalDirs || "0",
    totalSize: data.totalSize || "0 B",
    avgFileSize: data.avgFileSize || "0 B",
    maxDepth: data.maxDepth || "0",
    processingTime: data.processingTime || "0",
    processingSpeed: data.processingSpeed || "0",
    depthData: Array.isArray(data.depthData) ? data.depthData : [],
    sizeData: Array.isArray(data.sizeData) ? data.sizeData : [],
    userData: Array.isArray(data.userData) ? data.userData : [],
    modifyData: Array.isArray(data.modifyData) ? data.modifyData : [],
    accessData: Array.isArray(data.accessData) ? data.accessData : [],
    fileTypeData: Array.isArray(data.fileTypeData) ? data.fileTypeData : [],
    pathData: Array.isArray(data.pathData) ? data.pathData : [],
    zeroByteFiles: typeof data.zeroByteFiles === "number" ? data.zeroByteFiles : 0,
    trashFiles: typeof data.trashFiles === "number" ? data.trashFiles : 0,
  };
}

export async function fetchBchEmployees(host: BchClientHost): Promise<BchEmployee[]> {
  const res = await fetch(`${baseUrl(host)}/api/ops/bch/employees`, {
    headers: authHeaders(host),
  });
  if (!res.ok) {
    throw new Error(`加载 BCH 数字员工失败 (${res.status})`);
  }
  const data = await res.json();
  return Array.isArray(data) ? data : [];
}

export async function chatBchFlinkJob(host: BchClientHost, id: string, message: string): Promise<string> {
  const res = await fetch(`${baseUrl(host)}/api/ops/bch/flink/jobs/${encodeURIComponent(id)}/chat`, {
    method: "POST",
    headers: {
      ...authHeaders(host),
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ message }),
  });
  if (!res.ok) {
    throw new Error(`数字员工回复失败 (${res.status})`);
  }
  const data = await res.json();
  return data.reply || "";
}
