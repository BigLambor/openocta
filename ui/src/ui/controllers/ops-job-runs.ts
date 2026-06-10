/** JobRun query API (C2-17). */

import { authFetch, type AuthFetchHost } from "../auth-http.ts";

export type OpsJobRunRecord = {
  id: string;
  jobId: string;
  taskId?: string;
  triggerType: string;
  triggerRef?: string;
  status: string;
  startedAt: number;
  finishedAt?: number;
  error?: string;
  input?: Record<string, unknown>;
  output?: Record<string, unknown>;
  createdAt: number;
  updatedAt: number;
};

export type OpsJobRunStep = {
  id: string;
  runId: string;
  stepOrder: number;
  kind: string;
  name: string;
  status: string;
  startedAt: number;
  finishedAt?: number;
  error?: string;
  inputSummary?: string;
  outputSummary?: string;
};

export type OpsJobRunDetail = {
  run: OpsJobRunRecord;
  steps: OpsJobRunStep[];
  toolInvocations?: OpsJobRunToolInvocation[];
};

export type OpsJobRunToolInvocation = {
  id: string;
  runId: string;
  toolName: string;
  provider?: string;
  inputSummary?: string;
  outputSummary?: string;
  status: string;
  durationMs?: number;
  error?: string;
  createdAt: number;
};

export type OpsJobRunsListResponse = {
  runs: OpsJobRunRecord[];
  total: number;
};

type OpsJobRunHost = AuthFetchHost & {
  gatewayHttpUrl: string;
};

function baseUrl(host: OpsJobRunHost): string {
  return host.gatewayHttpUrl.replace(/\/$/, "");
}

export async function fetchOpsJobRuns(
  host: OpsJobRunHost,
  params: { jobId?: string; triggerType?: string; triggerRef?: string; limit?: number } = {},
): Promise<OpsJobRunsListResponse> {
  const q = new URLSearchParams();
  if (params.jobId) {
    q.set("jobId", params.jobId);
  }
  if (params.triggerType) {
    q.set("triggerType", params.triggerType);
  }
  if (params.triggerRef) {
    q.set("triggerRef", params.triggerRef);
  }
  if (params.limit && params.limit > 0) {
    q.set("limit", String(params.limit));
  }
  const suffix = q.toString() ? `?${q.toString()}` : "";
  const res = await authFetch(host, `${baseUrl(host)}/api/ops/job-runs${suffix}`);
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(body.error || `加载 JobRun 列表失败 (${res.status})`);
  }
  return (await res.json()) as OpsJobRunsListResponse;
}

export async function fetchOpsJobRunDetail(host: OpsJobRunHost, runId: string): Promise<OpsJobRunDetail> {
  const res = await authFetch(host, `${baseUrl(host)}/api/ops/job-runs/${encodeURIComponent(runId)}`);
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(body.error || `加载 JobRun 详情失败 (${res.status})`);
  }
  return (await res.json()) as OpsJobRunDetail;
}

export function formatJobRunTimestamp(ms?: number): string {
  if (!ms) {
    return "—";
  }
  const d = new Date(ms);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

export function jobRunStatusLabel(status: string): string {
  switch (status) {
    case "queued":
      return "排队中";
    case "running":
      return "运行中";
    case "waiting_approval":
      return "待审批";
    case "succeeded":
      return "成功";
    case "failed":
      return "失败";
    case "cancelled":
      return "已取消";
    default:
      return status || "—";
  }
}

export function resolveJobRunIdForCronEntry(
  entry: { runId?: string; runAtMs?: number; ts: number },
  runs: OpsJobRunRecord[],
): string | null {
  const direct = entry.runId?.trim();
  if (direct) {
    return direct;
  }
  const targetMs = entry.runAtMs ?? entry.ts;
  let best: OpsJobRunRecord | null = null;
  let bestDelta = Number.POSITIVE_INFINITY;
  for (const run of runs) {
    const delta = Math.abs(run.startedAt - targetMs);
    if (delta < bestDelta) {
      bestDelta = delta;
      best = run;
    }
  }
  if (best && bestDelta <= 5000) {
    return best.id;
  }
  return null;
}
