import { render } from "lit";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { OpsClusterRecord } from "../controllers/ops-clusters.ts";
import { renderAssetsView, type AssetViewId } from "./assets-view.ts";

function cluster(overrides: Partial<OpsClusterRecord> = {}): OpsClusterRecord {
  return {
    id: "cluster-1",
    name: "测试-治理-平台A",
    domain: "governance",
    region: "北京",
    nodeCount: 6,
    components: ["Atlas", "DataHub"],
    owner: "ops-test",
    status: "healthy",
    createdAtMs: 1,
    updatedAtMs: 1,
    ...overrides,
  };
}

describe("renderAssetsView", () => {
  let container: HTMLElement;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
  });

  it("does not expose job assets in sidebar", () => {
    render(
      renderAssetsView({
        clusters: [cluster()],
        activeAssetView: "clusters",
        user: { roleName: "admin", permissions: [] },
      }),
      container,
    );

    expect(container.textContent).not.toContain("作业资产");
    expect(container.textContent).toContain("服务资产（规划中）");
    expect(container.textContent).not.toContain("搜索过滤");
    expect(container.textContent).not.toContain("资产运行状态");
  });

  it("renders all cluster rows instead of limiting the table to eight", () => {
    const clusters = Array.from({ length: 12 }, (_, index) =>
      cluster({
        id: `cluster-${index + 1}`,
        name: `测试集群-${index + 1}`,
      }),
    );

    render(
      renderAssetsView({
        clusters,
        activeAssetView: "clusters",
        canManage: true,
        user: { roleName: "admin", permissions: [] },
      }),
      container,
    );

    expect(container.querySelectorAll(".asset-table tbody tr")).toHaveLength(12);
    expect(container.textContent).toContain("测试集群-12");
  });

  it("weakens service assets view with planning preview", () => {
    render(
      renderAssetsView({
        clusters: [cluster()],
        activeAssetView: "services",
        user: { roleName: "admin", permissions: [] },
      }),
      container,
    );

    expect(container.textContent).toContain("服务资产（预览）");
    expect(container.textContent).toContain("前往集群资产");
    expect(container.textContent).not.toContain("AI 分析");
  });

  it("component assets link back to cluster edit without AI analysis", () => {
    const onAssetViewChange = vi.fn();
    const onOpenEditDrawer = vi.fn();

    render(
      renderAssetsView({
        clusters: [cluster()],
        activeAssetView: "components",
        canManage: true,
        onAssetViewChange,
        onOpenEditDrawer,
        user: { roleName: "admin", permissions: [] },
      }),
      container,
    );

    expect(container.textContent).toContain("Atlas");
    expect(container.textContent).toContain("修改");
    expect(container.textContent).toContain("纳管中 (健康)");
    expect(container.textContent).toContain("开发治理平台");
    expect(container.textContent).not.toContain("AI 分析");

    const editButton = Array.from(container.querySelectorAll("button")).find((btn) =>
      btn.textContent?.trim() === "修改",
    );
    editButton?.click();
    expect(onAssetViewChange).toHaveBeenCalledWith("clusters");
    expect(onOpenEditDrawer).toHaveBeenCalledWith("cluster-1");
  });

  it("normalizes legacy jobs view to clusters", () => {
    render(
      renderAssetsView({
        clusters: [cluster()],
        activeAssetView: "jobs" as AssetViewId,
        canManage: true,
        onOpenAddDrawer: vi.fn(),
        user: { roleName: "admin", permissions: [] },
      }),
      container,
    );

    expect(container.textContent).toContain("新增纳管集群");
    expect(container.textContent).not.toContain("作业资产");
  });
});
