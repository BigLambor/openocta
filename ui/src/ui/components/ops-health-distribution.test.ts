import { describe, expect, it } from "vitest";
import type { OpsClusterRecord } from "../controllers/ops-clusters.ts";
import {
  computeRollupHealthScore,
  distributionFromClusters,
  pickTopRiskClusters,
} from "./ops-health-distribution.ts";

function cluster(name: string, status: OpsClusterRecord["status"]): OpsClusterRecord {
  return {
    id: `id-${name}`,
    name,
    domain: "hadoop",
    nodeCount: 1,
    components: [],
    status,
    createdAtMs: 0,
    updatedAtMs: 0,
  };
}

describe("ops-health-distribution", () => {
  it("aggregates cluster statuses into distribution counts", () => {
    expect(
      distributionFromClusters([
        cluster("a", "healthy"),
        cluster("b", "healthy"),
        cluster("c", "warning"),
        cluster("d", "critical"),
        cluster("e", "unknown"),
      ]),
    ).toEqual({ healthy: 2, warning: 1, critical: 1, unknown: 1, inactive: 0 });
  });

  it("computes rollup health score from cluster status mix", () => {
    expect(computeRollupHealthScore({ healthy: 2, warning: 0, critical: 0 })).toBe(100);
    expect(computeRollupHealthScore({ healthy: 1, warning: 1, critical: 0 })).toBe(86);
    expect(computeRollupHealthScore({ healthy: 0, warning: 0, critical: 2 })).toBe(35);
  });

  it("picks top risk clusters by severity then name", () => {
    const picked = pickTopRiskClusters(
      [
        cluster("healthy-a", "healthy"),
        cluster("warn-b", "warning"),
        cluster("crit-c", "critical"),
        cluster("warn-a", "warning"),
      ],
      3,
    );
    expect(picked.map((c) => c.name)).toEqual(["crit-c", "warn-a", "warn-b"]);
  });
});
