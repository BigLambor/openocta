import { html, nothing } from "lit";
import { icons } from "../icons.ts";
import type { OpsClusterRecord } from "../controllers/ops-clusters.ts";
import { renderOpsEmpty, renderOpsError, renderOpsSkeleton } from "../components/ops-status.ts";
import type { OpsMonitorLabelsField } from "../components/ops-monitor-labels-field.ts";
import { monitorLinkStatus } from "../utils/monitor-labels.ts";
import "../components/ops-monitor-labels-field.ts";

export type AssetManagementProps = {
  /** 嵌入「服务与资产」页时隐藏重复标题与工具栏 */
  embedded?: boolean;
  clusters?: OpsClusterRecord[];
  loading?: boolean;
  error?: string | null;
  canManage?: boolean;
  onRefresh?: () => void;
  onSyncCmdb?: () => void | Promise<void>;
  cmdbSyncing?: boolean;
  cmdbSyncHint?: string | null;
  onOpenSettings?: () => void;
  onAddCluster?: (payload: {
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
  }) => Promise<void>;
  onAnalyzeAsset?: (params: { domain: string; assetRef: string; type: string; summary: string }) => void;
};

const DOMAIN_OPTIONS = [
  { value: "hadoop", label: "BCH生态" },
  { value: "fi", label: "FI商业生态" },
  { value: "gbase", label: "GBase数据库" },
  { value: "governance", label: "开发治理平台" },
  { value: "dataapps", label: "数据App运维" },
];

const DOMAIN_LABEL: Record<string, string> = Object.fromEntries(
  DOMAIN_OPTIONS.map((o) => [o.value, o.label]),
);

