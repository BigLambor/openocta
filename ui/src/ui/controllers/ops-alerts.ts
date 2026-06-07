/** Ops alert groups API (P2-B). */

export type OpsAlertGroupRecord = {
  id: string;
  source: string;
  domain?: string;
  title: string;
  severity: "critical" | "warning" | "info";
  status: "active" | "analyzing" | "resolved";
  originalCount: number;
  reducedTo: number;
  sessionKey?: string;
  runId?: string;
  rootCauseMarkdown?: string;
  impactMarkdown?: string;
  rootCauseSummary?: string;
  impactAnalysis?: string;
  createdAtMs: number;
  updatedAtMs: number;
};

export type OpsAlertGroupsResponse = {
  groups: OpsAlertGroupRecord[];
  total: number;
  originalTotal: number;
  mergedTotal: number;
  reductionRate: number;
  pendingActive: number;
};

type OpsAlertHost = {
  gatewayHttpUrl: string;
  rbacToken: string | null;
  settings: { token: string };
};

function authHeaders(host: OpsAlertHost): Record<string, string> {
  const headers: Record<string, string> = { Accept: "application/json" };
  if (host.rbacToken) {
    headers.Authorization = `Bearer ${host.rbacToken}`;
  } else if (host.settings.token.trim()) {
    headers.Authorization = `Bearer ${host.settings.token.trim()}`;
  }
  return headers;
}

function baseUrl(host: OpsAlertHost): string {
  return host.gatewayHttpUrl.replace(/\/$/, "");
}

export async function fetchOpsAlertGroups(
  host: OpsAlertHost,
  domain?: string,
  status?: string,
): Promise<OpsAlertGroupsResponse> {
  const params = new URLSearchParams();
  if (domain) {
    params.set("domain", domain);
  }
  if (status) {
    params.set("status", status);
  }
  const q = params.toString() ? `?${params.toString()}` : "";
  const res = await fetch(`${baseUrl(host)}/api/ops/alerts/groups${q}`, {
    headers: authHeaders(host),
  });
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(body.error || `加载告警组失败 (${res.status})`);
  }
  return (await res.json()) as OpsAlertGroupsResponse;
}

export async function patchOpsAlertGroup(
  host: OpsAlertHost,
  id: string,
  patch: { status: string; ackNote?: string; resolvedReason?: string },
): Promise<OpsAlertGroupRecord> {
  const res = await fetch(`${baseUrl(host)}/api/ops/alerts/groups/${encodeURIComponent(id)}`, {
    method: "PATCH",
    headers: { ...authHeaders(host), "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  });
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(body.error || `更新告警组失败 (${res.status})`);
  }
  return (await res.json()) as OpsAlertGroupRecord;
}

export function formatAlertTimestamp(ms: number): string {
  if (!ms) {
    return "—";
  }
  const d = new Date(ms);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

export function mapAlertGroupForUI(g: OpsAlertGroupRecord) {
  const severity =
    g.severity === "critical" || g.severity === "warning" ? g.severity : ("info" as const);
  return {
    id: g.id,
    domain: g.domain?.trim() || "",
    title: g.title,
    severity,
    timestamp: formatAlertTimestamp(g.createdAtMs),
    originalCount: g.originalCount,
    reducedTo: g.reducedTo,
    rootCause: g.rootCauseSummary?.trim() || g.rootCauseMarkdown?.trim() || (g.status === "analyzing" ? "Agent 正在分析合并告警…" : "暂无根因分析"),
    impact: g.impactAnalysis?.trim() || g.impactMarkdown?.trim() || "—",
    status: g.status === "resolved" ? ("resolved" as const) : ("active" as const),
    analysisMarkdown: g.rootCauseMarkdown?.trim() || "",
    sessionKey: g.sessionKey?.trim() || "",
  };
}
