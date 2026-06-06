import { render } from "lit";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { OpsClusterRecord } from "../controllers/ops-clusters.ts";
import { renderAssetManagement } from "./asset-management.ts";

const confirmMock = vi.hoisted(() => vi.fn(() => true));

function cluster(overrides: Partial<OpsClusterRecord> = {}): OpsClusterRecord {
  return {
    id: "cluster-1",
    name: "测试-治理-平台A",
    domain: "governance",
    region: "北京",
    nodeCount: 6,
    components: ["Atlas", "DataHub"],
    owner: "ops-test",
    status: "warning",
    createdAtMs: 1,
    updatedAtMs: 1,
    monitorLabels: 'cluster="gov-a",env="test"',
    ...overrides,
  };
}

describe("renderAssetManagement", () => {
  let container: HTMLElement;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
    confirmMock.mockReset();
    confirmMock.mockReturnValue(true);
    vi.stubGlobal("confirm", confirmMock);
  });

  it("shows add button in table toolbar and opens drawer", async () => {
    const onOpenAddDrawer = vi.fn();
    render(
      renderAssetManagement({
        embedded: true,
        clusters: [cluster()],
        canManage: true,
        drawerOpen: false,
        onOpenAddDrawer,
      }),
      container,
    );

    const addButton = Array.from(container.querySelectorAll("button")).find((btn) =>
      btn.textContent?.includes("新增纳管集群"),
    );
    expect(addButton).toBeDefined();
    addButton?.click();
    expect(onOpenAddDrawer).toHaveBeenCalledOnce();
  });

  it("renders edit/delete actions instead of AI analysis", () => {
    render(
      renderAssetManagement({
        embedded: true,
        clusters: [cluster()],
        canManage: true,
        onOpenEditDrawer: vi.fn(),
        onDeleteCluster: vi.fn(),
      }),
      container,
    );

    expect(container.textContent).toContain("修改");
    expect(container.textContent).toContain("删除");
    expect(container.textContent).not.toContain("AI 分析");
  });

  it("calls onOpenEditDrawer when edit is clicked", () => {
    const onOpenEditDrawer = vi.fn();
    render(
      renderAssetManagement({
        embedded: true,
        clusters: [cluster()],
        canManage: true,
        onOpenEditDrawer,
      }),
      container,
    );

    const editButton = Array.from(container.querySelectorAll("button")).find((btn) =>
      btn.textContent?.trim() === "修改",
    );
    editButton?.click();
    expect(onOpenEditDrawer).toHaveBeenCalledWith("cluster-1");
  });

  it("calls onDeleteCluster after confirm", async () => {
    const onDeleteCluster = vi.fn(async () => undefined);
    render(
      renderAssetManagement({
        embedded: true,
        clusters: [cluster()],
        canManage: true,
        onDeleteCluster,
      }),
      container,
    );

    const deleteButton = Array.from(container.querySelectorAll("button")).find((btn) =>
      btn.textContent?.trim() === "删除",
    );
    deleteButton?.click();
    expect(confirmMock).toHaveBeenCalledOnce();
    expect(onDeleteCluster).toHaveBeenCalledWith("cluster-1");
  });

  it("renders drawer form for add and edit modes", () => {
    render(
      renderAssetManagement({
        embedded: true,
        clusters: [cluster({ name: "编辑目标" })],
        canManage: true,
        drawerOpen: true,
        drawerMode: "edit",
        editingClusterId: "cluster-1",
        onCloseDrawer: vi.fn(),
        onUpdateCluster: vi.fn(),
      }),
      container,
    );

    expect(container.textContent).toContain("修改纳管集群");
    expect(container.textContent).toContain("编辑目标");
    expect((container.querySelector('input[name="name"]') as HTMLInputElement)?.value).toBe("编辑目标");
    expect(container.textContent).toContain("保存修改");
  });
});
