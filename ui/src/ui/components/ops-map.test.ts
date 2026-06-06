import { describe, expect, it } from "vitest";
import { OpsMap } from "./ops-map.ts";
import type { OpsClusterRecord } from "../controllers/ops-clusters.ts";

describe("ops-map component", () => {
  it("computes empty cities data when clusters are empty", () => {
    const component = new OpsMap();
    component.clusters = [];
    const data = (component as any).getCitiesData();
    
    expect(data.beijing).toHaveLength(0);
    expect(data.shanghai).toHaveLength(0);
    expect(data.shenzhen).toHaveLength(0);
    expect(data.chengdu).toHaveLength(0);
    expect(data.wuhan).toHaveLength(0);
    expect(data.xian).toHaveLength(0);
  });

  it("populates cities data with real clusters when provided", () => {
    const component = new OpsMap();
    const mockCluster: OpsClusterRecord = {
      id: "cluster-beijing",
      name: "北京数据节点",
      domain: "hadoop",
      region: "北京",
      nodeCount: 10,
      components: [],
      status: "critical",
      createdAtMs: 0,
      updatedAtMs: 0,
    };
    component.clusters = [mockCluster];
    const data = (component as any).getCitiesData();
    
    // Beijing should have the cluster
    expect(data.beijing).toHaveLength(1);
    expect(data.beijing[0]).toEqual({ domain: "hadoop", score: 50, status: "critical", alerts: 1 });
    
    // Other cities should be empty
    expect(data.shanghai).toHaveLength(0);
    expect(data.shenzhen).toHaveLength(0);
  });

  it("populates cities data with health snapshots when provided", () => {
    const component = new OpsMap();
    const mockCluster: OpsClusterRecord = {
      id: "cluster-beijing",
      name: "北京数据节点",
      domain: "hadoop",
      region: "北京",
      nodeCount: 10,
      components: [],
      status: "healthy",
      createdAtMs: 0,
      updatedAtMs: 0,
    };
    component.clusters = [mockCluster];
    component.snapshots = [{
      clusterId: "cluster-beijing",
      scoreStatus: "critical",
      score: 42,
    }];
    const data = (component as any).getCitiesData();
    
    // Beijing should reflect the snapshot status and score rather than static status
    expect(data.beijing).toHaveLength(1);
    expect(data.beijing[0]).toEqual({ domain: "hadoop", score: 42, status: "critical", alerts: 1 });
  });

  it("renders SVG elements in DOM dynamically based on clusters", async () => {
    const div = document.createElement("div");
    document.body.appendChild(div);
    const component = document.createElement("ops-map") as OpsMap;
    
    const mockClusters: OpsClusterRecord[] = [
      { id: "c1", name: "北京集群", domain: "hadoop", region: "北京", nodeCount: 1, components: [], status: "healthy", createdAtMs: 0, updatedAtMs: 0 },
      { id: "c2", name: "哈池", domain: "fi", region: "哈尔滨", nodeCount: 1, components: [], status: "warning", createdAtMs: 0, updatedAtMs: 0 },
      { id: "c3", name: "杭州集群", domain: "gbase", region: "杭州", nodeCount: 1, components: [], status: "critical", createdAtMs: 0, updatedAtMs: 0 }
    ];
    component.clusters = mockClusters;
    
    div.appendChild(component);
    
    await new Promise(resolve => setTimeout(resolve, 100)); // wait for Lit render
    
    const svg = component.querySelector("svg");
    expect(svg).not.toBeNull();
    
    const nodes = component.querySelectorAll(".ops-map__node-group");
    expect(nodes.length).toBe(3);
    
    const labels = Array.from(component.querySelectorAll(".ops-map__node-label")).map(el => el.textContent);
    expect(labels).toContain("北京");
    expect(labels).toContain("哈尔滨");
    expect(labels).toContain("杭州");
    
    expect(component.querySelectorAll(".ops-map__node-group > .ops-map__node-halo")).toHaveLength(3);
    expect(component.querySelectorAll(".ops-map__node-group > .ops-map__node-center")).toHaveLength(3);
    
    document.body.removeChild(div);
  });
});
