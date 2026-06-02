export type OpsDomainKey = "hadoop" | "fi" | "gbase" | "governance" | "dataapps";

export type OpsCapabilityKey =
  | "observability-alert"
  | "health-inspection"
  | "diagnosis-incident"
  | "governance-optimization"
  | "capacity-performance-cost"
  | "change-config-compliance";

export const OPS_DOMAIN_KEYS: OpsDomainKey[] = [
  "hadoop",
  "fi",
  "gbase",
  "governance",
  "dataapps",
];

export const OPS_CAPABILITY_KEYS: OpsCapabilityKey[] = [
  "observability-alert",
  "health-inspection",
  "diagnosis-incident",
  "governance-optimization",
  "capacity-performance-cost",
  "change-config-compliance",
];

export const OPS_DOMAIN_LABELS: Record<OpsDomainKey, string> = {
  hadoop: "BCH生态",
  fi: "FI商业生态",
  gbase: "GBase数据库",
  governance: "开发治理平台",
  dataapps: "数据App运维",
};

export const OPS_CAPABILITY_LABELS: Record<OpsCapabilityKey, string> = {
  "observability-alert": "可观测与告警",
  "health-inspection": "健康度与巡检",
  "diagnosis-incident": "故障诊断与应急",
  "governance-optimization": "治理与优化",
  "capacity-performance-cost": "容量性能与成本",
  "change-config-compliance": "变更配置与合规",
};

export function normalizeOpsDomainKey(raw: string | null | undefined): OpsDomainKey | string {
  const key = (raw ?? "").trim().toLowerCase();
  switch (key) {
    case "":
    case "bch":
    case "bigdata":
    case "hadoop-ecosystem":
    case "hadoop":
      return "hadoop";
    case "fi":
    case "fusioninsight":
      return "fi";
    case "gbase":
      return "gbase";
    case "governance":
    case "dev-governance":
    case "development-governance":
      return "governance";
    case "data-apps":
    case "data_apps":
    case "dataapp":
    case "dataapps":
      return "dataapps";
    default:
      return key;
  }
}

export function normalizeOpsCapabilityKey(raw: string | null | undefined): OpsCapabilityKey | string {
  const key = (raw ?? "").trim().toLowerCase();
  switch (key) {
    case "":
    case "observability":
    case "alerts":
    case "alert":
    case "observability-alert":
      return "observability-alert";
    case "inspection":
    case "health":
    case "health-inspection":
      return "health-inspection";
    case "diagnosis":
    case "incident":
    case "diagnosis-incident":
      return "diagnosis-incident";
    case "governance":
    case "optimization":
    case "governance-optimization":
      return "governance-optimization";
    case "capacity":
    case "performance":
    case "cost":
    case "capacity-performance-cost":
      return "capacity-performance-cost";
    case "change":
    case "config":
    case "compliance":
    case "change-config-compliance":
      return "change-config-compliance";
    default:
      return key;
  }
}

export function opsDomainLabel(raw: string | null | undefined): string {
  const key = normalizeOpsDomainKey(raw);
  return OPS_DOMAIN_LABELS[key as OpsDomainKey] ?? key;
}

export function opsCapabilityLabel(raw: string | null | undefined): string {
  const key = normalizeOpsCapabilityKey(raw);
  return OPS_CAPABILITY_LABELS[key as OpsCapabilityKey] ?? key;
}
