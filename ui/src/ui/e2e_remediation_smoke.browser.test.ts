import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { OpenClawApp } from "./app.ts";
import { GatewayBrowserClient } from "./gateway.ts";
import "../styles.css";

const originalConnect = OpenClawApp.prototype.connect;
const originalCheckRbacSession = OpenClawApp.prototype.checkRbacSession;
const originalStart = GatewayBrowserClient.prototype.start;
const originalFetch = window.fetch;
const originalPrompt = window.prompt;

function mountApp(pathname: string) {
  window.history.replaceState({}, "", pathname);
  const app = document.createElement("openclaw-app") as OpenClawApp;
  document.body.append(app);
  return app;
}

beforeEach(() => {
  OpenClawApp.prototype.connect = () => {
    // no-op
  };
  OpenClawApp.prototype.checkRbacSession = async function() {
    this.rbacUser = {
      userId: 1,
      username: "admin",
      roleName: "admin",
      permissions: [
        "menu:chat",
        "menu:sessions",
        "menu:overview",
        "menu:cron",
        "menu:config",
        "ops:inspect",
        "ops:ack",
        "ops:diagnose",
      ],
    };
    this.rbacChecked = true;
  };
  GatewayBrowserClient.prototype.start = function() {
    // no-op
  };
  localStorage.clear();
  document.body.innerHTML = "";
});

afterEach(() => {
  OpenClawApp.prototype.connect = originalConnect;
  OpenClawApp.prototype.checkRbacSession = originalCheckRbacSession;
  GatewayBrowserClient.prototype.start = originalStart;
  window.fetch = originalFetch;
  window.prompt = originalPrompt;
  localStorage.clear();
  document.body.innerHTML = "";
});

