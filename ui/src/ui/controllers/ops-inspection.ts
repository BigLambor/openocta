/** Ops inspection IM delivery status (P2-A5). */

export type OpsInspectionIMStatus = {
  imConfigured: boolean;
  channels: string[];
  lowScoreThreshold: number;
  hint?: string;
};

type OpsInspectionHost = {
  gatewayHttpUrl: string;
  rbacToken: string | null;
  settings: { token: string };
};

function authHeaders(host: OpsInspectionHost): Record<string, string> {
  const headers: Record<string, string> = { Accept: "application/json" };
  if (host.rbacToken) {
    headers.Authorization = `Bearer ${host.rbacToken}`;
  } else if (host.settings.token.trim()) {
    headers.Authorization = `Bearer ${host.settings.token.trim()}`;
  }
  return headers;
}

export async function fetchOpsInspectionIMStatus(
  host: OpsInspectionHost,
): Promise<OpsInspectionIMStatus> {
  const base = host.gatewayHttpUrl.replace(/\/$/, "");
  const res = await fetch(`${base}/api/ops/inspection/im-status`, {
    headers: authHeaders(host),
  });
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(body.error || `加载巡检 IM 状态失败 (${res.status})`);
  }
  return (await res.json()) as OpsInspectionIMStatus;
}
