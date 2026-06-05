/** Ops dashboard feed: alert highlights + recent global inspections. */

import type { GatewayBrowserClient } from "../gateway.ts";
import { opsDomainLabel } from "../components/domain-filter.ts";
import {
  fetchOpsAlertGroups,
  formatAlertTimestamp,
  type OpsAlertGroupRecord,
} from "./ops-alerts.ts";
import { GLOBAL_INSPECT_JOB_IDS } from "./ops-dashboard.ts";

export const DOMAIN_TABLE_THRESHOLD = 6;
export const ALERT_FETCH_CONCURRENCY = 2;
export const ALERTS_PER_DOMAIN = 2;
export const MAX_ALERT_HIGHLIGHTS = 6;
export const INSPECTION_RUNS_PER_JOB = 5;
export const INSPECTION_FETCH_CONCURRENCY = 3;
export const MAX_RECENT_INSPECTIONS = 3;

export type DashboardAlertHighlight = {
  id: string;
  domain: string;
  domainLabel: string;
  title: string;
  severity: "critical" | "warning" | "info";
  status: "active" | "analyzing" | "resolved";
  timestamp: string;
  createdAtMs: number;
};

export type DashboardInspectionRun = {
  id: string;
  jobId: string;
  domain: string;
  domainLabel: string;
  time: string;
  runAtMs: number;
  score: number | null;
  status: "healthy" | "warning" | "critical" | "unknown" | "error";
  summary: string;
};

type FeedHost = {
  connected: boolean;
  client: GatewayBrowserClient | null;
  gatewayHttpUrl: string;
  rbacToken: string | null;
  settings: { token: string };
};

type CronRunEntry = {
  ts: number;
  jobId: string;
  status?: string;
  error?: string;
  summary?: string;
  sessionId?: string;
  runAtMs?: number;
  result?: {
    score?: number | null;
    scoreStatus?: string;
    reportMarkdown?: string;
  };
};

const SEVERITY_RANK: Record<DashboardAlertHighlight["severity"], number> = {
  critical: 0,
  warning: 1,
  info: 2,
};

export function domainFromInspectJobId(jobId: string): string {
  const prefix = "job-inspect-";
  if (jobId.startsWith(prefix)) {
    return jobId.slice(prefix.length);
  }
  return jobId;
}

export function isPendingAlertStatus(status: string): boolean {
  return status === "active" || status === "analyzing";
}

export function pickTopAlertsPerDomain(
  groups: OpsAlertGroupRecord[],
  perDomain: number,
): OpsAlertGroupRecord[] {
  const pending = groups.filter((g) => isPendingAlertStatus(g.status));
  pending.sort((a, b) => {
    const sa = SEVERITY_RANK[normalizeSeverity(a.severity)] ?? 3;
    const sb = SEVERITY_RANK[normalizeSeverity(b.severity)] ?? 3;
    if (sa !== sb) {
      return sa - sb;
    }
    return b.createdAtMs - a.createdAtMs;
  });
  return pending.slice(0, perDomain);
}

export function mergeAlertHighlights(
  byDomain: Array<{ domain: string; groups: OpsAlertGroupRecord[]; pendingActive: number }>,
  maxTotal: number,
): { highlights: DashboardAlertHighlight[]; pendingByDomain: Record<string, number> } {
  const pendingByDomain: Record<string, number> = {};
  const highlights: DashboardAlertHighlight[] = [];

  for (const item of byDomain) {
    pendingByDomain[item.domain] = item.pendingActive;
    for (const group of pickTopAlertsPerDomain(item.groups, ALERTS_PER_DOMAIN)) {
      highlights.push(mapAlertHighlight(item.domain, group));
    }
  }

  highlights.sort((a, b) => {
    const sa = SEVERITY_RANK[a.severity];
    const sb = SEVERITY_RANK[b.severity];
    if (sa !== sb) {
      return sa - sb;
    }
    return b.createdAtMs - a.createdAtMs;
  });

  return {
    highlights: highlights.slice(0, maxTotal),
    pendingByDomain,
  };
}

export function mergeRecentInspectionRuns(
  entries: CronRunEntry[],
  maxTotal: number,
): DashboardInspectionRun[] {
  const mapped = entries.map((entry, idx) => mapInspectionRun(entry, idx));
  mapped.sort((a, b) => b.runAtMs - a.runAtMs);
  return mapped.slice(0, maxTotal);
}

function normalizeSeverity(sev: string): DashboardAlertHighlight["severity"] {
  if (sev === "critical") {
    return "critical";
  }
  if (sev === "warning") {
    return "warning";
  }
  return "info";
}

function mapAlertHighlight(domain: string, group: OpsAlertGroupRecord): DashboardAlertHighlight {
  return {
    id: group.id,
    domain,
    domainLabel: opsDomainLabel(domain, true),
    title: group.title,
    severity: normalizeSeverity(group.severity),
    status: group.status,
    timestamp: formatAlertTimestamp(group.createdAtMs),
    createdAtMs: group.createdAtMs,
  };
}

