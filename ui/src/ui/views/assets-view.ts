import { html } from "lit";
import { icons } from "../icons.ts";
import {
  OPS_DOMAIN_OPTIONS,
  opsDomainLabel,
  renderDomainFilter,
  normalizeOpsDomain,
  type DomainFilterUser,
} from "../components/domain-filter.ts";
import {
  renderOpsShellHeader,
  renderOpsShellStatGrid,
  renderOpsViewNav,
  type OpsViewNavItem,
} from "../components/ops-shell.ts";
import { renderAssetManagement, type AssetManagementProps } from "./asset-management.ts";

export type AssetsViewProps = AssetManagementProps & {
  selectedDomain?: string;
  activeAssetView?: "clusters" | "services" | "components" | "jobs" | "topology";
  user?: DomainFilterUser;
  onDomainChange?: (domain: string) => void;
  onAssetViewChange?: (view: "clusters" | "services" | "components" | "jobs" | "topology") => void;
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
  const filteredClusters =
    normalized === "all"
      ? props.clusters
      : props.clusters.filter((cluster) => cluster.domain === normalized);
  const activeAssetView = props.activeAssetView ?? "clusters";
  const { healthy, warning, critical } = clusterHealthCounts(filteredClusters);
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

  const domainCounts = OPS_DOMAIN_OPTIONS.filter((o) => o.key !== "all").map(({ key, label }) => ({
    key,
    label,
    count: props.clusters.filter((c) => c.domain === key).length,
  }));

  return html`
    <main class="ops-dashboard ops-shell">
      ${renderOpsShellHeader({
        kicker: `服务与资产 · ${opsDomainLabel(selectedDomain)}`,
        title: "资产目录",
        description: "技术域作为全局上下文过滤器，资产、拓扑和责任关系在此统一管理。",
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

      ${renderDomainFilter({
        selectedDomain,
        user: props.user ?? null,
        includeAll: true,
        onChange: (domain) => props.onDomainChange?.(domain),
      })}

      <section class="ops-domain-stats" aria-label="各技术域资产概览">
        <button
          type="button"
          class="ops-domain-stat ${normalized === "all" ? "ops-domain-stat--active" : ""}"
          @click=${() => props.onDomainChange?.("all")}
        >
          <span class="ops-domain-stat__label">全部技术域</span>
          <span class="ops-domain-stat__value">${props.clusters.length}</span>
          <span class="ops-domain-stat__hint">全部资产</span>
        </button>
        ${domainCounts.map(
          (item) => html`
            <button
              type="button"
              class="ops-domain-stat ${normalized === item.key ? "ops-domain-stat--active" : ""}"
              @click=${() => props.onDomainChange?.(item.key)}
            >
              <span class="ops-domain-stat__label">${item.label}</span>
              <span class="ops-domain-stat__value">${item.count}</span>
              <span class="ops-domain-stat__hint">技术域资产</span>
            </button>
          `,
        )}
      </section>

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

      ${renderOpsViewNav(ASSET_VIEWS, activeAssetView, (view) => props.onAssetViewChange?.(view))}

      ${activeAssetView === "clusters"
        ? renderAssetManagement({
            ...props,
            embedded: true,
            clusters: filteredClusters,
          })
        : activeAssetView === "services"
          ? renderServiceAssets(filteredClusters)
          : activeAssetView === "components"
            ? renderComponentAssets(componentRows)
            : activeAssetView === "jobs"
              ? renderJobAssets(jobRows)
              : renderTopologyAssets(filteredClusters, componentRows)}
    </main>
  `;
}

function renderServiceAssets(clusters: AssetsViewProps["clusters"]) {
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
                    <div class="alert-item__noise">
                      <span>区域: <strong>${cluster.region || "-"}</strong></span>
                      <span class="divider">|</span>
                      <span>组件: <strong>${cluster.components?.length ?? 0}</strong></span>
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

function renderComponentAssets(rows: Array<{ cluster: AssetsViewProps["clusters"][number]; component: string }>) {
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
                    <div class="ops-domain-stat" style="cursor:default;">
                      <span class="ops-domain-stat__label">${row.cluster.name}</span>
                      <span class="ops-domain-stat__value" style="font-size:16px;">${row.component}</span>
                      <span class="ops-domain-stat__hint"
                        >${row.cluster.status || "unknown"} · ${row.cluster.owner || "未设置负责人"}</span
                      >
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
                  <div class="muted">作业资产来源于当前集群组件，后续接入真实作业运行状态。</div>
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