describe("Ops remediation E2E smoke tests", () => {
  it("verifies the login, cluster creation, domain overview, inspection and alert ack flow", async () => {
    // 1. Set up mock database arrays
    const mockClusters: any[] = [];
    const mockAlerts: any[] = [
      {
        id: "alert-group-1",
        source: "prometheus",
        domain: "hadoop",
        title: "HDFS Capacity Exceeded",
        severity: "critical",
        status: "active",
        originalCount: 5,
        reducedTo: 1,
        createdAtMs: Date.now() - 60000,
        updatedAtMs: Date.now() - 60000,
        rootCauseMarkdown: "Root cause analysis report from AI agent.",
        impactMarkdown: "Critical impact on YARN jobs.",
      }
    ];

    // 2. Intercept fetches
    const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
      if (url.includes("/api/auth/me")) {
        return new Response(JSON.stringify({
          userId: 1,
          username: "admin",
          roleName: "admin",
          permissions: [
            "menu:chat", "menu:sessions", "menu:overview", "menu:cron", "menu:config",
            "ops:inspect", "ops:ack", "ops:diagnose"
          ]
        }));
      }
      if (url.includes("/api/ops/clusters/sync-cmdb")) {
        const newCluster = {
          id: "cluster-cmdb-456",
          name: "CMDB Import Hadoop",
          domain: "hadoop",
          region: "guangzhou",
          nodeCount: 20,
          components: ["HDFS", "YARN", "Hive"],
          status: "healthy",
          owner: "System",
          createdAtMs: Date.now(),
          updatedAtMs: Date.now()
        };
        mockClusters.push(newCluster);
        return new Response(JSON.stringify({
          created: 1,
          updated: 0,
          skipped: 0,
          total: 1,
          source: "CMDB",
          strategy: "upsert",
          dryRun: false
        }));
      }
      if (url.includes("/api/ops/clusters")) {
        if (init?.method === "POST") {
          const body = JSON.parse(init.body as string);
          const newCluster = {
            id: "cluster-new-123",
            name: body.name,
            domain: body.domain,
            region: body.region ?? "",
            nodeCount: body.nodeCount ?? 0,
            components: body.components ?? [],
            status: body.status ?? "healthy",
            owner: body.owner ?? "Admin",
            createdAtMs: Date.now(),
            updatedAtMs: Date.now()
          };
          mockClusters.push(newCluster);
          return new Response(JSON.stringify(newCluster));
        }
        return new Response(JSON.stringify({ clusters: mockClusters }));
      }
      if (url.includes("/api/ops/dashboard/summary")) {
        return new Response(JSON.stringify({
          totalClusters: mockClusters.length,
          healthyClusters: mockClusters.filter(c => c.status === "healthy").length,
          warningClusters: 0,
          criticalClusters: 0,
          pendingAlerts: mockAlerts.filter(a => a.status === "active").length,
          vmConfigured: true,
          domains: [
            { domain: "hadoop", clusterCount: mockClusters.filter(c => c.domain === "hadoop").length, healthyCount: 1, warningCount: 0, criticalCount: 0 }
          ]
        }));
      }
      if (url.includes("/api/ops/inspection/im-status")) {
        return new Response(JSON.stringify({ imConfigured: true }));
      }
      if (url.includes("/api/ops/alerts/groups")) {
        if (init?.method === "PATCH") {
          const body = JSON.parse(init.body as string);
          const parts = url.split("/");
          const alertId = decodeURIComponent(parts[parts.length - 1]);
          const alert = mockAlerts.find(a => a.id === alertId);
          if (alert) {
            alert.status = body.status;
            if (body.ackNote) {
              alert.ackNote = body.ackNote;
            }
            return new Response(JSON.stringify(alert));
          }
        }
        return new Response(JSON.stringify({
          groups: mockAlerts,
          total: mockAlerts.length,
          originalTotal: mockAlerts.length,
          mergedTotal: mockAlerts.length,
          reductionRate: 0,
          pendingActive: mockAlerts.filter(a => a.status === "active").length
        }));
      }
      return new Response(JSON.stringify({}));
    });
    window.fetch = fetchMock;

    // 3. Mount App
    const app = mountApp("/asset-management");
    await app.updateComplete;

    // Set mock client for RPCs
    app.client = {
      start: () => {},
      stop: () => {},
      request: vi.fn(async (method: string, params: unknown) => {
        if (method === "cron.run") {
          return { entries: [{ status: "ok", sessionId: "session-new-123" }] };
        }
        if (method === "cron.runs") {
          return {
            entries: [
              {
                ts: Date.now(),
                jobId: "job-inspect-hadoop-deep",
                status: "ok",
                sessionId: "session-new-123",
                result: {
                  score: 95,
                  scoreStatus: "ok",
                  reportMarkdown: "# Hadoop Cluster Healthy\nEverything runs great."
                }
              }
            ]
          };
        }
        if (method === "cron.list") {
          return { jobs: [{ id: "job-inspect-hadoop-deep", name: "Hadoop Deep Inspect", enabled: true }] };
        }
        if (method === "cron.status") {
          return { running: false };
        }
        return { ok: true };
      })
    } as any;
    app.connected = true;

    // Trigger settings/app configuration load so cluster data is fetched
    await app.loadOpsClusters();
    await app.updateComplete;

    // 4. Verify empty state first
    const emptyState = app.querySelector(".ops-status--empty");
    expect(emptyState).not.toBeNull();
    expect(emptyState?.textContent).toContain("尚未纳管任何集群");

    // 5. Test CMDB Sync
    const syncButton = Array.from(app.querySelectorAll("button")).find(
      btn => btn.textContent?.includes("同步 CMDB")
    );
    expect(syncButton).toBeDefined();
    syncButton?.click();
    
    // Wait for CMDB sync to start and complete
    await new Promise(resolve => setTimeout(resolve, 50));
    while (app.opsCmdbSyncing) {
      await new Promise(resolve => setTimeout(resolve, 20));
    }
    await app.updateComplete;

    expect(app.opsCmdbSyncMessage).toContain("CMDB 同步完成");
    expect(app.opsClusters.length).toBe(1);
    expect(app.opsClusters[0].name).toBe("CMDB Import Hadoop");
    
    // Wait for another tick to allow layout rendering
    await new Promise(resolve => setTimeout(resolve, 50));
    await app.updateComplete;

    const firstCell = app.querySelector(".asset-table td");
    expect(firstCell?.textContent?.trim()).toBe("CMDB Import Hadoop");

    // 6. Test Manual Add Cluster
    const details = app.querySelector(".asset-form-panel") as HTMLDetailsElement;
    expect(details).not.toBeNull();
    details.open = true;
    await app.updateComplete;

    const form = app.querySelector(".asset-form") as HTMLFormElement;
    const nameInput = form.querySelector('input[name="name"]') as HTMLInputElement;
    nameInput.value = "New Test Cluster";
    form.dispatchEvent(new Event("submit"));
    
    // Wait for manual add/reload to complete
    await new Promise(resolve => setTimeout(resolve, 50));
    while (app.opsClustersLoading) {
      await new Promise(resolve => setTimeout(resolve, 20));
    }
    await app.updateComplete;

    const cells = Array.from(app.querySelectorAll(".asset-table td")).map(td => td.textContent?.trim());
    expect(cells).toContain("New Test Cluster");

    // 7. Navigate to Domain Overview tab
    app.tab = "hadoop";
    await app.loadOpsDomainClusters("hadoop");
    await app.updateComplete;

    const pageHeader = app.querySelector(".ops-sidebar__domain-title");
    expect(pageHeader?.textContent?.trim()).toBe("BCH生态");

    // 8. Test Manual Inspection
    const subtabs = Array.from(app.querySelectorAll(".ops-sidebar__menu-item")) as HTMLButtonElement[];
    // Click subtab index 3 -> "深度健康巡检"
    subtabs[3].click();
    await app.updateComplete;

    const inspectButton = Array.from(app.querySelectorAll("button")).find(
      btn => btn.textContent?.includes("一键手动巡检")
    ) as HTMLButtonElement;
    expect(inspectButton).toBeDefined();
    inspectButton.click();
    
    // Wait for the manual inspection run and its polling loop to finish
    await new Promise(resolve => setTimeout(resolve, 50));
    while (app.opsIsInspecting["hadoop"]) {
      await new Promise(resolve => setTimeout(resolve, 20));
    }
    await app.updateComplete;

    const markdownArea = app.querySelector(".ops-markdown");
    expect(markdownArea?.textContent).toContain("Everything runs great");

    // 9. Test Alert Ack
    subtabs[2].click(); // "告警降噪与影响评估"
    await app.loadOpsDomainAlerts("hadoop");
    await app.updateComplete;

    const alertTitle = app.querySelector(".alert-item__title");
    expect(alertTitle?.textContent?.trim()).toBe("HDFS Capacity Exceeded");

    // Stub window.prompt
    const promptSpy = vi.spyOn(window, "prompt").mockImplementation(() => "Verified and resolved manually");

    const ackButton = Array.from(app.querySelectorAll("button")).find(
      btn => btn.textContent?.includes("标记为已处理")
    ) as HTMLButtonElement;
    expect(ackButton).toBeDefined();
    ackButton.click();
    
    // Wait for alert updates/reloads to finish
    await new Promise(resolve => setTimeout(resolve, 50));
    while (app.opsAlertsLoading["hadoop"]) {
      await new Promise(resolve => setTimeout(resolve, 20));
    }
    await app.updateComplete;

    expect(promptSpy).toHaveBeenCalled();
    // Verify patch fetch was called with status resolved and custom ack note
    const patchCall = fetchMock.mock.calls.find(call => call[0].includes("/api/ops/alerts/groups/"));
    expect(patchCall).toBeDefined();
    const patchBody = JSON.parse(patchCall![1]!.body as string);
    expect(patchBody).toEqual({ status: "resolved", ackNote: "Verified and resolved manually" });
  });
});
