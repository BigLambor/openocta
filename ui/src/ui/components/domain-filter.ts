export type OpsDomainKey = "all" | "hadoop" | "fi" | "gbase" | "governance" | "dataapps";

export type DomainFilterUser = {
  roleName?: string;
  permissions?: string[];
} | null;

export const OPS_DOMAIN_OPTIONS: Array<{ key: OpsDomainKey; label: string; shortLabel: string }> = [
  { key: "all", label: "全部技术域", shortLabel: "全部" },
  { key: "hadoop", label: "BCH 生态", shortLabel: "BCH" },
  { key: "fi", label: "FI 商业生态", shortLabel: "FI" },
  { key: "gbase", label: "GBase 数据库", shortLabel: "GBase" },
  { key: "governance", label: "开发治理平台", shortLabel: "开发治理" },
  { key: "dataapps", label: "数据 App 运维", shortLabel: "数据 App" },
];

const DOMAIN_KEYS = OPS_DOMAIN_OPTIONS.filter((item) => item.key !== "all").map((item) => item.key);

export function normalizeOpsDomain(raw?: string | null): OpsDomainKey {
  const value = (raw || "").trim().toLowerCase();
  if (value === "bch" || value === "bigdata" || value === "hadoop-ecosystem") {
    return "hadoop";
  }
  if (value === "data-apps" || value === "data_apps" || value === "dataapp") {
    return "dataapps";
  }
  return OPS_DOMAIN_OPTIONS.some((item) => item.key === value) ? (value as OpsDomainKey) : "all";
}

export function canAccessOpsDomain(user: DomainFilterUser, domain: OpsDomainKey): boolean {
  if (domain === "all") {
    return DOMAIN_KEYS.some((key) => canAccessOpsDomain(user, key));
  }
  if (!user) {
    return false;
  }
  if (user.roleName === "admin") {
    return true;
  }
  return Boolean(user.permissions?.includes(`menu:${domain}`));
}

export function firstAccessibleOpsDomain(user: DomainFilterUser): OpsDomainKey {
  if (canAccessOpsDomain(user, "all")) {
    return "all";
  }
  return DOMAIN_KEYS.find((key) => canAccessOpsDomain(user, key)) ?? "hadoop";
}

export function opsDomainLabel(domain: string, short = false): string {
  const normalized = normalizeOpsDomain(domain);
  const option = OPS_DOMAIN_OPTIONS.find((item) => item.key === normalized);
  return short ? option?.shortLabel ?? domain : option?.label ?? domain;
}

export function effectiveOpsDomain(domain: string): Exclude<OpsDomainKey, "all"> {
  const normalized = normalizeOpsDomain(domain);
  return normalized === "all" ? "hadoop" : normalized;
}

export type OpsAssistant = {
  employeeId: string;
  name: string;
  persona: string;
};

/**
 * Single source of truth for mapping a technical domain to its digital-employee
 * template. Used by both the workbench AI side panel (display) and the
 * confirm/reject flow (execution-record write) so the persona shown always
 * matches the employee id recorded.
 */
export function opsAssistantForDomain(domain: string): OpsAssistant {
  switch (effectiveOpsDomain(domain)) {
    case "gbase":
      return {
        employeeId: "emp_gbase_diagnose",
        name: "GBase 诊断数字员工",
        persona: "专家人设：GBase 慢 SQL、锁等待、容量与性能根因分析。",
      };
    case "fi":
      return {
        employeeId: "emp_fi_inspect",
        name: "FI 巡检数字员工",
        persona: "专家人设：FusionInsight 组件健康巡检、告警降噪与风险研判。",
      };
    case "governance":
      return {
        employeeId: "emp_governance_remediate",
        name: "开发治理数字员工",
        persona: "专家人设：元数据治理、血缘影响面与配置合规整改。",
      };
    case "dataapps":
      return {
        employeeId: "emp_dataapps_ops",
        name: "数据 App 护航数字员工",
        persona: "专家人设：数据应用链路、调度稳定性与 SLA 护航。",
      };
    case "hadoop":
    default:
      return {
        // Seeded BCH on-call employee (pkg/init/employee.go); keep in sync with
        // the backend automation.domainEmployeeIDs mapping.
        employeeId: "emp_bch_duty",
        name: "BCH 值班数字员工",
        persona: "专家人设：BCH 告警降噪、根因候选、影响面判断、处置建议。",
      };
  }
}
