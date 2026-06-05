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
  activeAssetView?: "clusters" | "services" | "components" | "jobs" | "topology";
  user?: DomainFilterUser;
  onDomainChange?: (domain: string) => void;
  onAssetViewChange?: (view: "clusters" | "services" | "components" | "jobs" | "topology") => void;
};

const ASSET_VIEWS = [
  { id: "clusters", label: "集群资产", icon: "server" },
  { id: "services", label: "服务资产", icon: "network" },
  { id: "components", label: "组件资产", icon: "layout" },
  { id: "jobs", label: "作业资产", icon: "activity" },
  { id: "topology", label: "拓扑关系", icon: "folder" },
] as const;

export function renderAssetsView(props: AssetsViewProps) {
  const selectedDomain = props.selectedDomain || "all";
  const filteredClusters =
    selectedDomain === "all"
      ? props.clusters
      : props.clusters.filter((cluster) => cluster.domain === selectedDomain);
  const activeAssetView = props.activeAssetView ?? "clusters";
  const componentRows = filteredClusters.flatMap((cluster) =>
    (cluster.components || []).map((component) => ({ cluster, component })),
  );
  const jobRows = filteredClusters.flatMap((cluster) =>
    (cluster.components || [])
      .filter((component) => /flink|spark|yarn|hive|job|scheduler/i.test(component))
      .map((component, index) => ({
        id: `${cluster.id || cluster.name}-${component}-${index}`,
        name: `${component} 作业链路`,
        cluster,
        status: cluster.status,
      })),
  );

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
        <div class="column-header">${icons.network} 资产视图</div>
        <div style="display:flex; gap:8px; flex-wrap:wrap; margin-bottom: 12px;">
          ${ASSET_VIEWS.map(
            (item) => html`
              <button
                type="button"
                class="ops-btn ${activeAssetView === item.id ? "ops-btn--primary" : ""}"
                @click=${() => props.onAssetViewChange?.(item.id)}
              >
                ${icons[item.icon]} ${item.label}
              </button>
            `,
          )}
        </div>
        <p class="muted" style="margin-top:0;">
          当前展示 ${opsDomainLabel(selectedDomain)} 资产。后续服务依赖、组件资产、作业资产和拓扑视图在此扩展。
        </p>
      </div>

      ${activeAssetView === "clusters"
        ? renderAssetManagement({
            ...props,
            clusters: filteredClusters,
          })
        : activeAssetView === "services"
          ? renderServiceAssets(filteredClusters)
          : activeAssetView === "components"
            ? renderComponentAssets(componentRows)
            : activeAssetView === "jobs"
              ? renderJobAssets(jobRows)
              : renderTopologyAssets(filteredClusters, componentRows)}
    </div>
  `;
}

function renderServiceAssets(clusters: AssetsViewProps["clusters"]) {
  return html`
    <div class="ops-main-columns">
      <div class="ops-card list-column">
        <div class="column-header">服务目录</div>
        ${clusters.length === 0
          ? html`<div class="empty-placeholder">当前技术域暂无服务资产。</div>`
          : html`
              <div class="alert-list">
                ${clusters.map(
                  (cluster) => html`
                    <div class="alert-item">
                      <div class="alert-item__meta">
                        <span class="alert-badge alert-badge--info">${cluster.domain}</span>
                        <span class="alert-time">${cluster.owner || "未设置负责人"}</span>
                      </div>
                      <div class="alert-item__title">${cluster.name}</div>
                      <div class="alert-item__noise">
                        <span>区域: <strong>${cluster.region || "-"}</strong></span>
                        <span class="divider">|</span>
                        <span>组件: <strong>${cluster.components?.length ?? 0}</strong></span>
                      </div>
                    </div>
                  `,
                )}
              </div>
            `}
      </div>
      <div class="ops-card detail-column">
        <div class="column-header">服务责任关系</div>
        <p class="muted">服务资产由集群、组件、负责人和监控标签组成。后续可在这里接入服务依赖和 SLA。</p>
      </div>
    </div>
  `;
}

function renderComponentAssets(rows: Array<{ cluster: AssetsViewProps["clusters"][number]; component: string }>) {
  return html`
    <div class="ops-card">
      <div class="column-header">组件资产</div>
      ${rows.length === 0
        ? html`<div class="empty-placeholder">当前技术域暂无组件资产。</div>`
        : html`
            <div class="ops-summary-cards">
              ${rows.map(
                (row) => html`
                  <div class="ops-card stat-card">
                    <div class="stat-label">${row.cluster.name}</div>
                    <div class="stat-value info" style="font-size:20px;">${row.component}</div>
                    <div class="muted">${row.cluster.status || "unknown"} · ${row.cluster.owner || "未设置负责人"}</div>
                  </div>
                `,
              )}
            </div>
          `}
    </div>
  `;
}

function renderJobAssets(rows: Array<{ id: string; name: string; cluster: AssetsViewProps["clusters"][number]; status?: string }>) {
  return html`
    <div class="ops-card">
      <div class="column-header">作业资产</div>
      ${rows.length === 0
        ? html`<div class="empty-placeholder">未从组件资产中识别到作业类服务。后续可接入 Flink/Spark/YARN 作业 API。</div>`
        : html`
            <div class="alert-list">
              ${rows.map(
                (row) => html`
                  <div class="alert-item">
                    <div class="alert-item__meta">
                      <span class="alert-badge alert-badge--warning">${row.status || "unknown"}</span>
                      <span class="alert-time">${row.cluster.name}</span>
                    </div>
                    <div class="alert-item__title">${row.name}</div>
                    <div class="muted">作业资产来源于当前集群组件，后续接入真实作业运行状态。</div>
                  </div>
                `,
              )}
            </div>
          `}
    </div>
  `;
}

function renderTopologyAssets(
  clusters: AssetsViewProps["clusters"],
  rows: Array<{ cluster: AssetsViewProps["clusters"][number]; component: string }>,
) {
  return html`
    <div class="ops-card">
      <div class="column-header">拓扑关系</div>
      <div class="ops-summary-cards">
        <div class="ops-card stat-card">
          <div class="stat-label">集群节点</div>
          <div class="stat-value info">${clusters.length}</div>
          <div class="muted">服务拓扑一级节点</div>
        </div>
        <div class="ops-card stat-card">
          <div class="stat-label">组件节点</div>
          <div class="stat-value ok">${rows.length}</div>
          <div class="muted">由集群组件生成</div>
        </div>
      </div>
      <div class="detail-section">
        <div class="detail-section__header">${icons.network} 拓扑摘要</div>
        <div class="detail-section__content">
          ${clusters.length === 0
            ? "当前技术域暂无拓扑数据。"
            : clusters
                .map((cluster) => `${cluster.name} -> ${(cluster.components || []).join(", ") || "暂无组件"}`)
                .join("；")}
        </div>
      </div>
    </div>
  `;
}
