/** Ops dashboard actions (P1-6). */

import type { GatewayBrowserClient } from "../gateway.ts";

export const GLOBAL_INSPECT_JOB_IDS = [
  "job-inspect-hadoop",
  "job-inspect-fi",
  "job-inspect-gbase",
  "job-inspect-governance",
  "job-inspect-dataapps",
] as const;

type InspectHost = {
  connected: boolean;
  client: GatewayBrowserClient | null;
};

export async function runGlobalInspection(
  host: InspectHost,
): Promise<{ started: number; failed: number }> {
  if (!host.client || !host.connected) {
    throw new Error("网关未连接，无法启动巡检");
  }
  let started = 0;
  let failed = 0;
  const results = await Promise.allSettled(
    GLOBAL_INSPECT_JOB_IDS.map((id) =>
      host.client!.request("cron.run", { id, mode: "force" }),
    ),
  );
  for (const r of results) {
    if (r.status === "fulfilled") {
      started++;
    } else {
      failed++;
    }
  }
  if (started === 0) {
    throw new Error("未能启动任何巡检任务，请确认定时任务已初始化");
  }
  return { started, failed };
}
