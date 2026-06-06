import { describe, expect, it, vi } from "vitest";
import { runDomainInspectionWithPoll, type OpsInspectionRunHost } from "./ops-inspection-run.ts";

describe("runDomainInspectionWithPoll", () => {
  it("passes the selected cluster and component to cron.run", async () => {
    const requests: Array<{ method: string; params: unknown }> = [];
    const state = {
      tab: "hadoop",
      connected: true,
      client: {
        request: vi.fn(async (method: string, params: unknown) => {
          requests.push({ method, params });
          if (method === "cron.runs") {
            return {
              entries: [
                {
                  status: "ok",
                  sessionId: "session-1",
                  runAtMs: Date.now(),
                  result: { score: 88, reportMarkdown: "巡检完成" },
                },
              ],
            };
          }
          return { ok: true };
        }),
      },
      cronRuns: [],
      opsIsInspecting: {},
      opsSelectedInspectionIds: {},
      opsSelectedEntityIds: {
        hadoop: "cluster-1#YARN%20ResourceManager",
      },
    } as OpsInspectionRunHost & { connected: boolean; opsSelectedEntityIds: Record<string, string> };

    await runDomainInspectionWithPoll(state, "job-inspect-hadoop");

    expect(requests[0]).toEqual({
      method: "cron.run",
      params: {
        id: "job-inspect-hadoop",
        mode: "force",
        domain: "hadoop",
        clusterId: "cluster-1",
        component: "YARN ResourceManager",
        scenarioKey: "ops-bch-health",
      },
    });
    expect(state.opsSelectedInspectionIds.hadoop).toBe("session-1");
    expect(state.opsIsInspecting.hadoop).toBe(false);
  });

  it("passes the GBase health scenario key to cron.run", async () => {
    const requests: Array<{ method: string; params: unknown }> = [];
    const state = {
      tab: "gbase",
      connected: true,
      client: {
        request: vi.fn(async (method: string, params: unknown) => {
          requests.push({ method, params });
          if (method === "cron.runs") {
            return {
              entries: [
                {
                  status: "ok",
                  sessionId: "session-gbase",
                  runAtMs: Date.now(),
                  result: { score: 91, reportMarkdown: "GBase 巡检完成" },
                },
              ],
            };
          }
          return { ok: true };
        }),
      },
      cronRuns: [],
      opsIsInspecting: {},
      opsSelectedInspectionIds: {},
      opsSelectedEntityIds: {
        gbase: "gbase-cluster-1",
      },
    } as OpsInspectionRunHost & { connected: boolean; opsSelectedEntityIds: Record<string, string> };

    await runDomainInspectionWithPoll(state, "job-inspect-gbase");

    expect(requests[0]).toEqual({
      method: "cron.run",
      params: {
        id: "job-inspect-gbase",
        mode: "force",
        domain: "gbase",
        clusterId: "gbase-cluster-1",
        component: "",
        scenarioKey: "ops-gbase-health",
      },
    });
    expect(state.opsSelectedInspectionIds.gbase).toBe("session-gbase");
  });
});
