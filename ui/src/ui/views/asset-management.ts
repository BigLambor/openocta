import { html, nothing } from "lit";
import { icons } from "../icons.ts";
import type { OpsClusterRecord } from "../controllers/ops-clusters.ts";
import { renderOpsEmpty, renderOpsError, renderOpsSkeleton } from "../components/ops-status.ts";
import type { OpsMonitorLabelsField } from "../components/ops-monitor-labels-field.ts";
import "../components/ops-monitor-labels-field.ts";
import {
  ASSET_DOMAIN_LABEL,
  ASSET_DOMAIN_OPTIONS,
  assetMonitorLinkLabel,
  assetStatusLabel,
} from "./asset-table-shared.ts";
import { monitorLinkStatus } from "../utils/monitor-labels.ts";

export type ClusterFormPayload = {
  name: string;
  domain: string;
  region: string;
  nodeCount: number;
  components: string;
  owner: string;
  status: string;
  monitorLabels: string;
  vmUrlRef: string;
  metricsBaseUrl: string;
  jmxUrl: string;
  fiManagerUrl: string;
  gbaseDsnRef: string;
  credentialsRef: string;
};

export type AssetManagementProps = {
  /** 嵌入「服务与资产」页时隐藏重复标题与工具栏 */
  embedded?: boolean;
  clusters?: OpsClusterRecord[];
  loading?: boolean;
  error?: string | null;
  canManage?: boolean;
  drawerOpen?: boolean;
  drawerMode?: "add" | "edit";
  editingClusterId?: string | null;
  onRefresh?: () => void;
  onSyncCmdb?: () => void | Promise<void>;
  cmdbSyncing?: boolean;
  cmdbSyncHint?: string | null;
  onOpenSettings?: () => void;
  onOpenAddDrawer?: () => void;
  onOpenEditDrawer?: (clusterId: string) => void;
  onCloseDrawer?: () => void;
  onAddCluster?: (payload: ClusterFormPayload) => Promise<void>;
  onUpdateCluster?: (id: string, payload: ClusterFormPayload) => Promise<void>;
  onDeleteCluster?: (id: string) => Promise<void>;
};

function readClusterFormPayload(form: HTMLFormElement): ClusterFormPayload | null {
  const labelsField = form.querySelector("ops-monitor-labels-field") as OpsMonitorLabelsField | null;
  if (labelsField && !labelsField.checkValidity()) {
    labelsField.focus();
    return null;
  }
  const fd = new FormData(form);
  return {
    name: String(fd.get("name") ?? "").trim(),
    domain: String(fd.get("domain") ?? "hadoop"),
    region: String(fd.get("region") ?? "").trim(),
    nodeCount: Number(fd.get("nodeCount") ?? 0),
    components: String(fd.get("components") ?? "").trim(),
    owner: String(fd.get("owner") ?? "").trim(),
    status: String(fd.get("status") ?? "unknown"),
    monitorLabels: labelsField?.inputValue ?? "",
    vmUrlRef: String(fd.get("vmUrlRef") ?? "").trim(),
    metricsBaseUrl: String(fd.get("metricsBaseUrl") ?? "").trim(),
    jmxUrl: String(fd.get("jmxUrl") ?? "").trim(),
    fiManagerUrl: String(fd.get("fiManagerUrl") ?? "").trim(),
    gbaseDsnRef: String(fd.get("gbaseDsnRef") ?? "").trim(),
    credentialsRef: String(fd.get("credentialsRef") ?? "").trim(),
  };
}

