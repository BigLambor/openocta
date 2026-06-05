/** PromQL label alignment for cluster assets → VictoriaMetrics. */

export type MonitorLabelPair = { key: string; value: string };

export type DomainMonitorGuide = {
  domain: string;
  domainLabel: string;
  labelKeys: string[];
  example: string;
  baseQueryHint: string;
  verifyQuery: string;
  checkSteps: string[];
};

export const DOMAIN_MONITOR_GUIDES: Record<string, DomainMonitorGuide> = {
  hadoop: {
    domain: "hadoop",
    domainLabel: "BCH 生态 (hadoop)",
    labelKeys: ["job", "cluster", "instance"],
    example: 'job="hadoop-prod",cluster="bj-bch-prod"',
    baseQueryHint:
      'avg(up{job=~".*(hadoop|yarn|hdfs).*"} or up{instance=~".*hadoop.*"})',
    verifyQuery: 'count(up{job=~".*(hadoop|yarn|hdfs).*"})',
    checkSteps: [
      "在 VM 执行 label_values(up, job) 或 label_values(up, cluster)，确认目标集群标签值",
      "登记 monitorLabels 使用与 VM 一致的 job/cluster，而非资产 id (cluster-uuid)",
      "保存后驾驶舱域健康分应能按该集群聚合；Agent vm_query 会注入相同标签",
    ],
  },
  fi: {
    domain: "fi",
    domainLabel: "FI 商业生态 (fi)",
    labelKeys: ["job", "cluster", "fusion_cluster", "instance"],
    example: 'job="fi-prod",cluster="huhe-fi-prod"',
    baseQueryHint: 'avg(up{job=~".*(fusion|fi).*"} or up{instance=~".*fi.*"})',
    verifyQuery: 'count(up{job=~".*(fusion|fi).*"})',
    checkSteps: [
      "确认 FI Manager / node_exporter 上报的 job 或 cluster 标签",
      "monitorLabels 至少包含 job 或 cluster 之一，且值与 VM 时序完全一致",
      "多 FI 集群同域时，必须用 cluster/env 等标签区分，避免域级粗查询串数据",
    ],
  },
  gbase: {
    domain: "gbase",
    domainLabel: "GBase 数据库 (gbase)",
    labelKeys: ["job", "cluster", "instance"],
    example: 'job="gbase-prod",instance="gbase-primary"',
    baseQueryHint: 'avg(up{job=~".*gbase.*"} or up{instance=~".*gbase.*"})',
    verifyQuery: 'count(up{job=~".*gbase.*"})',
    checkSteps: [
      "核对 GBase exporter 的 job/instance 标签",
      "主备或多套库用 cluster 或 instance 区分",
      "gbaseDsnRef 用于 SQL 巡检，monitorLabels 仅负责指标关联",
    ],
  },
  governance: {
    domain: "governance",
    domainLabel: "开发治理平台 (governance)",
    labelKeys: ["job", "cluster", "service"],
    example: 'job="gov-platform",service="metadata-registry"',
    baseQueryHint:
      'avg(up{job=~".*(governance|metadata).*"} or up{instance=~".*governance.*"})',
    verifyQuery: 'count(up{job=~".*(governance|metadata).*"})',
    checkSteps: [
      "治理平台组件常以 service/job 区分，确认 VM 中实际 label 名",
      "monitorLabels 与平台 Prometheus 抓取配置保持一致",
    ],
  },
  dataapps: {
    domain: "dataapps",
    domainLabel: "数据 App 运维 (dataapps)",
    labelKeys: ["job", "cluster", "app"],
    example: 'job="dataapp-scheduler",app="core-scheduler"',
    baseQueryHint:
      'avg(up{job=~".*(dataapp|scheduler|pipeline).*"} or up{instance=~".*dataapp.*"})',
    verifyQuery: 'count(up{job=~".*(dataapp|scheduler|pipeline).*"})',
    checkSteps: [
      "调度类 App 常用 job/app 标签，先在 VM 查 label_values(up, app)",
      "monitorLabels 用于驾驶舱健康分与 Agent PromQL 注入，与资产 id 无关",
    ],
  },
};

const LABEL_KEY_PATTERN = /^[a-zA-Z_][a-zA-Z0-9_]*$/;

function looksLikeJsonLabels(raw: string): boolean {
  const s = raw.trim();
  return s.startsWith("{") && (s.includes('":') || s.includes('": '));
}

