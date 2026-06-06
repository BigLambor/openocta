import { render } from "lit";
import { describe, expect, it } from "vitest";
import { renderWorkbench, type WorkbenchProps } from "./workbench.ts";

function buildProps(overrides: Partial<WorkbenchProps> = {}): WorkbenchProps {
  return {
    domainName: "全部技术域",
    selectedDomain: "all",
    activeView: "diagnosis",
    user: {
      roleName: "viewer",
      permissions: ["menu:hadoop"],
    },
    alertGroups: [],
    ...overrides,
  };
}

describe("workbench view scenario directory filtering", () => {
  it("filters scenarios by domain permissions for non-admin user in all domains view", async () => {
    const container = document.createElement("div");
    const props = buildProps({
      user: {
        roleName: "viewer",
        permissions: ["menu:hadoop"], // Only has access to BCH (hadoop)
      },
    });

    render(renderWorkbench(props), container);
    await Promise.resolve();

    const titles = Array.from(container.querySelectorAll("h3")).map((el) => el.textContent?.trim());
    // In diagnosis center, should only see BCH Flink health scenario
    expect(titles).toContain("Flink 作业健康度");
    expect(titles).not.toContain("GBase 慢 SQL 诊断");
    expect(titles).not.toContain("FI 组件异常诊断");
  });

  it("shows all scenarios in all domains view for admin user", async () => {
    const container = document.createElement("div");
    const props = buildProps({
      user: {
        roleName: "admin",
        permissions: [],
      },
    });

    render(renderWorkbench(props), container);
    await Promise.resolve();

    const titles = Array.from(container.querySelectorAll("h3")).map((el) => el.textContent?.trim());
    expect(titles).toContain("Flink 作业健康度");
    expect(titles).toContain("GBase 慢 SQL 诊断");
    expect(titles).toContain("GBase 锁等待诊断");
    expect(titles).toContain("FI 组件异常诊断");
    expect(titles).toContain("调度失败诊断");
  });

  it("renders event AI operations inside the shared drawer task switch", async () => {
    const container = document.createElement("div");
    const props = buildProps({
      activeView: "events",
      selectedAlertGroupId: "alert-1",
      aiPanelOpen: true,
      aiPanelMode: "root-cause",
      alertGroups: [
        {
          id: "alert-1",
          title: "HDFS NameNode 内存占用过高",
          severity: "warning",
          timestamp: "2026-06-05 09:28:09",
          originalCount: 3,
          reducedTo: 1,
          rootCause: "NameNode heap 使用率持续高于阈值。",
        },
      ],
    });

    render(renderWorkbench(props), container);
    await Promise.resolve();

    expect(container.querySelector(".ops-ai-drawer")).not.toBeNull();
    const taskLabels = Array.from(container.querySelectorAll(".ops-ai-task-switch__item")).map((el) =>
      el.textContent?.trim(),
    );
    expect(taskLabels).toEqual(["根因分析", "相似聚合", "处置建议"]);
    expect(container.textContent).toContain("告警 AI 分析");
  });

  it("uses ops-nav-icon styling for scenario cards and sidebar items", async () => {
    const container = document.createElement("div");
    render(
      renderWorkbench(
        buildProps({
          activeView: "capacity",
          user: { roleName: "admin", permissions: [] },
        }),
      ),
      container,
    );
    await Promise.resolve();

    const scenarioIcons = container.querySelectorAll(".minimal-scenario-card .ops-nav-icon");
    expect(scenarioIcons.length).toBeGreaterThan(0);
    const sidebarIcons = container.querySelectorAll(".ops-sidebar-nav-item .ops-nav-icon");
    expect(sidebarIcons.length).toBeGreaterThan(0);
  });

  it("uses a single AI entry in scenario detail and opens the same drawer", async () => {
    const container = document.createElement("div");
    const props = buildProps({
      selectedDomain: "hadoop",
      domainName: "BCH 生态",
      activeView: "diagnosis",
      selectedScenarioId: "bch-flink-health",
      aiPanelOpen: true,
      aiPanelMode: "action",
      selectedObjectScope: "all",
      selectedTimeRange: "24h",
    });

    render(renderWorkbench(props), container);
    await Promise.resolve();

    expect(container.querySelector(".ops-ai-drawer")).not.toBeNull();
    expect(container.textContent).toContain("AI 辅助分析");
    expect(container.textContent).not.toContain("记录闭环");
    expect(container.textContent).not.toContain("执行记录");
    expect(container.querySelector(".workbench-scenario-toolbar")).toBeNull();
  });
});
