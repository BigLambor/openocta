import { html } from "lit";
import { icons } from "../icons.ts";
import {
  OPS_DOMAIN_OPTIONS,
  opsDomainLabel,
  renderDomainFilter,
  type DomainFilterUser,
} from "../components/domain-filter.ts";
import { renderAssetManagement, type AssetManagementProps } from "./asset-management.ts";

export type AssetsViewProps = AssetManagementProps & {
  selectedDomain?: string;
  user?: DomainFilterUser;
  onDomainChange?: (domain: string) => void;
};

export function renderAssetsView(props: AssetsViewProps) {
  const selectedDomain = props.selectedDomain || "all";
  const filteredClusters =
    selectedDomain === "all"
      ? props.clusters
      : props.clusters.filter((cluster) => cluster.domain === selectedDomain);

  const domainCounts = OPS_DOMAIN_OPTIONS.map(({ key, label }) => ({
    key,
    label,
    count: key === "all" ? props.clusters.length : props.clusters.filter((c) => c.domain === key).length,
  }));

  return html`
    <div class="ops-domain-page">
      <div class="ops-domain-hero">
        <div>
          <div class="ops-domain-kicker">服务与资产 · ${opsDomainLabel(selectedDomain)}</div>
          <h1>资产目录</h1>
          <p>技术域作为全局上下文过滤器使用，资产、拓扑和责任关系在这里统一管理。</p>
        </div>
        ${renderDomainFilter({
          selectedDomain,
          user: props.user ?? null,
          includeAll: true,
          onChange: (domain) => props.onDomainChange?.(domain),
        })}
      </div>

      <div class="ops-summary-cards">
        ${domainCounts.map(
          (item) => html`
            <div class="ops-card stat-card">
              <div class="stat-label">${item.label}</div>
              <div class="stat-value ${item.key === selectedDomain ? "ok" : "info"}">${item.count}</div>
              <div class="muted">${item.key === "all" ? "全部资产" : "技术域资产"}</div>
            </div>
          `,
        )}
      </div>

      <div class="ops-card" style="margin-bottom: 16px;">
        <div class="column-header">${icons.network} 资产管理</div>
        <p class="muted" style="margin-top:0;">
          当前展示 ${opsDomainLabel(selectedDomain)} 资产。后续服务依赖、组件资产、作业资产和拓扑视图在此扩展。
        </p>
      </div>

      ${renderAssetManagement({
        ...props,
        clusters: filteredClusters,
      })}
    </div>
  `;
}
