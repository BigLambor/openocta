import { afterEach, describe, expect, it, vi } from "vitest";
import {
  createOpsCluster,
  deleteOpsCluster,
  fetchOpsClusters,
  updateOpsCluster,
  type OpsClusterRecord,
} from "./ops-clusters.ts";

const host = {
  gatewayHttpUrl: "http://127.0.0.1:1455",
  rbacToken: "token",
  settings: { token: "" },
};

const sampleCluster: OpsClusterRecord = {
  id: "cluster-1",
  name: "测试集群",
  domain: "governance",
  region: "北京",
  nodeCount: 6,
  components: ["Atlas", "DataHub"],
  owner: "ops-test",
  status: "healthy",
  createdAtMs: 1,
  updatedAtMs: 1,
  monitorLabels: 'cluster="test",env="prod"',
};

afterEach(() => {
  vi.restoreAllMocks();
});

describe("ops-clusters API client", () => {
  it("fetchOpsClusters loads cluster list", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(JSON.stringify({ clusters: [sampleCluster] }))) as typeof fetch,
    );

    const clusters = await fetchOpsClusters(host);
    expect(clusters).toHaveLength(1);
    expect(clusters[0]?.name).toBe("测试集群");
  });

  it("createOpsCluster posts cluster payload", async () => {
    const fetchMock = vi.fn(async (_url: string, init?: RequestInit) => {
      expect(init?.method).toBe("POST");
      const body = JSON.parse(String(init?.body));
      expect(body).toMatchObject({
        name: "新集群",
        domain: "hadoop",
        components: ["HDFS", "YARN"],
      });
      return new Response(JSON.stringify({ ...sampleCluster, ...body, id: "cluster-new" }));
    });
    vi.stubGlobal("fetch", fetchMock as typeof fetch);

    const created = await createOpsCluster(host, {
      name: "新集群",
      domain: "hadoop",
      region: "上海",
      nodeCount: 12,
      components: ["HDFS", "YARN"],
      owner: "alice",
      status: "healthy",
      monitorLabels: 'cluster="new",env="prod"',
    });

    expect(created.id).toBe("cluster-new");
    expect(fetchMock).toHaveBeenCalledOnce();
  });

  it("updateOpsCluster patches cluster by id", async () => {
    const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
      expect(url).toContain("/api/ops/clusters/cluster-1");
      expect(init?.method).toBe("PATCH");
      const body = JSON.parse(String(init?.body));
      expect(body).toMatchObject({
        components: ["Atlas", "DataHub", "Kafka"],
        owner: "ops-bob",
      });
      return new Response(JSON.stringify({
        ...sampleCluster,
        components: body.components,
        owner: body.owner,
      }));
    });
    vi.stubGlobal("fetch", fetchMock as typeof fetch);

    const updated = await updateOpsCluster(host, "cluster-1", {
      components: ["Atlas", "DataHub", "Kafka"],
      owner: "ops-bob",
    });

    expect(updated.owner).toBe("ops-bob");
    expect(updated.components).toEqual(["Atlas", "DataHub", "Kafka"]);
  });

  it("deleteOpsCluster removes cluster by id", async () => {
    const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
      expect(url).toContain("/api/ops/clusters/cluster-1");
      expect(init?.method).toBe("DELETE");
      return new Response(JSON.stringify({ ok: true }));
    });
    vi.stubGlobal("fetch", fetchMock as typeof fetch);

    await deleteOpsCluster(host, "cluster-1");
    expect(fetchMock).toHaveBeenCalledOnce();
  });

  it("surfaces API errors", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(JSON.stringify({ error: "无权限" }), { status: 403 })) as typeof fetch,
    );

    await expect(deleteOpsCluster(host, "cluster-1")).rejects.toThrow("无权限");
  });
});
