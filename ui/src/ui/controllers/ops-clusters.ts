/** Ops cluster assets API (P1). */

export type OpsClusterRecord = {
  id: string;
  name: string;
  domain: string;
  region?: string;
  nodeCount: number;
  components: string[];
  owner?: string;
  status: "healthy" | "warning" | "critical" | "unknown" | "inactive";
  description?: string;
  createdAtMs: number;
  updatedAtMs: number;
  monitorLabels?: string;
  vmUrlRef?: string;
  metricsBaseUrl?: string;
  jmxUrl?: string;
  fiManagerUrl?: string;
  gbaseDsnRef?: string;
  credentialsRef?: string;
};

export type OpsDashboardSummary = {
  totalClusters: number;
  healthyClusters: number;
  warningClusters: number;
  criticalClusters: number;
  pendingAlerts: number;
  vmConfigured?: boolean;
  domains: Array<{
    domain: string;
    clusterCount: number;
    healthyCount: number;
    warningCount: number;
    criticalCount: number;
    healthScore?: number | null;
    healthScoreSource?: string;
    healthScoreNote?: string;
    note?: string;
  }>;
};

type OpsClusterHost = {
  gatewayHttpUrl: string;
  rbacToken: string | null;
  settings: { token: string };
};

function authHeaders(host: OpsClusterHost): Record<string, string> {
  const headers: Record<string, string> = { Accept: "application/json" };
  if (host.rbacToken) {
    headers.Authorization = `Bearer ${host.rbacToken}`;
  } else if (host.settings.token.trim()) {
    headers.Authorization = `Bearer ${host.settings.token.trim()}`;
  }
  return headers;
}

function baseUrl(host: OpsClusterHost): string {
  return host.gatewayHttpUrl.replace(/\/$/, "");
}

export async function fetchOpsClusters(
  host: OpsClusterHost,
  domain?: string,
): Promise<OpsClusterRecord[]> {
  const q = domain ? `?domain=${encodeURIComponent(domain)}` : "";
  const res = await fetch(`${baseUrl(host)}/api/ops/clusters${q}`, {
    headers: authHeaders(host),
  });
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(body.error || `加载集群列表失败 (${res.status})`);
  }
  const data = (await res.json()) as { clusters: OpsClusterRecord[] };
  return data.clusters ?? [];
}

export async function fetchOpsDashboardSummary(host: OpsClusterHost): Promise<OpsDashboardSummary> {
  const res = await fetch(`${baseUrl(host)}/api/ops/dashboard/summary`, {
    headers: authHeaders(host),
  });
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(body.error || `加载运维概览失败 (${res.status})`);
  }
  return (await res.json()) as OpsDashboardSummary;
}

export type OpsCMDBSyncResult = {
  created: number;
  updated: number;
  skipped: number;
  total: number;
  source: string;
  strategy?: string;
  dryRun?: boolean;
  errors?: Array<{ rowIndex: number; name: string; error: string }>;
};

export async function syncOpsClustersFromCMDB(host: OpsClusterHost): Promise<OpsCMDBSyncResult> {
  const res = await fetch(`${baseUrl(host)}/api/ops/clusters/sync-cmdb`, {
    method: "POST",
    headers: { ...authHeaders(host), "Content-Type": "application/json" },
    body: JSON.stringify({}),
  });
  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(err.error || `同步 CMDB 失败 (${res.status})`);
  }
  return (await res.json()) as OpsCMDBSyncResult;
}

export async function createOpsCluster(
  host: OpsClusterHost,
  body: {
    name: string;
    domain: string;
    region?: string;
    nodeCount: number;
    components: string[];
    owner?: string;
    status: string;
    description?: string;
    monitorLabels?: string;
    vmUrlRef?: string;
    metricsBaseUrl?: string;
    jmxUrl?: string;
    fiManagerUrl?: string;
    gbaseDsnRef?: string;
    credentialsRef?: string;
  },
): Promise<OpsClusterRecord> {
  const res = await fetch(`${baseUrl(host)}/api/ops/clusters`, {
    method: "POST",
    headers: { ...authHeaders(host), "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(err.error || `创建集群失败 (${res.status})`);
  }
  return (await res.json()) as OpsClusterRecord;
}
