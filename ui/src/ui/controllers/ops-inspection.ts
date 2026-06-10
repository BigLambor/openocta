/** Ops inspection IM delivery status (P2-A5). */

import { authFetch, type AuthFetchHost } from "../auth-http.ts";

export type OpsInspectionIMStatus = {
  imConfigured: boolean;
  channels: string[];
  lowScoreThreshold: number;
  hint?: string;
};

type OpsInspectionHost = AuthFetchHost & {
  gatewayHttpUrl: string;
};

export async function fetchOpsInspectionIMStatus(
  host: OpsInspectionHost,
): Promise<OpsInspectionIMStatus> {
  const base = host.gatewayHttpUrl.replace(/\/$/, "");
  const res = await authFetch(host, `${base}/api/ops/inspection/im-status`);
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(body.error || `加载巡检 IM 状态失败 (${res.status})`);
  }
  return (await res.json()) as OpsInspectionIMStatus;
}