function mapInspectionRun(entry: CronRunEntry, idx: number): DashboardInspectionRun {
  const domain = domainFromInspectJobId(entry.jobId);
  const runAtMs = entry.runAtMs ?? entry.ts ?? 0;
  let score: number | null = null;
  let status: DashboardInspectionRun["status"] = "unknown";
  let summary = "";

  if (entry.result) {
    if (entry.result.score != null && Number.isFinite(Number(entry.result.score))) {
      score = Number(entry.result.score);
    }
    const scoreStatus = entry.result.scoreStatus || "";
    if (scoreStatus === "ok" || (score != null && score >= 90)) {
      status = "healthy";
    } else if (scoreStatus === "warning" || (score != null && score >= 75)) {
      status = "warning";
    } else if (score != null) {
      status = "critical";
    }
    summary = String(entry.result.reportMarkdown || entry.summary || "").trim();
  } else {
    const raw = String(entry.summary || "");
    const scoreMatch = raw.match(/(?:健康得分|健康度|Score)\s*[：:]\s*(\d+)/i);
    if (scoreMatch?.[1]) {
      score = parseInt(scoreMatch[1], 10);
      if (score >= 90) {
        status = "healthy";
      } else if (score >= 75) {
        status = "warning";
      } else {
        status = "critical";
      }
    }
    summary = raw;
  }

  if (entry.error || entry.status === "error") {
    status = "error";
    if (!summary) {
      summary = entry.error || "巡检执行失败";
    }
  }

  summary = summary
    .replace(/[#*`\-]/g, "")
    .replace(/\s+/g, " ")
    .trim();
  if (summary.length > 120) {
    summary = `${summary.slice(0, 120)}…`;
  }
  if (!summary) {
    summary = score != null ? `健康得分 ${score}` : "巡检记录暂无摘要";
  }

  return {
    id: entry.sessionId || `${entry.jobId}-${runAtMs}-${idx}`,
    jobId: entry.jobId,
    domain,
    domainLabel: opsDomainLabel(domain, true),
    time: formatAlertTimestamp(runAtMs),
    runAtMs,
    score,
    status,
    summary,
  };
}

async function mapWithConcurrency<T, R>(
  items: T[],
  limit: number,
  fn: (item: T) => Promise<R>,
): Promise<R[]> {
  if (items.length === 0) {
    return [];
  }
  const results = new Array<R>(items.length);
  let cursor = 0;
  const workers = Array.from({ length: Math.min(limit, items.length) }, async () => {
    while (cursor < items.length) {
      const index = cursor++;
      results[index] = await fn(items[index]);
    }
  });
  await Promise.all(workers);
  return results;
}

export async function fetchDashboardAlertHighlights(
  host: FeedHost,
  managedDomains: string[],
): Promise<{ highlights: DashboardAlertHighlight[]; pendingByDomain: Record<string, number> }> {
  if (managedDomains.length === 0) {
    return { highlights: [], pendingByDomain: {} };
  }

  const uniqueDomains = [...new Set(managedDomains.map((d) => d.trim().toLowerCase()).filter(Boolean))];
  const rows = await mapWithConcurrency(uniqueDomains, ALERT_FETCH_CONCURRENCY, async (domain) => {
    try {
      const res = await fetchOpsAlertGroups(host, domain);
      return { domain, groups: res.groups, pendingActive: res.pendingActive };
    } catch {
      return { domain, groups: [], pendingActive: 0 };
    }
  });

  return mergeAlertHighlights(rows, MAX_ALERT_HIGHLIGHTS);
}

async function fetchInspectionRunsForJob(
  host: FeedHost,
  jobId: string,
): Promise<CronRunEntry[]> {
  if (!host.client || !host.connected) {
    return [];
  }
  try {
    const res = await host.client.request<{ entries?: CronRunEntry[] }>("cron.runs", {
      id: jobId,
      limit: INSPECTION_RUNS_PER_JOB,
    });
    return Array.isArray(res.entries) ? res.entries : [];
  } catch {
    return [];
  }
}

export async function fetchDashboardRecentInspections(
  host: FeedHost,
): Promise<DashboardInspectionRun[]> {
  const jobIds = [...GLOBAL_INSPECT_JOB_IDS];
  const batches = await mapWithConcurrency(jobIds, INSPECTION_FETCH_CONCURRENCY, (jobId) =>
    fetchInspectionRunsForJob(host, jobId),
  );
  const merged = batches.flat();
  return mergeRecentInspectionRuns(merged, MAX_RECENT_INSPECTIONS);
}

export async function loadOpsDashboardFeed(
  host: FeedHost,
  managedDomains: string[],
): Promise<{
  highlights: DashboardAlertHighlight[];
  pendingByDomain: Record<string, number>;
  inspections: DashboardInspectionRun[];
}> {
  const [alertResult, inspections] = await Promise.all([
    fetchDashboardAlertHighlights(host, managedDomains),
    fetchDashboardRecentInspections(host),
  ]);
  return {
    highlights: alertResult.highlights,
    pendingByDomain: alertResult.pendingByDomain,
    inspections,
  };
}
