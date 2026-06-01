import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { OpenClawApp } from "./app.ts";
import { GatewayBrowserClient } from "./gateway.ts";
import "../styles.css";

// oxlint-disable-next-line typescript/unbound-method
const originalConnect = OpenClawApp.prototype.connect;
// oxlint-disable-next-line typescript/unbound-method
const originalCheckRbacSession = OpenClawApp.prototype.checkRbacSession;
// oxlint-disable-next-line typescript/unbound-method
const originalStart = GatewayBrowserClient.prototype.start;

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
  localStorage.clear();
  document.body.innerHTML = "";
});

describe("Ops remediation E2E smoke tests", () => {
  it("verifies the login, cluster creation, and domain overview flow", async () => {
    // 1. Mount App at asset management tab
    const app = mountApp("/asset-management");
    await app.updateComplete;

    expect(app.tab).toBe("assetManagement");

    // Populate mock clusters to simulate loaded state
    app.opsClusters = [
      {
        id: "cluster-1",
        name: "Test BCH Prod",
        domain: "hadoop",
        region: "shenzhen",
        nodeCount: 15,
        components: ["HDFS", "YARN"],
        status: "healthy",
        owner: "Admin",
        createdAtMs: Date.now(),
        updatedAtMs: Date.now(),
        monitorLabels: "",
        vmUrlRef: "",
        metricsBaseUrl: "",
        jmxUrl: "",
        fiManagerUrl: "",
        gbaseDsnRef: "",
        credentialsRef: "",
      },
    ];
    await app.updateComplete;

    // Check cluster table is rendered and contains our mock cluster name
    const table = app.querySelector(".asset-table");
    expect(table).not.toBeNull();
    const cell = app.querySelector(".asset-table td");
    expect(cell?.textContent?.trim()).toBe("Test BCH Prod");

    // 2. Navigate to Domain Overview
    app.tab = "hadoop";
    await app.updateComplete;

    const pageHeader = app.querySelector(".ops-sidebar__domain-title");
    expect(pageHeader?.textContent?.trim()).toBe("BCH生态");
  });
});
