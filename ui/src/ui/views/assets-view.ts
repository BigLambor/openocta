import { html } from "lit";
import { icons } from "../icons.ts";
import {
  OPS_DOMAIN_OPTIONS,
  opsDomainLabel,
  normalizeOpsDomain,
  canAccessOpsDomain,
  type OpsDomainKey,
  type DomainFilterUser,
} from "../components/domain-filter.ts";
import {
  renderOpsShellHeader,
  renderOpsShellStatGrid,
  type OpsViewNavItem,
} from "../components/ops-shell.ts";
import { renderAssetsContextSidebar, type AssetsViewSidebarItem } from "../components/assets-context-sidebar.ts";
import { renderAssetManagement, type AssetManagementProps } from "./asset-management.ts";

export type AssetsViewProps = AssetManagementProps & {
  selectedDomain?: string;
  activeAssetView?: "clusters" | "services" | "components" | "jobs" | "topology";
  user?: DomainFilterUser;
  onDomainChange?: (domain: string) => void;
  onAssetViewChange?: (view: "clusters" | "services" | "components" | "jobs" | "topology") => void;
  searchQuery?: string;
  statusFilter?: string;
  onSearchChange?: (query: string) => void;
  onStatusFilterChange?: (status: string) => void;
};

const ASSET_VIEWS: OpsViewNavItem<NonNullable<AssetsViewProps["activeAssetView"]>>[] = [
  { id: "clusters", label: "集群资产", icon: "server" },
  { id: "services", label: "服务资产", icon: "network" },
  { id: "components", label: "组件资产", icon: "layout" },
  { id: "jobs", label: "作业资产", icon: "activity" },
  { id: "topology", label: "拓扑关系", icon: "folder" },
];

function clusterHealthCounts(clusters: AssetsViewProps["clusters"]) {
  let healthy = 0;
  let warning = 0;
  let critical = 0;
  for (const c of clusters) {
    if (c.status === "healthy") healthy++;
    else if (c.status === "warning") warning++;
    else if (c.status === "critical") critical++;
  }
  return { healthy, warning, critical };
}

export function renderAssetsView(props: AssetsViewProps) {
  const selectedDomain = props.selectedDomain || "all";
  const normalized = normalizeOpsDomain(selectedDomain);
  const searchQuery = props.searchQuery || "";
  const statusFilter = props.statusFilter || "all";

  // 1. Filter by technical domain
  const domainClusters =
    normalized === "all"
      ? props.clusters
      : props.clusters.filter((cluster) => cluster.domain === normalized);

  // 2. Filter by status
  let statusClusters = domainClusters;
  if (statusFilter !== "all") {
    statusClusters = statusClusters.filter((c) => c.status === statusFilter);
  }

  // 3. Filter by search query
  let filteredClusters = statusClusters;
  if (searchQuery.trim()) {
    const q = searchQuery.toLowerCase().trim();
    filteredClusters = filteredClusters.filter((c) => {
      const nameMatch = c.name?.toLowerCase().includes(q) || false;
      const ownerMatch = c.owner?.toLowerCase().includes(q) || false;
      const componentMatch = c.components?.some((comp: string) => comp.toLowerCase().includes(q)) || false;
      return nameMatch || ownerMatch || componentMatch;
    });
  }

  const activeAssetView = props.activeAssetView ?? "clusters";
  
  // Aggregate stats from the domain clusters
  const { healthy, warning, critical } = clusterHealthCounts(domainClusters);
  
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

  // Domain tree options with counts
  const domainCountsWithAll = [
    { key: "all" as OpsDomainKey, label: "全部技术域", count: props.clusters.length },
    ...OPS_DOMAIN_OPTIONS.filter((o) => o.key !== "all" && canAccessOpsDomain(props.user ?? null, o.key)).map(({ key, label }) => ({
      key,
      label,
      count: props.clusters.filter((c) => c.domain === key).length,
    }))
  ];

  const sidebarViews: AssetsViewSidebarItem<NonNullable<AssetsViewProps["activeAssetView"]>>[] = ASSET_VIEWS.map((item) => {
    let viewCount = 0;
    if (item.id === "clusters") viewCount = filteredClusters.length;
    else if (item.id === "services") viewCount = filteredClusters.length;
    else if (item.id === "components") viewCount = componentRows.length;
    else if (item.id === "jobs") viewCount = jobRows.length;
    else if (item.id === "topology") viewCount = filteredClusters.length;
    return {
      id: item.id,
      label: item.label,
      icon: item.icon,
      count: viewCount,
    };
  });

  return html`
    <div class="ops-assets-layout" style="display: flex; height: 100%; width: 100%;">
      ${renderAssetsContextSidebar({
        domains: domainCountsWithAll,
        selectedDomain: normalized,
        views: sidebarViews,
        activeView: activeAssetView,
        searchQuery,
        statusFilter,
        onDomainChange: props.onDomainChange,
        onViewChange: props.onAssetViewChange,
        onSearchChange: props.onSearchChange,
        onStatusFilterChange: props.onStatusFilterChange,
      })}

      <!-- Right Content area -->
      <div style="flex: 1; min-width: 0; overflow-y: auto;">
        <main class="ops-dashboard ops-shell" style="height: 100%; box-sizing: border-box; display: flex; flex-direction: column;">
          ${renderOpsShellHeader({
            kicker: `服务与资产 · ${opsDomainLabel(selectedDomain)}`,
            title: "资产目录",
            description: "资产、拓扑和责任关系在此统一管理。",
            toolbar: html`
              <button
                type="button"
                class="ops-btn"
                ?disabled=${props.loading || props.cmdbSyncing}
                @click=${() => props.onRefresh?.()}
              >
                ${icons.refreshCw} 刷新
              </button>
              ${props.onSyncCmdb
                ? html`
                    <button
                      type="button"
                      class="ops-btn ops-btn--primary"
                      ?disabled=${props.loading || props.cmdbSyncing}
                      title=${props.cmdbSyncHint ?? "从 CMDB 同步集群"}
                      @click=${() => props.onSyncCmdb?.()}
                    >
                      ${props.cmdbSyncing ? icons.loader : icons.refreshCw}
                      ${props.cmdbSyncing ? "同步中…" : "同步 CMDB"}
                    </button>
                  `
                : ""}
            `,
          })}

          ${renderOpsShellStatGrid([
            {
              label: "纳管集群",
              value: filteredClusters.length,
              hint: opsDomainLabel(selectedDomain),
              tone: "blue",
              icon: "server",
            },
            {
              label: "健康",
              value: healthy,
              hint: "状态正常",
              tone: "ok",
              icon: "checkCircle",
            },
            {
              label: "亚健康",
              value: warning,
              hint: "需关注",
              tone: "warn",
              icon: "alertTriangle",
            },
            {
              label: "异常",
              value: critical,
              hint: "优先处理",
              tone: "danger",
              icon: "bell",
            },
          ])}

          ${activeAssetView === "clusters"
            ? renderAssetManagement({
                ...props,
                embedded: true,
                clusters: filteredClusters,
              })
            : activeAssetView === "services"
              ? renderServiceAssets(filteredClusters, props)
              : activeAssetView === "components"
                ? renderComponentAssets(componentRows, props)
                : activeAssetView === "jobs"
                  ? renderJobAssets(jobRows, props)
                  : renderTopologyAssets(filteredClusters, componentRows)}
        </main>
      </div>
    </div>
  `;
}

