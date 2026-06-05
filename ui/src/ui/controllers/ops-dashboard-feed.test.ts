import { describe, expect, it } from "vitest";
import type { OpsAlertGroupRecord } from "./ops-alerts.ts";
import {
  domainFromInspectJobId,
  mergeAlertHighlights,
  mergeRecentInspectionRuns,
  pickTopAlertsPerDomain,
} from "./ops-dashboard-feed.ts";

function alert(
  partial: Partial<OpsAlertGroupRecord> & Pick<OpsAlertGroupRecord, "id" | "title">,
): OpsAlertGroupRecord {
  return {
    source: "test",
    severity: "warning",
    status: "active",
    originalCount: 1,
    reducedTo: 1,
    createdAtMs: 1,
    updatedAtMs: 1,
    ...partial,
  };
}

describe("ops-dashboard-feed", () => {
  it("maps inspect job id to domain key", () => {
    expect(domainFromInspectJobId("job-inspect-hadoop")).toBe("hadoop");
    expect(domainFromInspectJobId("other")).toBe("other");
  });

  it("picks pending alerts by severity then recency", () => {
    const groups = pickTopAlertsPerDomain(
      [
        alert({ id: "a", title: "info", severity: "info", status: "active", createdAtMs: 300 }),
        alert({ id: "b", title: "critical", severity: "critical", status: "active", createdAtMs: 100 }),
        alert({ id: "c", title: "warning", severity: "warning", status: "analyzing", createdAtMs: 200 }),
        alert({ id: "d", title: "resolved", severity: "critical", status: "resolved", createdAtMs: 400 }),
      ],
      2,
    );
    expect(groups.map((g) => g.id)).toEqual(["b", "c"]);
  });

  it("merges alert highlights across domains with cap", () => {
    const { highlights, pendingByDomain } = mergeAlertHighlights(
      [
        {
          domain: "hadoop",
          pendingActive: 3,
          groups: [
            alert({ id: "h1", title: "HDFS", severity: "critical", status: "active", createdAtMs: 100 }),
            alert({ id: "h2", title: "YARN", severity: "warning", status: "active", createdAtMs: 90 }),
          ],
        },
        {
          domain: "fi",
          pendingActive: 1,
          groups: [
            alert({ id: "f1", title: "FI", severity: "warning", status: "active", createdAtMs: 80 }),
          ],
        },
      ],
      2,
    );
    expect(pendingByDomain).toEqual({ hadoop: 3, fi: 1 });
    expect(highlights.map((h) => h.id)).toEqual(["h1", "h2"]);
  });

  it("merges recent inspection runs globally by time", () => {
    const runs = mergeRecentInspectionRuns(
      [
        {
          ts: 1000,
          jobId: "job-inspect-hadoop",
          runAtMs: 1000,
          result: { score: 88 },
          summary: "hadoop ok",
        },
        {
          ts: 3000,
          jobId: "job-inspect-fi",
          runAtMs: 3000,
          result: { score: 95 },
          summary: "fi ok",
        },
        {
          ts: 2000,
          jobId: "job-inspect-gbase",
          runAtMs: 2000,
          error: "timeout",
          status: "error",
        },
      ],
      2,
    );
    expect(runs.map((r) => r.domain)).toEqual(["fi", "gbase"]);
    expect(runs[1]?.status).toBe("error");
  });
});
