import { describe, expect, it } from "vitest";
import { resolveJobRunIdForCronEntry } from "./ops-job-runs.ts";

describe("ops-job-runs", () => {
  it("prefers explicit runId on cron entry", () => {
    const id = resolveJobRunIdForCronEntry({ runId: "run-1", ts: 1000 }, []);
    expect(id).toBe("run-1");
  });

  it("matches nearest job run by timestamp", () => {
    const id = resolveJobRunIdForCronEntry(
      { ts: 5000, runAtMs: 5000 },
      [
        { id: "old", jobId: "job-1", triggerType: "cron", status: "succeeded", startedAt: 1000, createdAt: 1000, updatedAt: 1000 },
        { id: "hit", jobId: "job-1", triggerType: "cron", status: "succeeded", startedAt: 5100, createdAt: 5100, updatedAt: 5100 },
      ],
    );
    expect(id).toBe("hit");
  });
});