function renderServiceAssets(clusters: AssetsViewProps["clusters"], props: AssetsViewProps) {
  return html`
    <div class="ops-shell-columns">
      <div class="ops-shell-panel list-column">
        <div class="ops-shell-panel__head">${icons.network} 服务目录</div>
        <div class="alert-list" style="padding:10px;">
          ${clusters.length === 0
            ? html`<div class="empty-placeholder">当前技术域暂无服务资产。</div>`
            : clusters.map(
                (cluster) => html`
                  <div class="alert-item">
                    <div class="alert-item__meta">
                      <span class="alert-badge alert-badge--info">${cluster.domain}</span>
                      <span class="alert-time">${cluster.owner || "未设置负责人"}</span>
                    </div>
                    <div class="alert-item__title">${cluster.name}</div>
                    <div class="alert-item__noise" style="display: flex; justify-content: space-between; align-items: center; margin-top: 6px;">
                      <div>
                        <span>区域: <strong>${cluster.region || "-"}</strong></span>
                        <span class="divider">|</span>
                        <span>组件: <strong>${cluster.components?.length ?? 0}</strong></span>
                      </div>
                      <button
                        class="ops-btn ops-btn--ghost"
                        style="padding: 2px 6px; font-size: 11px; display: inline-flex; align-items: center; gap: 4px; border: 1px solid var(--border-color, #ccc);"
                        @click=${() => props.onAnalyzeAsset?.({
                          domain: cluster.domain,
                          assetRef: cluster.name,
                          type: "service",
                          summary: `服务资产: ${cluster.name}, 负责人: ${cluster.owner || "未分配"}, 组件数: ${cluster.components?.length ?? 0}`
                        })}
                      >
                        ${icons.messageSquare} AI 分析
                      </button>
                    </div>
                  </div>
                `,
              )}
        </div>
      </div>
      <div class="ops-shell-panel detail-column">
        <div class="ops-shell-panel__head">${icons.users} 服务责任关系</div>
        <div style="padding:16px;">
          <p class="muted" style="margin:0;">
            服务资产由集群、组件、负责人和监控标签组成。后续可在这里接入服务依赖和 SLA。
          </p>
        </div>
      </div>
    </div>
  `;
}

