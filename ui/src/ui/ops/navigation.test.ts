import { describe, expect, it } from "vitest";
import {
  DEFAULT_TECH_OPS_CAPABILITY,
  normalizeTechOpsCapabilityTab,
} from "./navigation.ts";

describe("normalizeTechOpsCapabilityTab", () => {
  it("accepts current capability tab ids", () => {
    expect(normalizeTechOpsCapabilityTab("overview")).toBe("overview");
    expect(normalizeTechOpsCapabilityTab("observability")).toBe("observability");
  });

  it("maps legacy sub-tab ids", () => {
    expect(normalizeTechOpsCapabilityTab("alerts")).toBe("observability");
    expect(normalizeTechOpsCapabilityTab("agent")).toBe("diagnosis");
    expect(normalizeTechOpsCapabilityTab("inspections")).toBe("inspection");
  });

  it("returns null for unknown values", () => {
    expect(normalizeTechOpsCapabilityTab("unknown")).toBeNull();
  });
});

describe("DEFAULT_TECH_OPS_CAPABILITY", () => {
  it("defaults to overview", () => {
    expect(DEFAULT_TECH_OPS_CAPABILITY).toBe("overview");
  });
});
