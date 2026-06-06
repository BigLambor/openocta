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
import { renderOpsEmpty } from "../components/ops-status.ts";
import {
  ASSET_DOMAIN_LABEL,
  assetMonitorLinkLabel,
  assetStatusLabel,
} from "./asset-table-shared.ts";
import { monitorLinkStatus } from "../utils/monitor-labels.ts";

export type AssetViewId = "clusters" | "components" | "topology" | "services";

export type AssetsViewProps = AssetManagementProps & {
  selectedDomain?: string;
  activeAssetView?: AssetViewId;
  user?: DomainFilterUser;
  onDomainChange?: (domain: string) => void;
  onAssetViewChange?: (view: AssetViewId) => void;
};

const ASSET_VIEWS: OpsViewNavItem<AssetViewId>[] = [
  { id: "clusters", label: "集群资产", icon: "server" },
  { id: "components", label: "组件资产", icon: "layout" },
  { id: "topology", label: "拓扑关系", icon: "folder" },
  { id: "services", label: "服务资产", icon: "network" },
];

function clusterHealthCounts(clusters: AssetsViewProps["clusters"]) {
  let healthy = 0;
  let warning = 0;
  let critical = 0;
  for (const c of clusters ?? []) {
    if (c.status === "healthy") healthy++;
    else if (c.status === "warning") warning++;
    else if (c.status === "critical") critical++;
  }
  return { healthy, warning, critical };
}

function normalizeAssetView(raw?: string | null): AssetViewId {
  if (raw === "components" || raw === "topology" || raw === "services") {
    return raw;
  }
  return "clusters";
}

export function renderAssetsView(props: AssetsViewProps) {
  const selectedDomain = props.selectedDomain || "all";
  const normalized = normalizeOpsDomain(selectedDomain);

  const filteredClusters =
    normalized === "all"
      ? (props.clusters ?? [])
      : (props.clusters ?? []).filter((cluster) => cluster.domain === normalized);

  const activeAssetView = normalizeAssetView(props.activeAssetView);
  const { healthy, warning, critical } = clusterHealthCounts(filteredClusters);

  const componentRows = filteredClusters.flatMap((cluster) =>
    (cluster.components || []).map((component) => ({ cluster, component })),
  );

  const domainCountsWithAll = [
    { key: "all" as OpsDomainKey, label: "全部技术域", count: (props.clusters ?? []).length },
    ...OPS_DOMAIN_OPTIONS.filter((o) => o.key !== "all" && canAccessOpsDomain(props.user ?? null, o.key)).map(({ key, label }) => ({
      key,
      label,
      count: (props.clusters ?? []).filter((c) => c.domain === key).length,
    })),
  ];

  const sidebarViews: AssetsViewSidebarItem<AssetViewId>[] = ASSET_VIEWS.map((item) => {
    let viewCount: number | undefined;
    if (item.id === "clusters") viewCount = filteredClusters.length;
    else if (item.id === "components") viewCount = componentRows.length;
    else if (item.id === "topology") viewCount = filteredClusters.length;
    else if (item.id === "services") viewCount = undefined;

    return {
      id: item.id,
      label: item.id === "services" ? "服务资产（规划中）" : item.label,
      icon: item.icon,
      count: viewCount,
      muted: item.id === "services",
    };
  });

  return html`
    <div class="ops-assets-layout" style="display: flex; height: 100%; width: 100%;">
      ${renderAssetsContextSidebar({
        domains: domainCountsWithAll,
        selectedDomain: normalized,
        views: sidebarViews,
        activeView: activeAssetView,
        onDomainChange: props.onDomainChange,
        onViewChange: props.onAssetViewChange,
      })}

      <div style="flex: 1; min-width: 0;">
        <main class="ops-dashboard ops-shell" style="min-height: 100%; box-sizing: border-box;">
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
              ? renderServiceAssetsPreview(props)
              : activeAssetView === "components"
                ? renderComponentAssets(componentRows, props)
                : renderTopologyAssets(filteredClusters, componentRows)}
        </main>
      </div>
    </div>
  `;
}

