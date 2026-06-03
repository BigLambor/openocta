import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { OpenClawApp } from "./app.ts";
import "../styles.css";

const originalConnect = OpenClawApp.prototype.connect;
const originalCheckRbacSession = OpenClawApp.prototype.checkRbacSession;

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
  localStorage.clear();
  document.body.innerHTML = "";
});

afterEach(() => {
  OpenClawApp.prototype.connect = originalConnect;
  OpenClawApp.prototype.checkRbacSession = originalCheckRbacSession;
  localStorage.clear();
  document.body.innerHTML = "";
});

describe("BCH ecosystem ops scenarios tests", () => {
  it("renders all BCH sub-pages properly under hadoop domain", async () => {
    // Intercept fetches
    const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
      if (url.includes("/api/auth/me")) {
        return new Response(JSON.stringify({
          userId: 1,
          username: "admin",
          roleName: "admin",
          permissions: ["menu:chat", "menu:sessions", "menu:overview", "menu:cron", "menu:config"]
        }));
      }
      if (url.includes("/api/ops/bch/clusters/health")) {
        return new Response(JSON.stringify([
          {
            id: "cluster-prod-a",
            name: "北京 BCH 生产集群 A (prod-a)",
            region: "北京",
            status: "healthy",
            score: 98,
            nodeCount: 120,
            activeAlerts: 0,
            cpuUsedPercent: 62.5,
            memUsedPercent: 78.2,
            dfsUsedPercent: 54.1,
            metrics: { activeContainers: 1840 }
          }
        ]));
      }
      if (url.includes("/api/ops/bch/flink/jobs")) {
        return new Response(JSON.stringify([
          {
            id: "job_tx_core",
            name: "交易核心链路 (Trade_Analysis)",
            owner: "cui.chao",
            cluster: "prod-a",
            status: "RUNNING",
            score: 100,
            sScore: 100,
            pScore: 100,
            eScore: 100,
            metrics: { lagTrend: 0, maxLag: 10, avgLag: 8, isBP: false, cpuMax: 60, cpuAvg: 55, heapMax: 60, fullGcCount: 0, restarts: 0 },
            penalties: [],
            diagnosis: "正常",
            rootCause: "S0",
            rootCauseText: "运行健康",
            actions: ["无需干预"],
            cotSteps: {
              step1: { text: "正常", state: "active" },
              step2: { text: "正常", state: "active" },
              step3: { text: "正常", state: "active" }
            }
          }
        ]));
      }
      if (url.includes("/api/ops/bch/hdfs/fsimage")) {
        return new Response(JSON.stringify({
          namespace: "NS1",
          totalRecords: "93030336",
          totalFiles: "77.1 M",
          totalDirs: "15.9 M",
          totalSize: "7.08 PB",
          avgFileSize: "98.53 MB",
          maxDepth: "18",
          processingTime: "8136.93",
          processingSpeed: "11433",
          depthData: [{ depth: "3 级", count: 5070698, percent: 31.8 }],
          sizeData: [{ size: "<1KB", count: 14054646, percent: 18.2 }],
          userData: [{ user: "production", files: 49778480, percent: 53.5, size: "5.56 PB" }],
          modifyData: [{"period": "<1周", count: 31974807, percent: 34.4}],
          accessData: [{"period": "<1周", count: 30224464, percent: 39.1}],
          fileTypeData: [{ ext: ".orc", count: 29733113, percent: 38.6 }],
          pathData: [{ path: "/user/bdoc", count: 40240488, percent: 43.3 }],
          zeroByteFiles: 1053,
          trashFiles: 495519
        }));
      }
      if (url.includes("/api/ops/bch/employees")) {
        return new Response(JSON.stringify([
          {
            id: "emp_bch_inspect",
            name: "BCH 深度巡检数字员工",
            status: "idle",
            statusDesc: "就绪",
            description: "负责巡检",
            skills: ["巡检技能"],
            tools: ["query_vm_metrics"],
            recentTasks: []
          }
        ]));
      }
      return new Response(JSON.stringify({}));
    });
    window.fetch = fetchMock;

    const app = mountApp("/hadoop");
    app.client = {
      start: () => {},
      stop: () => {},
      request: vi.fn(async () => ({ ok: true }))
    } as any;
    app.connected = true;

    // Explicitly navigate and load domain clusters
    app.tab = "hadoop";
    await app.loadOpsDomainClusters("hadoop");
    await app.updateComplete;

    // Verify subtabs exist in domain view
    const subtabs = Array.from(app.querySelectorAll(".ops-sidebar__menu-item")) as HTMLButtonElement[];
    expect(subtabs.length).toBeGreaterThan(0);

    // 1. Overview Tab
    subtabs[0].click(); // "概览"
    app.opsActiveSubTabs = { ...app.opsActiveSubTabs, hadoop: "overview" };
    await app.updateComplete;
    const overviewEl = app.querySelector("bch-cluster-overview");
    expect(overviewEl).not.toBeNull();

    // 2. Job Governance Tab
    subtabs[4].click(); // "作业治理"
    app.opsActiveSubTabs = { ...app.opsActiveSubTabs, hadoop: "jobGovernance" };
    await app.updateComplete;
    const govEl = app.querySelector("bch-job-governance");
    expect(govEl).not.toBeNull();

    // 3. Capacity Tab
    subtabs[7].click(); // "容量性能与成本" (capacity)
    app.opsActiveSubTabs = { ...app.opsActiveSubTabs, hadoop: "capacity" };
    await app.updateComplete;
    const capEl = app.querySelector("bch-fsimage-dashboard");
    expect(capEl).not.toBeNull();

    // 4. Employees Tab
    subtabs[9].click(); // "数字员工" (employees)
    app.opsActiveSubTabs = { ...app.opsActiveSubTabs, hadoop: "employees" };
    await app.updateComplete;
    const empEl = app.querySelector("bch-employee-workstation");
    expect(empEl).not.toBeNull();
  });
});
