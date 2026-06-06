import { monitorLinkStatus } from "../utils/monitor-labels.ts";

export const ASSET_DOMAIN_OPTIONS = [
  { value: "hadoop", label: "BCH生态" },
  { value: "fi", label: "FI商业生态" },
  { value: "gbase", label: "GBase数据库" },
  { value: "governance", label: "开发治理平台" },
  { value: "dataapps", label: "数据App运维" },
] as const;

export const ASSET_DOMAIN_LABEL: Record<string, string> = Object.fromEntries(
  ASSET_DOMAIN_OPTIONS.map((o) => [o.value, o.label]),
);

export function assetMonitorLinkLabel(domain: string, status: string, monitorLabels?: string) {
  const link = monitorLinkStatus(domain, status, monitorLabels);
  switch (link) {
    case "linked":
      return "已关联 VM";
    case "missing":
      return "未配置标签";
    case "na":
      return "已下线";
    default:
      return "标签待修正";
  }
}

export function assetStatusLabel(status: string) {
  switch (status) {
    case "healthy":
      return "纳管中 (健康)";
    case "warning":
      return "亚健康";
    case "critical":
      return "异常";
    case "inactive":
      return "已下线";
    default:
      return "未知";
  }
}