function renderClusterForm(
  cluster: OpsClusterRecord | null,
  mode: "add" | "edit",
  onSubmit: (payload: ClusterFormPayload) => Promise<void>,
  onCancel: () => void,
) {
  const domain = cluster?.domain ?? "hadoop";
  const status = cluster?.status ?? "unknown";

  return html`
    <form
      class="asset-form"
      @submit=${async (e: Event) => {
        e.preventDefault();
        const form = e.target as HTMLFormElement;
        const payload = readClusterFormPayload(form);
        if (!payload) {
          return;
        }
        await onSubmit(payload);
      }}
    >
      <label class="asset-form__field">
        <span>集群名称</span>
        <input
          name="name"
          required
          class="asset-form__input"
          placeholder="例如：北京 BCH 生产"
          .value=${cluster?.name ?? ""}
        />
      </label>
      <label class="asset-form__field">
        <span>业务域</span>
        <span class="select">
          <select
            name="domain"
            .value=${domain}
            @change=${(e: Event) => {
              const select = e.target as HTMLSelectElement;
              const field = select.form?.querySelector(
                "ops-monitor-labels-field",
              ) as OpsMonitorLabelsField | null;
              if (field) {
                field.domain = select.value;
              }
            }}
          >
            ${ASSET_DOMAIN_OPTIONS.map((o) => html`<option value=${o.value}>${o.label}</option>`)}
          </select>
        </span>
      </label>
      <label class="asset-form__field">
        <span>区域</span>
        <input name="region" class="asset-form__input" placeholder="例如：北京" .value=${cluster?.region ?? ""} />
      </label>
      <label class="asset-form__field">
        <span>节点规模</span>
        <input
          name="nodeCount"
          type="number"
          min="0"
          class="asset-form__input"
          .value=${String(cluster?.nodeCount ?? 0)}
        />
      </label>
      <label class="asset-form__field span-2">
        <span>核心组件（逗号分隔）</span>
        <input
          name="components"
          class="asset-form__input"
          placeholder="HDFS, YARN, HIVE"
          .value=${(cluster?.components ?? []).join(", ")}
        />
      </label>
      <label class="asset-form__field">
        <span>负责人</span>
        <input name="owner" class="asset-form__input" placeholder="张三" .value=${cluster?.owner ?? ""} />
      </label>
      <label class="asset-form__field">
        <span>纳管状态</span>
        <span class="select">
          <select
            name="status"
            .value=${status}
            @change=${(e: Event) => {
              const select = e.target as HTMLSelectElement;
              const field = select.form?.querySelector(
                "ops-monitor-labels-field",
              ) as OpsMonitorLabelsField | null;
              if (field) {
                field.status = select.value;
              }
            }}
          >
            <option value="healthy">健康</option>
            <option value="warning">亚健康</option>
            <option value="critical">异常</option>
            <option value="unknown">未知</option>
            <option value="inactive">已下线</option>
          </select>
        </span>
      </label>
      <div class="asset-form__field span-2">
        <ops-monitor-labels-field
          domain=${domain}
          status=${status}
          initialLabels=${cluster?.monitorLabels ?? ""}
        ></ops-monitor-labels-field>
      </div>
      <label class="asset-form__field span-2">
        <span>VictoriaMetrics/Prometheus 引用 URL</span>
        <input
          name="vmUrlRef"
          class="asset-form__input"
          placeholder="例如：http://victoria-metrics:8428"
          .value=${cluster?.vmUrlRef ?? ""}
        />
      </label>
      <label class="asset-form__field span-2">
        <span>指标 Base URL</span>
        <input
          name="metricsBaseUrl"
          class="asset-form__input"
          placeholder="例如：http://prometheus:9090"
          .value=${cluster?.metricsBaseUrl ?? ""}
        />
      </label>
      <label class="asset-form__field span-2">
        <span>JMX URL</span>
        <input
          name="jmxUrl"
          class="asset-form__input"
          placeholder="例如：http://hadoop-namenode:8088"
          .value=${cluster?.jmxUrl ?? ""}
        />
      </label>
      <label class="asset-form__field span-2">
        <span>FI Manager URL</span>
        <input
          name="fiManagerUrl"
          class="asset-form__input"
          placeholder="例如：http://fi-manager:8080"
          .value=${cluster?.fiManagerUrl ?? ""}
        />
      </label>
      <label class="asset-form__field span-2">
        <span>GBase DSN 引用</span>
        <input
          name="gbaseDsnRef"
          class="asset-form__input"
          placeholder="例如：gbase_dsn_production"
          .value=${cluster?.gbaseDsnRef ?? ""}
        />
      </label>
      <label class="asset-form__field span-2">
        <span>敏感凭证引用</span>
        <input
          name="credentialsRef"
          class="asset-form__input"
          placeholder="例如：secret_credentials"
          .value=${cluster?.credentialsRef ?? ""}
        />
      </label>
      <div class="asset-form__actions span-2">
        <button type="button" class="ops-btn" @click=${onCancel}>取消</button>
        <button type="submit" class="ops-btn ops-btn--primary">
          ${mode === "edit" ? "保存修改" : "保存集群"}
        </button>
      </div>
    </form>
  `;
}

