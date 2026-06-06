/** Domain inspection cron.run + polling (stage 3 frontend). */

import { loadCronRuns, type CronState } from "./cron.ts";

const POLL_INTERVAL_MS = 2000;
const MAX_POLL_ATTEMPTS = 15;

export type OpsInspectionRunHost = CronState & {
  tab: string;
  client?: { request: (method: string, params: unknown) => Promise<unknown> } | null;
  cronRuns: Array<{ status?: string; sessionId?: string }>;
  opsIsInspecting: Record<string, boolean>;
  opsSelectedInspectionIds: Record<string, string | null>;
};

export async function runDomainInspectionWithPoll(
  state: OpsInspectionRunHost,
  jobId: string,
  domainKeyOverride?: string,
): Promise<void> {
  if (!state.client) {
    throw new Error("网关未连接");
  }
  const domainKey = domainKeyOverride || state.tab;
  state.opsIsInspecting = { ...state.opsIsInspecting, [domainKey]: true };
  try {
    const entityId = (state as any).opsSelectedEntityIds?.[domainKey] ?? "all";
    let clusterId = "";
    let component = "";
    if (entityId && entityId !== "all") {
      const parts = entityId.split("#");
      clusterId = parts[0] ?? "";
      component = parts[1] ? decodeURIComponent(parts[1]) : "";
    }
    const scenarioKey = scenarioKeyForInspection(domainKey, jobId);
    await state.client.request("cron.run", {
      id: jobId,
      mode: "force",
      domain: domainKey,
      clusterId,
      component,
      ...(scenarioKey ? { scenarioKey } : {}),
    });
    await pollInspectionRuns(state, jobId, domainKey);
  } finally {
    state.opsIsInspecting = { ...state.opsIsInspecting, [domainKey]: false };
  }
}

function scenarioKeyForInspection(domainKey: string, jobId: string): string | undefined {
  if (domainKey === "gbase" || jobId === "job-inspect-gbase") {
    return "ops-gbase-health";
  }
  return undefined;
}

export async function pollInspectionRuns(
  state: OpsInspectionRunHost,
  jobId: string,
  domainKey?: string,
): Promise<void> {
  for (let attempt = 1; attempt <= MAX_POLL_ATTEMPTS; attempt++) {
    await loadCronRuns(state, jobId);
    const latest = state.cronRuns[0];
    if (latest?.status === "ok" || attempt >= MAX_POLL_ATTEMPTS) {
      if (latest?.sessionId) {
        const dKey = domainKey || state.tab;
        state.opsSelectedInspectionIds = {
          ...state.opsSelectedInspectionIds,
          [dKey]: latest.sessionId,
        };
      }
      return;
    }
    await sleep(POLL_INTERVAL_MS);
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