function renderServiceAssetsPreview(props: AssetsViewProps) {
  return html`
    <div class="ops-panel asset-service-preview">
      <span class="asset-service-preview__badge">规划中</span>
      <h3 class="asset-service-preview__title">服务资产（预览）</h3>
      <p class="asset-service-preview__desc">
        独立服务登记（如 DataApp 应用服务、SLA、上下游依赖）能力尚在规划中。当前阶段请通过
        <strong>集群资产</strong> 完成纳管、负责人与监控标签维护。
      </p>
      <button
        type="button"
        class="ops-btn ops-btn--primary"
        @click=${() => props.onAssetViewChange?.("clusters")}
      >
        ${icons.server} 前往集群资产
      </button>
    </div>
    <style>
      .asset-service-preview {
        padding: 28px 32px;
        max-width: 640px;
      }
      .asset-service-preview__badge {
        display: inline-block;
        font-size: 11px;
        font-weight: 600;
        padding: 2px 8px;
        border-radius: 999px;
        color: var(--text-secondary);
        background: color-mix(in srgb, var(--text-secondary) 10%, var(--bg));
        margin-bottom: 12px;
      }
      .asset-service-preview__title {
        margin: 0 0 10px;
        font-size: 18px;
        font-weight: 600;
        color: var(--text-primary);
      }
      .asset-service-preview__desc {
        margin: 0 0 20px;
        font-size: 14px;
        line-height: 1.6;
        color: var(--text-secondary);
      }
    </style>
  `;
}

function renderComponentAssets(
  rows: Array<{ cluster: NonNullable<AssetsViewProps["clusters"]>[number]; component: string }>,
  props: AssetsViewProps,
) {
  const canManage = props.canManage !== false;

  return html`
    <div class="ops-panel asset-table-panel">
      <div class="asset-table-toolbar">
        <span class="asset-table-toolbar__title">组件列表</span>
        <span class="asset-table-toolbar__hint muted">由集群登记的核心组件展开，修改请回到集群资产</span>
      </div>
      ${rows.length === 0
        ? html`
            <div class="asset-table-panel__empty">
              ${renderOpsEmpty({
                icon: "layout",
                title: "当前技术域暂无组件资产",
                description: "请先在集群资产中登记核心组件（如 Atlas、DataHub、HDFS）。",
                actionLabel: "前往集群资产",
                onAction: () => props.onAssetViewChange?.("clusters"),
              })}
            </div>
          `
        : html`
            <table class="asset-table">
              <thead>
                <tr>
                  <th>组件名称</th>
                  <th>所属集群</th>
                  <th>业务域</th>
                  <th>区域</th>
                  <th>纳管状态</th>
                  <th>监控关联</th>
                  <th>负责人</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                ${rows.map(
                  (row) => html`
                    <tr>
                      <td style="font-weight: 500;">${row.component}</td>
                      <td>${row.cluster.name}</td>
                      <td>${ASSET_DOMAIN_LABEL[row.cluster.domain] ?? row.cluster.domain}</td>
                      <td>${row.cluster.region || "—"}</td>
                      <td>
                        <span class="asset-status asset-status--${row.cluster.status || "unknown"}">
                          ${assetStatusLabel(row.cluster.status || "unknown")}
                        </span>
                      </td>
                      <td>
                        <span
                          class="asset-monitor-link asset-monitor-link--${monitorLinkStatus(row.cluster.domain, row.cluster.status, row.cluster.monitorLabels)}"
                          title=${row.cluster.monitorLabels || "未配置 monitorLabels"}
                        >
                          ${assetMonitorLinkLabel(row.cluster.domain, row.cluster.status, row.cluster.monitorLabels)}
                        </span>
                      </td>
                      <td>${row.cluster.owner || "—"}</td>
                      <td>
                        ${canManage && props.onOpenEditDrawer
                          ? html`
                              <div class="asset-table__actions">
                                <button
                                  type="button"
                                  class="ops-btn ops-btn--ghost asset-table__action"
                                  @click=${() => {
                                    props.onAssetViewChange?.("clusters");
                                    props.onOpenEditDrawer?.(row.cluster.id);
                                  }}
                                >
                                  修改
                                </button>
                              </div>
                            `
                          : html`<span class="muted">—</span>`}
                      </td>
                    </tr>
                  `,
                )}
              </tbody>
            </table>
          `}
    </div>
  `;
}

function renderTopologyAssets(
  clusters: NonNullable<AssetsViewProps["clusters"]>,
  rows: Array<{ cluster: NonNullable<AssetsViewProps["clusters"]>[number]; component: string }>,
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