function renderClusterDrawer(props: AssetManagementProps) {
  if (!props.drawerOpen) {
    return nothing;
  }
  const mode = props.drawerMode ?? "add";
  const editing =
    mode === "edit" && props.editingClusterId
      ? (props.clusters ?? []).find((c) => c.id === props.editingClusterId) ?? null
      : null;

  const handleSubmit = async (payload: ClusterFormPayload) => {
    if (mode === "edit" && editing) {
      await props.onUpdateCluster?.(editing.id, payload);
    } else {
      await props.onAddCluster?.(payload);
    }
    props.onCloseDrawer?.();
  };

  return html`
    <div class="ops-ai-drawer__overlay" @click=${() => props.onCloseDrawer?.()}></div>
    <aside class="ops-ai-drawer">
      <div class="ops-ai-drawer__header">
        <div class="ops-ai-drawer__heading">
          <span class="ops-ai-drawer__icon">${icons.server}</span>
          <div>
            <div class="ops-ai-drawer__title">${mode === "edit" ? "修改纳管集群" : "新增纳管集群"}</div>
            ${editing
              ? html`<div class="ops-ai-drawer__subtitle">${editing.name}</div>`
              : html`<div class="ops-ai-drawer__subtitle">填写集群基础信息与监控关联</div>`}
          </div>
        </div>
        <button
          type="button"
          class="ops-btn ops-btn--ghost ops-btn--icon"
          title="关闭"
          @click=${() => props.onCloseDrawer?.()}
        >
          ${icons.x}
        </button>
      </div>
      <div class="ops-ai-drawer__body">
        ${renderClusterForm(editing, mode, handleSubmit, () => props.onCloseDrawer?.())}
      </div>
    </aside>
  `;
}

function renderTableToolbar(props: AssetManagementProps) {
  if (!props.canManage || !props.onOpenAddDrawer) {
    return nothing;
  }
  return html`
    <div class="asset-table-toolbar">
      <span class="asset-table-toolbar__title">集群列表</span>
      <button type="button" class="ops-btn ops-btn--primary" @click=${() => props.onOpenAddDrawer?.()}>
        ${icons.plus} 新增纳管集群
      </button>
    </div>
  `;
}

