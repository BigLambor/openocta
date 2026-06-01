import { describe, expect, it } from "vitest";
import {
  canAckAlerts,
  canDiagnose,
  canRunInspection,
  hasOpsPermission,
} from "./rbac.ts";

describe("ops rbac helpers", () => {
  it("allows all ops actions when user is null (dev / pre-auth)", () => {
    expect(hasOpsPermission(null, "ops:inspect")).toBe(false);
    expect(canRunInspection(null)).toBe(false);
    expect(canDiagnose(null)).toBe(false);
    expect(canAckAlerts(null)).toBe(false);
  });

  it("grants admin every permission", () => {
    const admin = { roleName: "admin", permissions: [] };
    expect(hasOpsPermission(admin, "ops:ack")).toBe(true);
    expect(canAckAlerts(admin)).toBe(true);
  });

  it("checks explicit permission codes for non-admin", () => {
    const viewer = { roleName: "viewer", permissions: ["ops:inspect"] };
    expect(canRunInspection(viewer)).toBe(true);
    expect(canAckAlerts(viewer)).toBe(false);
    expect(canDiagnose(viewer)).toBe(false);
  });
});
