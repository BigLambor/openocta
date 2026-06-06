/** Domain inspection cron.run + polling (stage 3 frontend). */

import { loadCronRuns, type CronState } from "./cron.ts";

const POLL_INTERVAL_MS = 2000;
const MAX_POLL_ATTEMPTS = 30;

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
  if (typeof (state as any).requestUpdate === "function") {
    (state as any).requestUpdate();
  }
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
    if (typeof (state as any).requestUpdate === "function") {
      (state as any).requestUpdate();
    }
  }
}

const INSPECTION_SCENARIO_KEYS: Record<string, string> = {
  hadoop: "ops-bch-health",
  fi: "ops-fi-health",
  gbase: "ops-gbase-health",
  governance: "ops-governance-health",
  dataapps: "ops-dataapps-health",
};

function scenarioKeyForInspection(domainKey: string, jobId: string): string | undefined {
  if (INSPECTION_SCENARIO_KEYS[domainKey]) {
    return INSPECTION_SCENARIO_KEYS[domainKey];
  }
  const suffix = jobId.replace(/^job-inspect-/, "");
  return INSPECTION_SCENARIO_KEYS[suffix];
}

function isInspectionRunComplete(entry?: {
  status?: string;
  error?: string;
  summary?: string;
  result?: { score?: number | null; reportMarkdown?: string };
}): boolean {
  if (!entry) {
    return false;
  }
  if (entry.status === "error" || entry.error) {
    return true;
  }
  if (entry.result?.score != null || entry.result?.reportMarkdown) {
    return true;
  }
  if (entry.summary?.trim()) {
    return true;
  }
  return false;
}

export async function pollInspectionRuns(
  state: OpsInspectionRunHost,
  jobId: string,
  domainKey?: string,
): Promise<void> {
  const dKey = domainKey || state.tab;
  const baselineTs = state.cronRuns[0]?.runAtMs ?? state.cronRuns[0]?.ts ?? 0;

  for (let attempt = 1; attempt <= MAX_POLL_ATTEMPTS; attempt++) {
    await loadCronRuns(state, jobId);
    const latest = state.cronRuns[0] as {
      status?: string;
      error?: string;
      summary?: string;
      sessionId?: string;
      runAtMs?: number;
      ts?: number;
      result?: { score?: number | null; reportMarkdown?: string };
    };
    const runTs = latest?.runAtMs ?? latest?.ts ?? 0;
    const isNewRun = runTs > baselineTs;
    if (isNewRun && isInspectionRunComplete(latest)) {
      const selectedId = latest?.sessionId || `inspection-${dKey}-0`;
      state.opsSelectedInspectionIds = {
        ...state.opsSelectedInspectionIds,
        [dKey]: selectedId,
      };
      if (latest?.status === "error" || latest?.error) {
        throw new Error(latest.error || "巡检执行失败");
      }
      return;
    }
    if (attempt >= MAX_POLL_ATTEMPTS) {
      throw new Error("巡检超时，请稍后在巡检报告中查看结果");
    }
    await sleep(POLL_INTERVAL_MS);
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
