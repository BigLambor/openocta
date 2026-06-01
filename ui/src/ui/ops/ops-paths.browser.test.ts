import { describe, expect, it } from "vitest";
import { pathForTab, tabFromPath, type Tab } from "../navigation.ts";

/** Routes referenced in docs/e2e-ops-smoke.md (P4-4 automation smoke). */
const OPS_SMOKE_TABS: Tab[] = [
  "overview",
  "assetManagement",
  "hadoop",
  "fi",
  "gbase",
  "governance",
  "dataapps",
  "message",
  "scheduledTasks",
];

describe("ops smoke paths", () => {
  for (const tab of OPS_SMOKE_TABS) {
    it(`round-trips path for ${tab}`, () => {
      const path = pathForTab(tab);
      expect(path.length).toBeGreaterThan(0);
      expect(tabFromPath(path)).toBe(tab);
    });
  }
});
