/** Ops button-level RBAC helpers (P4-2). */

export type RbacUserLike = {
  roleName?: string;
  permissions?: string[];
} | null;

export function hasOpsPermission(user: RbacUserLike, code: string): boolean {
  if (!user) {
    return false;
  }
  if (user.roleName === "admin") {
    return true;
  }
  return (user.permissions ?? []).includes(code);
}

export function canRunInspection(user: RbacUserLike): boolean {
  return hasOpsPermission(user, "ops:inspect");
}

export function canDiagnose(user: RbacUserLike): boolean {
  return hasOpsPermission(user, "ops:diagnose");
}

export function canAckAlerts(user: RbacUserLike): boolean {
  return hasOpsPermission(user, "ops:ack");
}