function monitorLinkLabel(domain: string, status: string, monitorLabels?: string) {
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

function statusLabel(status: string) {
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

export function renderAssetManagement(props: AssetManagementProps = {}) {
  const clusters = props.clusters ?? [];
  const hasData = clusters.length > 0;
  const showForm = props.canManage !== false;

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

      ${showForm && props.onAddCluster
        ? html`
            <details class="ops-panel asset-form-panel" style="margin-bottom: 16px; padding: 16px;">
              <summary style="cursor: pointer; font-size: 14px; font-weight: 600;">新增纳管集群</summary>
              <form
                class="asset-form"
                @submit=${async (e: Event) => {
                  e.preventDefault();
                  const form = e.target as HTMLFormElement;
                  const labelsField = form.querySelector(
                    "ops-monitor-labels-field",
                  ) as OpsMonitorLabelsField | null;
                  if (labelsField && !labelsField.checkValidity()) {
                    labelsField.focus();
                    return;
                  }
                  const fd = new FormData(form);
                  await props.onAddCluster?.({
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
                  });
                  form.reset();
                }}
              >
                <label class="asset-form__field">
                  <span>集群名称</span>
                  <input name="name" required class="asset-form__input" placeholder="例如：北京 BCH 生产" />
                </label>
                <label class="asset-form__field">
                  <span>业务域</span>
                  <span class="select">
                    <select
                      name="domain"
                      @change=${(e: Event) => {
                        const select = e.target as HTMLSelectElement;
                        const field = select.form?.querySelector(
                          "ops-monitor-labels-field",
                        ) as OpsMonitorLabelsField | null;
                        if (field) field.domain = select.value;
                      }}
                    >
                      ${DOMAIN_OPTIONS.map(
                        (o) => html`<option value=${o.value}>${o.label}</option>`,
                      )}
                    </select>
                  </span>
                </label>
                <label class="asset-form__field">
                  <span>区域</span>
                  <input name="region" class="asset-form__input" placeholder="例如：北京" />
                </label>
                <label class="asset-form__field">
                  <span>节点规模</span>
                  <input name="nodeCount" type="number" min="0" value="0" class="asset-form__input" />
                </label>
                <label class="asset-form__field span-2">
                  <span>核心组件（逗号分隔）</span>
                  <input name="components" class="asset-form__input" placeholder="HDFS, YARN, HIVE" />
                </label>
                <label class="asset-form__field">
                  <span>负责人</span>
                  <input name="owner" class="asset-form__input" placeholder="张三" />
                </label>
                <label class="asset-form__field">
                  <span>纳管状态</span>
                  <span class="select">
                    <select
                      name="status"
                      @change=${(e: Event) => {
                        const select = e.target as HTMLSelectElement;
                        const field = select.form?.querySelector(
                          "ops-monitor-labels-field",
                        ) as OpsMonitorLabelsField | null;
                        if (field) field.status = select.value;
                      }}
                    >
                      <option value="healthy">健康</option>
                      <option value="warning">亚健康</option>
                      <option value="critical">异常</option>
                      <option value="unknown" selected>未知</option>
                      <option value="inactive">已下线</option>
                    </select>
                  </span>
                </label>
                <div class="asset-form__field span-2">
                  <ops-monitor-labels-field domain="hadoop" status="unknown"></ops-monitor-labels-field>
                </div>
                <label class="asset-form__field span-2">
                  <span>VictoriaMetrics/Prometheus 引用 URL</span>
                  <input name="vmUrlRef" class="asset-form__input" placeholder="例如：http://victoria-metrics:8428" />
                </label>
                <label class="asset-form__field span-2">
                  <span>指标 Base URL</span>
                  <input name="metricsBaseUrl" class="asset-form__input" placeholder="例如：http://prometheus:9090" />
                </label>
                <label class="asset-form__field span-2">
                  <span>JMX URL</span>
                  <input name="jmxUrl" class="asset-form__input" placeholder="例如：http://hadoop-namenode:8088" />
                </label>
                <label class="asset-form__field span-2">
                  <span>FI Manager URL</span>
                  <input name="fiManagerUrl" class="asset-form__input" placeholder="例如：http://fi-manager:8080" />
                </label>
                <label class="asset-form__field span-2">
                  <span>GBase DSN 引用</span>
                  <input name="gbaseDsnRef" class="asset-form__input" placeholder="例如：gbase_dsn_production" />
                </label>
                <label class="asset-form__field span-2">
                  <span>敏感凭证引用</span>
                  <input name="credentialsRef" class="asset-form__input" placeholder="例如：secret_credentials" />
                </label>
                <div class="asset-form__field span-2" style="margin-top: 8px;">
                  <button type="submit" class="ops-btn ops-btn--primary" style="width: fit-content;">
                    保存集群
                  </button>
                </div>
              </form>
            </details>
          `
        : nothing}

      ${props.loading
        ? html`
            <div class="ops-panel">${renderOpsSkeleton({ lines: 5 })}</div>
          `
        : !hasData && !props.error
          ? html`
              <div class="ops-panel">
                ${renderOpsEmpty({
                  icon: "server",
                  title: "尚未纳管任何集群",
                  description: "点击上方「新增纳管集群」登记第一台集群，或确认当前账号有 menu:config 权限。",
                  actionLabel: props.onOpenSettings ? "打开系统配置" : undefined,
                  onAction: props.onOpenSettings,
                })}
              </div>
            `
          : html`
              <div class="ops-panel">
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
                          <td>${DOMAIN_LABEL[row.domain] ?? row.domain}</td>
                          <td>${row.region || "—"}</td>
                          <td>${row.nodeCount} Nodes</td>
                          <td>${(row.components ?? []).join(", ") || "—"}</td>
                          <td>
                            <span class="asset-status asset-status--${row.status}">
                              ${statusLabel(row.status)}
                            </span>
                          </td>
                          <td>
                            <span
                              class="asset-monitor-link asset-monitor-link--${monitorLinkStatus(row.domain, row.status, row.monitorLabels)}"
                              title=${row.monitorLabels || "未配置 monitorLabels"}
                            >
                              ${monitorLinkLabel(row.domain, row.status, row.monitorLabels)}
                            </span>
                          </td>
                          <td>${row.owner || "—"}</td>
                          <td>
                            <button
                              class="ops-btn ops-btn--ghost"
                              style="padding: 2px 6px; font-size: 12px; display: inline-flex; align-items: center; gap: 4px;"
                              @click=${() => props.onAnalyzeAsset?.({
                                domain: row.domain,
                                assetRef: row.name,
                                type: "cluster",
                                summary: `集群名称: ${row.name}, 节点规模: ${row.nodeCount}, 组件: ${(row.components ?? []).join(", ")}, 负责人: ${row.owner || "未分配"}`
                              })}
                            >
                              ${icons.messageSquare} AI 分析
                            </button>
                          </td>
                        </tr>
                      `,
                    )}
                  </tbody>
                </table>
              </div>
            `}
    <style>
      .asset-form {
        margin-top: 16px;
        display: grid;
        grid-template-columns: 1fr;
        gap: 16px;
        max-width: 800px;
      }
      @media (min-width: 640px) {
        .asset-form {
          grid-template-columns: repeat(2, 1fr);
        }
        .asset-form__field.span-2 {
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
      .asset-table {
        width: 100%;
        border-collapse: collapse;
      }
      .asset-table th,
      .asset-table td {
        padding: 14px 18px;
        text-align: left;
        border-bottom: 1px solid var(--border);
        font-size: 13px;
      }
      .asset-table th {
        font-weight: 500;
        color: var(--text-secondary);
        background: var(--bg-content);
        border-bottom: 2px solid var(--border);
      }
      .asset-table tr {
        transition: background var(--duration-fast) ease;
      }
      .asset-table tr:hover {
        background: var(--bg-content);
      }
      .asset-table tr:last-child td {
        border-bottom: none;
      }
      .asset-status {
        padding: 3px 8px;
        border-radius: 20px;
        font-size: 12px;
        font-weight: 500;
      }
      .asset-status--healthy {
        background: rgba(16, 185, 129, 0.1);
        color: #10b981;
        border: 1px solid rgba(16, 185, 129, 0.2);
      }
      .asset-status--warning {
        background: rgba(245, 158, 11, 0.1);
        color: #f59e0b;
        border: 1px solid rgba(245, 158, 11, 0.2);
      }
      .asset-status--critical {
        background: rgba(239, 68, 68, 0.1);
        color: #ef4444;
        border: 1px solid rgba(239, 68, 68, 0.2);
      }
      .asset-status--unknown {
        background: rgba(255, 255, 255, 0.05);
        color: var(--text-muted);
        border: 1px solid var(--border);
      }
      .asset-status--inactive {
        background: rgba(148, 163, 184, 0.12);
        color: #94a3b8;
        border: 1px solid rgba(148, 163, 184, 0.24);
      }
      .asset-monitor-link {
        font-size: 12px;
        padding: 2px 8px;
        border-radius: 12px;
        border: 1px solid var(--border);
      }
      .asset-monitor-link--linked {
        color: #10b981;
        border-color: rgba(16, 185, 129, 0.3);
        background: rgba(16, 185, 129, 0.08);
      }
      .asset-monitor-link--missing {
        color: #f59e0b;
        border-color: rgba(245, 158, 11, 0.3);
        background: rgba(245, 158, 11, 0.08);
      }
      .asset-monitor-link--invalid {
        color: #ef4444;
        border-color: rgba(239, 68, 68, 0.3);
        background: rgba(239, 68, 68, 0.08);
      }
      .asset-monitor-link--na {
        color: var(--text-muted);
        border-color: var(--border);
        background: transparent;
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
      ${props.cmdbSyncHint
        ? html`<p class="ops-muted-hint" style="margin: 0 0 12px; font-size: 12px;">${props.cmdbSyncHint}</p>`
        : nothing}
      ${body}
    </div>
  `;
}