export function renderAssetManagement(props: AssetManagementProps = {}) {
  const clusters = props.clusters ?? [];
  const hasData = clusters.length > 0;
  const canManage = props.canManage !== false;

  const body = html`
    ${!props.embedded && props.cmdbSyncHint
      ? html`<p class="ops-muted-hint" style="margin: 0 0 12px; font-size: 12px;">${props.cmdbSyncHint}</p>`
      : nothing}

    ${props.error
      ? html`
          <div class="ops-panel" style="margin-bottom: 16px;">
            ${renderOpsError({ message: props.error, onRetry: props.onRefresh })}
          </div>
        `
      : nothing}

    ${props.loading
      ? html`
          <div class="ops-panel">${renderOpsSkeleton({ lines: 5 })}</div>
        `
      : html`
          <div class="ops-panel asset-table-panel">
            ${renderTableToolbar(props)}
            ${!hasData && !props.error
              ? html`
                  <div class="asset-table-panel__empty">
                    ${renderOpsEmpty({
                      icon: "server",
                      title: "尚未纳管任何集群",
                      description: canManage
                        ? "点击右上角「新增纳管集群」登记第一台集群，或确认当前账号有 menu:config 权限。"
                        : "当前账号暂无纳管权限，请联系管理员。",
                      actionLabel:
                        canManage && props.onOpenAddDrawer ? "新增纳管集群" : props.onOpenSettings ? "打开系统配置" : undefined,
                      onAction: canManage && props.onOpenAddDrawer ? props.onOpenAddDrawer : props.onOpenSettings,
                    })}
                  </div>
                `
              : html`
                  <table class="asset-table">
                    <thead>
                      <tr>
                        <th>集群名称</th>
                        <th>业务域</th>
                        <th>区域</th>
                        <th>节点规模</th>
                        <th>核心组件</th>
                        <th>纳管状态</th>
                        <th>监控关联</th>
                        <th>负责人</th>
                        <th>操作</th>
                      </tr>
                    </thead>
                    <tbody>
                      ${clusters.map(
                        (row) => html`
                          <tr>
                            <td style="font-weight: 500;">${row.name}</td>
                            <td>${ASSET_DOMAIN_LABEL[row.domain] ?? row.domain}</td>
                            <td>${row.region || "—"}</td>
                            <td>${row.nodeCount} Nodes</td>
                            <td>${(row.components ?? []).join(", ") || "—"}</td>
                            <td>
                              <span class="asset-status asset-status--${row.status}">
                                ${assetStatusLabel(row.status)}
                              </span>
                            </td>
                            <td>
                              <span
                                class="asset-monitor-link asset-monitor-link--${monitorLinkStatus(row.domain, row.status, row.monitorLabels)}"
                                title=${row.monitorLabels || "未配置 monitorLabels"}
                              >
                                ${assetMonitorLinkLabel(row.domain, row.status, row.monitorLabels)}
                              </span>
                            </td>
                            <td>${row.owner || "—"}</td>
                            <td>
                              ${canManage
                                ? html`
                                    <div class="asset-table__actions">
                                      <button
                                        type="button"
                                        class="ops-btn ops-btn--ghost asset-table__action"
                                        @click=${() => props.onOpenEditDrawer?.(row.id)}
                                      >
                                        修改
                                      </button>
                                      <button
                                        type="button"
                                        class="ops-btn ops-btn--ghost asset-table__action asset-table__action--danger"
                                        @click=${async () => {
                                          if (!window.confirm(`确定删除集群「${row.name}」？此操作不可撤销。`)) {
                                            return;
                                          }
                                          await props.onDeleteCluster?.(row.id);
                                        }}
                                      >
                                        删除
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
        `}

    ${renderClusterDrawer(props)}

    <style>
      .asset-form {
        display: grid;
        grid-template-columns: 1fr;
        gap: 16px;
      }
      @media (min-width: 640px) {
        .asset-form {
          grid-template-columns: repeat(2, 1fr);
        }
        .asset-form__field.span-2,
        .asset-form__actions.span-2 {
          grid-column: span 2;
        }
      }
      .asset-form__field {
        display: flex;
        flex-direction: column;
        gap: 6px;
        font-size: 12px;
        color: var(--text-secondary);
      }
      .asset-form__input {
        padding: 8px 10px;
        border-radius: 8px;
        border: 1px solid var(--border);
        background: var(--bg);
        color: var(--text-primary);
        font-size: 13px;
      }
      .asset-form__actions {
        display: flex;
        justify-content: flex-end;
        gap: 8px;
        margin-top: 8px;
      }
    </style>
  `;

  if (props.embedded) {
    return body;
  }

  return html`
    <div class="ops-page">
      <div class="ops-page-header">
        <div>
          <h1>集群资产管理</h1>
          <p>
            登记各业务域集群，供运维大屏汇总与上下文选择器使用。监控关联依赖
            <code>monitorLabels</code>（非资产 id），详见仓库文档
            <code>docs/ops-monitor-labels-checklist.md</code>。
          </p>
        </div>
        <div class="ops-toolbar">
          ${props.onSyncCmdb
            ? html`
                <button
                  type="button"
                  class="ops-btn ops-btn--primary"
                  ?disabled=${props.loading || props.cmdbSyncing}
                  title=${props.cmdbSyncHint ?? "从 OPS_CMDB_SYNC_URL 拉取并合并集群"}
                  @click=${() => props.onSyncCmdb?.()}
                >
                  <span style="width: 16px; height: 16px; display: flex;">${icons.refreshCw}</span>
                  ${props.cmdbSyncing ? "同步中…" : "同步 CMDB"}
                </button>
              `
            : nothing}
          <button type="button" class="ops-btn" ?disabled=${props.loading || props.cmdbSyncing} @click=${() => props.onRefresh?.()}>
            <span style="width: 16px; height: 16px; display: flex;">${icons.refreshCw}</span>
            刷新
          </button>
        </div>
      </div>
      ${body}
    </div>
  `;
}