function splitLabelPairs(raw: string): string[] {
  const parts: string[] = [];
  let current = "";
  let inQuotes = false;
  for (let i = 0; i < raw.length; i++) {
    const ch = raw[i];
    if (ch === '"') {
      inQuotes = !inQuotes;
      current += ch;
      continue;
    }
    if (ch === "," && !inQuotes) {
      parts.push(current);
      current = "";
      continue;
    }
    current += ch;
  }
  if (current) parts.push(current);
  return parts;
}

function unquoteLabelValue(value: string): string | null {
  const v = value.trim();
  if (v.length >= 2 && v.startsWith('"') && v.endsWith('"')) {
    return v.slice(1, -1).replace(/\\"/g, '"');
  }
  if (/[{},]/.test(v)) return null;
  return v;
}

export function parseMonitorLabels(raw: string): MonitorLabelPair[] | null {
  const trimmed = raw.trim();
  if (!trimmed) return null;
  if (looksLikeJsonLabels(trimmed)) {
    throw new Error(
      'monitorLabels 请使用 PromQL 标签格式（如 job="prod",cluster="a"），不要使用 JSON',
    );
  }

  let body = trimmed;
  if (body.startsWith("{")) body = body.slice(1);
  if (body.endsWith("}")) body = body.slice(0, -1);
  body = body.trim();
  if (!body) throw new Error("monitorLabels 不能为空片段");

  const pairs: MonitorLabelPair[] = [];
  for (const part of splitLabelPairs(body)) {
    const segment = part.trim();
    if (!segment) continue;
    const eq = segment.indexOf("=");
    if (eq <= 0) throw new Error(`monitorLabels 片段无效: ${segment}（应为 key=value）`);
    const key = segment.slice(0, eq).trim();
    const valueRaw = segment.slice(eq + 1).trim();
    if (!LABEL_KEY_PATTERN.test(key)) {
      throw new Error(`monitorLabels 标签名无效: ${key}`);
    }
    const value = unquoteLabelValue(valueRaw);
    if (!value) {
      throw new Error(`monitorLabels 值 ${valueRaw} 含特殊字符时请使用双引号`);
    }
    pairs.push({ key, value });
  }
  if (pairs.length === 0) throw new Error("monitorLabels 至少包含一组 key=value");
  return pairs;
}

export function formatMonitorLabels(pairs: MonitorLabelPair[]): string {
  return pairs.map((p) => `${p.key}="${p.value}"`).join(",");
}

export function normalizeMonitorLabels(raw: string): string {
  const pairs = parseMonitorLabels(raw);
  if (!pairs) return "";
  return formatMonitorLabels(pairs);
}

export type MonitorLabelsValidation = {
  ok: boolean;
  error?: string;
  normalized?: string;
  guide?: DomainMonitorGuide;
};

export function validateMonitorLabelsForCluster(
  domain: string,
  status: string,
  raw: string,
): MonitorLabelsValidation {
  const guide = DOMAIN_MONITOR_GUIDES[domain];
  const trimmed = raw.trim();
  const normalizedStatus = status.trim().toLowerCase();

  if (normalizedStatus === "inactive") {
    if (!trimmed) return { ok: true, normalized: "", guide };
    try {
      return { ok: true, normalized: normalizeMonitorLabels(trimmed), guide };
    } catch (err) {
      return { ok: false, error: err instanceof Error ? err.message : String(err), guide };
    }
  }

  if (!trimmed) {
    return {
      ok: false,
      error:
        "非下线集群必须配置 monitorLabels，否则无法关联 VictoriaMetrics 指标（资产 id 不会自动写入 PromQL）",
      guide,
    };
  }

  let pairs: MonitorLabelPair[];
  try {
    pairs = parseMonitorLabels(trimmed) ?? [];
  } catch (err) {
    return { ok: false, error: err instanceof Error ? err.message : String(err), guide };
  }

  if (guide) {
    const keys = new Set(pairs.map((p) => p.key));
    const hasDomainKey = guide.labelKeys.some((k) => keys.has(k));
    if (!hasDomainKey) {
      return {
        ok: false,
        error: `${guide.domainLabel} 的 monitorLabels 须包含以下标签之一: ${guide.labelKeys.join(", ")}（示例: ${guide.example}）`,
        guide,
      };
    }
  }

  return { ok: true, normalized: formatMonitorLabels(pairs), guide };
}

export function monitorLinkStatus(
  domain: string,
  status: string,
  monitorLabels?: string,
): "linked" | "missing" | "invalid" | "na" {
  if (status.trim().toLowerCase() === "inactive") return "na";
  const result = validateMonitorLabelsForCluster(domain, status, monitorLabels ?? "");
  if (result.ok && result.normalized) return "linked";
  if ((monitorLabels ?? "").trim() === "") return "missing";
  return "invalid";
}
