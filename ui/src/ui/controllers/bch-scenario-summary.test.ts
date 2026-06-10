import { afterEach, describe, expect, it, vi } from "vitest";
import { fetchBchDomainScenarioSummary } from "./bch-scenario-summary.ts";

const host = {
  gatewayHttpUrl: "http://127.0.0.1:1455",
  settings: { token: "" },
};

afterEach(() => {
  vi.restoreAllMocks();
});

describe("fetchBchDomainScenarioSummary", () => {
  it("builds BCH scenario cards from live API payloads", async () => {
    vi.spyOn(Date, "now").mockReturnValue(123456);
    vi.stubGlobal("fetch", vi.fn(async (url: string) => {
      if (url.includes("/api/ops/bch/clusters/health")) {
        return new Response(JSON.stringify([{ id: "c1", name: "prod-a", score: 92 }]));
      }
      if (url.includes("/api/ops/bch/flink/jobs")) {
        return new Response(JSON.stringify([
          {
            id: "f1",
            name: "flink-a",
            status: "RUNNING",
            score: 91,
            metrics: { restarts: 1, fullGcCount: 0, isBP: false },
          },
          {
            id: "f2",
            name: "flink-b",
            status: "RUNNING",
            score: 83,
            metrics: { restarts: 0, fullGcCount: 2, isBP: true },
          },
        ]));
      }
      if (url.includes("/api/ops/bch/spark/jobs")) {
        return new Response(JSON.stringify([
          {
            id: "s1",
            name: "spark-a",
            status: "FAILED",
            metrics: { failedTasks: 2, cpuSkewRatio: 3, memorySkewRatio: 1, executorMemoryOverheadMB: 2048 },
            tuningAdvice: "reduce executor memory",
          },
        ]));
      }
      if (url.includes("/api/ops/bch/hdfs/fsimage")) {
        return new Response(JSON.stringify({
          namespace: "default",
          totalRecords: "100",
          totalFiles: "80",
          totalDirs: "20",
          totalSize: "8 TB",
          avgFileSize: "20 MB",
          maxDepth: "18",
          sizeData: [{ size: "<1KB", count: 10, percent: 18 }],
          zeroByteFiles: 1000,
          trashFiles: 200000,
        }));
      }
      return new Response(JSON.stringify({}), { status: 404 });
    }) as any);

    const summary = await fetchBchDomainScenarioSummary(host);

    expect(summary.updatedAtMs).toBe(123456);
    expect(summary.source).toBe("bch-api");
    expect(summary.errors).toEqual([]);
    expect(summary.scenarios).toHaveLength(3);
    expect(summary.scenarios[0]).toMatchObject({
      id: "flink-health",
      score: 87,
      primaryMetric: "运行中作业: 2/2",
    });
    expect(summary.scenarios[1]).toMatchObject({
      id: "spark-tuning",
      score: 83,
      primaryMetric: "调优候选: 1/1",
    });
    expect(summary.scenarios[2]).toMatchObject({
      id: "hdfs-storage",
      score: 67,
      primaryMetric: "小文件 18%，最大目录深度 18",
    });
    expect(summary.aggregate.jobHealthScore).toBe(85);
    expect(summary.aggregate.serviceHealthScore).toBe(92);
  });

  it("keeps partial summaries when one BCH API fails", async () => {
    vi.stubGlobal("fetch", vi.fn(async (url: string) => {
      if (url.includes("/api/ops/bch/flink/jobs")) {
        return new Response(JSON.stringify([{ id: "f1", status: "RUNNING", score: 100, metrics: {} }]));
      }
      if (url.includes("/api/ops/bch/spark/jobs")) {
        return new Response(JSON.stringify([]));
      }
      if (url.includes("/api/ops/bch/hdfs/fsimage")) {
        return new Response(JSON.stringify({ namespace: "default", maxDepth: "1", sizeData: [] }));
      }
      return new Response(JSON.stringify({ error: "down" }), { status: 500 });
    }) as any);

    const summary = await fetchBchDomainScenarioSummary(host);

    expect(summary.errors.length).toBeGreaterThan(0);
    expect(summary.scenarios).toHaveLength(3);
    expect(summary.scenarios[0].score).toBe(100);
    expect(summary.aggregate.serviceHealthScore).toBeNull();
  });
});