function renderComponentAssets(
  rows: Array<{ cluster: AssetsViewProps["clusters"][number]; component: string }>,
  props: AssetsViewProps
) {
  return html`
    <div class="ops-shell-panel">
      <div class="ops-shell-panel__head">${icons.layout} 组件资产</div>
      <div style="padding:16px;">
        ${rows.length === 0
          ? html`<div class="empty-placeholder">当前技术域暂无组件资产。</div>`
          : html`
              <div class="ops-domain-stats">
                ${rows.map(
                  (row) => html`
                    <div class="ops-domain-stat" style="cursor:default; height: auto; min-height: 90px; display: flex; flex-direction: column; justify-content: space-between;">
                      <div>
                        <span class="ops-domain-stat__label">${row.cluster.name}</span>
                        <span class="ops-domain-stat__value" style="font-size:16px;">${row.component}</span>
                      </div>
                      <span class="ops-domain-stat__hint" style="display: flex; justify-content: space-between; align-items: center; width: 100%; margin-top: 8px;">
                        <span>${row.cluster.status || "unknown"} · ${row.cluster.owner || "未设置负责人"}</span>
                        <button
                          class="ops-btn ops-btn--ghost"
                          style="padding: 1px 4px; font-size: 10px; border: 1px solid var(--border-color, #ccc);"
                          @click=${() => props.onAnalyzeAsset?.({
                            domain: row.cluster.domain,
                            assetRef: `${row.cluster.name}/${row.component}`,
                            type: "component",
                            summary: `组件资产: ${row.component}, 所属集群: ${row.cluster.name}, 负责人: ${row.cluster.owner || "未分配"}`
                          })}
                        >
                          AI 分析
                        </button>
                      </span>
                    </div>
                  `,
                )}
              </div>
            `}
      </div>
    </div>
  `;
}

function renderJobAssets(
  rows: Array<{ id: string; name: string; cluster: AssetsViewProps["clusters"][number]; status?: string }>,
  props: AssetsViewProps
) {
  return html`
    <div class="ops-shell-panel">
      <div class="ops-shell-panel__head">${icons.activity} 作业资产</div>
      <div class="alert-list" style="padding:10px;">
        ${rows.length === 0
          ? html`<div class="empty-placeholder">未从组件资产中识别到作业类服务。后续可接入 Flink/Spark/YARN 作业 API。</div>`
          : rows.map(
              (row) => html`
                <div class="alert-item">
                  <div class="alert-item__meta">
                    <span class="alert-badge alert-badge--warning">${row.status || "unknown"}</span>
                    <span class="alert-time">${row.cluster.name}</span>
                  </div>
                  <div class="alert-item__title">${row.name}</div>
                  <div class="alert-item__noise" style="display: flex; justify-content: space-between; align-items: center; margin-top: 6px;">
                    <span class="muted">作业资产来源于当前集群组件，后续接入真实作业运行状态。</span>
                    <button
                      class="ops-btn ops-btn--ghost"
                      style="padding: 2px 6px; font-size: 11px; display: inline-flex; align-items: center; gap: 4px; border: 1px solid var(--border-color, #ccc);"
                      @click=${() => props.onAnalyzeAsset?.({
                        domain: row.cluster.domain,
                        assetRef: `${row.cluster.name}/${row.name}`,
                        type: "job",
                        summary: `作业资产: ${row.name}, 运行集群: ${row.cluster.name}`
                      })}
                    >
                      ${icons.messageSquare} AI 分析
                    </button>
                  </div>
                </div>
              `,
            )}
      </div>
    </div>
  `;
}

function renderTopologyAssets(
  clusters: AssetsViewProps["clusters"],
  rows: Array<{ cluster: AssetsViewProps["clusters"][number]; component: string }>,
) {
  return html`
    <div class="ops-shell-panel">
      <div class="ops-shell-panel__head">${icons.network} 拓扑关系</div>
      <div style="padding:16px;">
        ${renderOpsShellStatGrid([
          {
            label: "集群节点",
            value: clusters.length,
            hint: "服务拓扑一级节点",
            tone: "info",
            icon: "server",
          },
          {
            label: "组件节点",
            value: rows.length,
            hint: "由集群组件生成",
            tone: "ok",
            icon: "layout",
          },
        ])}
        <div class="detail-section" style="margin-top:16px;">
          <div class="detail-section__header">${icons.network} 拓扑摘要</div>
          <div class="detail-section__content">
            ${clusters.length === 0
              ? "当前技术域暂无拓扑数据。"
              : clusters
                  .map((cluster) => `${cluster.name} → ${(cluster.components || []).join(", ") || "暂无组件"}`)
                  .join("；")}
          </div>
        </div>
      </div>
    </div>
  `;
}
