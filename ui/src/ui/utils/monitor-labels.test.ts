import { describe, expect, it } from "vitest";
import {
  normalizeMonitorLabels,
  parseMonitorLabels,
  validateMonitorLabelsForCluster,
} from "./monitor-labels.ts";

describe("monitor-labels", () => {
  it("parses promql label pairs", () => {
    expect(parseMonitorLabels('job="hadoop-prod",cluster="bj"')).toEqual([
      { key: "job", value: "hadoop-prod" },
      { key: "cluster", value: "bj" },
    ]);
  });

  it("rejects json format", () => {
    expect(() => parseMonitorLabels('{"job":"hadoop-prod"}')).toThrow(/JSON/);
  });

  it("normalizes unquoted values", () => {
    expect(normalizeMonitorLabels("job=hadoop-prod")).toBe('job="hadoop-prod"');
  });

  it("requires labels for active clusters", () => {
    expect(validateMonitorLabelsForCluster("hadoop", "healthy", "").ok).toBe(false);
    expect(validateMonitorLabelsForCluster("hadoop", "inactive", "").ok).toBe(true);
  });

  it("requires domain-specific label keys", () => {
    expect(validateMonitorLabelsForCluster("fi", "healthy", 'env="prod"').ok).toBe(false);
    expect(validateMonitorLabelsForCluster("gbase", "healthy", 'job="gbase-prod"').ok).toBe(
      true,
    );
  });
});
