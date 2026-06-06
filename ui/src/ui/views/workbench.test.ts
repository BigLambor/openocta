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
});
