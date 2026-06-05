import { html } from "lit";

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

export function renderDomainFilter(props: {
  selectedDomain: string;
  user: DomainFilterUser;
  includeAll?: boolean;
  onChange?: (domain: OpsDomainKey) => void;
}) {
  const selected = normalizeOpsDomain(props.selectedDomain);
  const options = OPS_DOMAIN_OPTIONS.filter((item) => {
    if (item.key === "all" && props.includeAll === false) {
      return false;
    }
    return canAccessOpsDomain(props.user, item.key);
  });
  return html`
    <div class="ops-domain-pills" role="group" aria-label="技术域过滤器">
      ${options.map(
        (item) => html`
          <button
            type="button"
            class="ops-domain-pill ${selected === item.key ? "ops-domain-pill--active" : ""}"
            @click=${() => props.onChange?.(item.key)}
          >
            ${item.label}
          </button>
        `,
      )}
    </div>
  `;
}
