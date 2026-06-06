import { LitElement, html, css, nothing } from "lit";
import { customElement, property, state } from "lit/decorators.js";
import { fetchBchHdfsFsImage, HdfsFsImageStats } from "../../controllers/bch-client.ts";
import { parseWorkbenchObjectScope } from "../../ops/workbench-context.ts";

@customElement("bch-fsimage-dashboard")
export class BchFsImageDashboard extends LitElement {
  @property({ type: Object }) host: any = null;
  @property({ type: String }) activeCluster = "all";
  @property({ type: String }) activeNamespace = "NS1";
  @property({ type: String }) objectScope = "all";
  @property({ type: String }) timeRange = "24h";

  @state() private stats: HdfsFsImageStats | null = null;
  @state() private loading = false;
  @state() private error: string | null = null;
  @state() private aiPreparing = false;

  static styles = css`
    :host {
      display: flex;
      flex-direction: column;
      font-family: var(--font-family, sans-serif);
      color: var(--text-primary);
      height: 100%;
      box-sizing: border-box;
    }

    .ns-tabs {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 12px 24px;
      border-bottom: 1px solid var(--border);
      background: var(--bg-content);
    }

    .ns-tab-btn {
      background: var(--bg);
      border: 1px solid var(--border);
      color: var(--text-muted);
      padding: 6px 16px;
      border-radius: 6px;
      cursor: pointer;
      font-size: 12px;
      transition: all 0.2s;
    }

    .ns-tab-btn:hover {
      color: var(--text-primary);
      border-color: var(--accent, #3b82f6);
    }

    .ns-tab-btn.active {
      background: var(--accent, #3b82f6);
      color: white;
      border-color: var(--accent, #3b82f6);
      box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
    }

    .dashboard-body {
      flex: 1;
      padding: 24px;
      overflow-y: auto;
    }

    .hdfs-dashboard-head {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 16px;
      margin-bottom: 18px;
    }

    .hdfs-dashboard-title {
      font-size: 15px;
      font-weight: 700;
      color: var(--text-primary);
    }

    .hdfs-dashboard-subtitle {
      margin-top: 4px;
      color: var(--text-muted);
      font-size: 12px;
    }

    .ai-namespace-link {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      border: 1px solid var(--border);
      background: var(--bg);
      color: var(--text-secondary);
      border-radius: 8px;
      padding: 7px 10px;
      font-size: 12px;
      font-weight: 500;
      cursor: pointer;
      transition: all 0.15s ease;
      white-space: nowrap;
    }

    .ai-namespace-link:hover:not(:disabled) {
      color: var(--accent, #3b82f6);
      border-color: rgba(59, 130, 246, 0.35);
      background: rgba(59, 130, 246, 0.05);
    }

    .ai-namespace-link:disabled {
      opacity: 0.55;
      cursor: not-allowed;
    }

    /* Top Cards Grid */
    .summary-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
      gap: 16px;
      margin-bottom: 24px;
    }

    .summary-card {
      background: var(--bg-content);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 16px;
      text-align: center;
      box-shadow: var(--shadow-sm, 0 2px 8px rgba(0, 0, 0, 0.05));
    }

    .summary-num {
      font-size: 20px;
      font-weight: 700;
      color: var(--text-primary);
    }

    .summary-lbl {
      font-size: 11px;
      color: var(--text-muted);
      margin-top: 4px;
    }

    /* Primary Visual Sections Split */
    .sections-grid {
      display: grid;
      grid-template-columns: 1fr 1.2fr;
      gap: 20px;
      margin-bottom: 24px;
    }

    .section-card {
      background: var(--bg-content);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 20px;
      box-shadow: var(--shadow-sm, 0 4px 12px rgba(0, 0, 0, 0.05));
    }

    .section-title {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 10px;
      font-size: 14px;
      font-weight: 600;
      border-bottom: 1px solid var(--border);
      padding-bottom: 8px;
      margin-bottom: 16px;
      color: var(--text-secondary);
    }

    .section-ai-btn {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: 30px;
      height: 30px;
      border: 1px solid rgba(59, 130, 246, 0.28);
      border-radius: 999px;
      background: rgba(59, 130, 246, 0.06);
      color: var(--accent, #3b82f6);
      cursor: pointer;
      flex: 0 0 auto;
      font-size: 15px;
    }

    .section-ai-btn:hover {
      background: var(--accent, #3b82f6);
      color: #fff;
    }

    /* Visual Distribution Curves */
    .bar-chart-list {
      display: flex;
      flex-direction: column;
      gap: 12px;
    }

    .bar-chart-item {
      font-size: 12px;
    }

    .bar-info-row {
      display: flex;
      justify-content: space-between;
      margin-bottom: 4px;
      color: var(--text-secondary);
    }

    .bar-value {
      font-weight: 600;
      color: var(--text-primary);
    }

    .bar-track {
      height: 8px;
      background: var(--bg, rgba(0, 0, 0, 0.05));
      border-radius: 4px;
      overflow: hidden;
    }

    .bar-fill {
      height: 100%;
      background: linear-gradient(90deg, #3b82f6 0%, #60a5fa 100%);
      border-radius: 4px;
    }

    /* Tables */
    .ops-table {
      width: 100%;
      border-collapse: collapse;
      font-size: 12px;
    }

    .ops-table th {
      padding: 8px 12px;
      text-align: left;
      font-weight: 600;
      border-bottom: 1px solid var(--border);
      color: var(--text-muted);
    }

    .ops-table td {
      padding: 10px 12px;
      border-bottom: 1px solid var(--border);
      color: var(--text-primary);
    }

    .ops-table tr:hover {
      background: var(--bg-hover, rgba(0, 0, 0, 0.02));
    }

    .warning-banners {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 16px;
      margin-bottom: 24px;
    }

    .warning-banner-card {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 14px 16px;
      border-radius: 8px;
      font-size: 12px;
    }

    .warning-banner-card.warn {
      background: rgba(245, 158, 11, 0.06);
      border: 1px solid rgba(245, 158, 11, 0.15);
      color: #f59e0b;
    }

    .warning-banner-card.critical {
      background: rgba(239, 68, 68, 0.06);
      border: 1px solid rgba(239, 68, 68, 0.15);
      color: #ef4444;
    }

    .warning-title {
      font-weight: 600;
      margin-bottom: 2px;
    }

    .warning-desc {
      font-size: 11px;
      opacity: 0.8;
    }
  `;

  connectedCallback() {
    super.connectedCallback();
    this.loadData();
  }

  updated(changed: Map<string, unknown>) {
    if (changed.has("activeNamespace") && this.host) {
      void this.loadData();
    }
  }

  async loadData() {
    if (!this.host) return;
    this.loading = true;
    this.error = null;
    try {
      this.stats = await fetchBchHdfsFsImage(this.host, this.activeNamespace);
    } catch (err: any) {
      this.error = err.message || String(err);
    } finally {
      this.loading = false;
    }
  }

  switchNamespace(ns: string) {
    if (this.activeNamespace === ns) return;
    this.activeNamespace = ns;
    this.loadData();
  }

  private dispatchAiRequest(detail: {
    title: string;
    objectType: string;
    objectId: string;
    objectScope: string;
    evidence: string[];
    expectedOutputs: string[];
    mode?: "root-cause" | "similar" | "action";
  }) {
    this.dispatchEvent(
      new CustomEvent("scenario-ai-request", {
        bubbles: true,
        composed: true,
        detail: {
          mode: detail.mode ?? "root-cause",
          timeRange: this.timeRange,
          ...detail,
        },
      }),
    );
  }

  private namespaceEvidence(stats: HdfsFsImageStats): string[] {
    return [
      `Namespace: ${stats.namespace}`,
      `元数据总记录数: ${stats.totalRecords}`,
      `文件总数: ${stats.totalFiles}`,
      `目录总数: ${stats.totalDirs}`,
      `存储容量总大小: ${stats.totalSize}`,
      `平均文件大小: ${stats.avgFileSize}`,
      `最大目录树深度: ${stats.maxDepth}`,
      `Trash 未清理: ${stats.trashFiles}`,
      `零字节文件: ${stats.zeroByteFiles}`,
    ];
  }

  private async askAiForAllNamespaces() {
    if (!this.host || this.aiPreparing) return;
    this.aiPreparing = true;
    try {
      const namespaces = ["NS1", "NS2", "NS3", "NS4", "NS5", "NS6", "NS7", "NS8"];
      const results = await Promise.all(
        namespaces.map((ns) => fetchBchHdfsFsImage(this.host, ns).catch(() => null)),
      );
      const evidence = results
        .filter((item): item is HdfsFsImageStats => !!item)
        .flatMap((item) => [
          `${item.namespace}: records=${item.totalRecords}, files=${item.totalFiles}, dirs=${item.totalDirs}, size=${item.totalSize}, avgFile=${item.avgFileSize}, maxDepth=${item.maxDepth}, trash=${item.trashFiles}, zeroByte=${item.zeroByteFiles}`,
        ]);
      this.dispatchAiRequest({
        mode: "root-cause",
        title: "全部 Namespace · AI 容量分析",
        objectType: "hdfs_namespace_set",
        objectId: "all-namespaces",
        objectScope: "all namespaces",
        evidence: evidence.length ? evidence : ["未能拉取全部 Namespace 摘要，请基于当前可见 FSImage 数据分析。"],
        expectedOutputs: ["跨 Namespace 风险排序", "容量/小文件/目录深度根因", "优先治理对象", "清理与扩容建议"],
      });
    } finally {
      this.aiPreparing = false;
    }
  }

  private askAiForChart(
    chartId: string,
    title: string,
    evidence: string[],
    expectedOutputs: string[],
    mode: "root-cause" | "similar" | "action" = "root-cause",
  ) {
    const stats = this.stats;
    const namespace = stats?.namespace || this.activeNamespace;
    this.dispatchAiRequest({
      mode,
      title: `${title} · 问问 AI`,
      objectType: "hdfs_chart",
      objectId: `${namespace}:${chartId}`,
      objectScope: namespace,
      evidence: [
        ...(stats ? this.namespaceEvidence(stats) : [`Namespace: ${namespace}`]),
        ...evidence,
      ],
      expectedOutputs,
    });
  }

  private renderSectionTitle(
    title: string,
    ask: () => void,
  ) {
    return html`
      <div class="section-title">
        <span>${title}</span>
        <button class="section-ai-btn" type="button" title="问问 AI" @click=${ask}>
          🤖
        </button>
      </div>
    `;
  }

  render() {
    const parsed = parseWorkbenchObjectScope(this.objectScope);
    const directoryPath = parsed.kind === "directory" ? parsed.value : null;
    const clusterName = parsed.cluster ?? (this.activeCluster === "all" ? "" : this.activeCluster);
    const namespaceName = parsed.namespace ?? this.activeNamespace;

    return html`
      <div class="ns-tabs">
        ${["NS1", "NS2", "NS3", "NS4", "NS5", "NS6", "NS7", "NS8"].map((ns) => html`
          <button
            class="ns-tab-btn ${this.activeNamespace === ns ? "active" : ""}"
            @click=${() => this.switchNamespace(ns)}
          >
            ${ns}
          </button>
        `)}
      </div>

      <div class="dashboard-body">
        ${directoryPath
          ? html`
              <div class="ops-banner info" style="margin-bottom: 20px; display: flex; align-items: center; gap: 10px; padding: 12px 18px; border-radius: 8px; background: rgba(59, 130, 246, 0.08); border: 1px solid rgba(59, 130, 246, 0.2); font-size: 13px;">
                <span style="font-weight:600;">目录</span>
                <span>
                  当前 HDFS 静态治理热点目录：
                  ${clusterName ? html`<strong style="color: var(--accent, #3b82f6); font-family: monospace;">${clusterName}</strong> / ` : nothing}
                  <strong style="color: var(--accent, #3b82f6); font-family: monospace;">${namespaceName}${directoryPath}</strong>。
                  此入口用于小文件/容量治理聚焦，不代表实时目录树枚举。
                </span>
              </div>
            `
          : nothing}
        ${this.loading
          ? html`
              <div class="loading-container" style="display: flex; flex-direction: column; align-items: center; justify-content: center; height: 200px; color: var(--text-muted);">
                <div class="spinner" style="width: 24px; height: 24px; border: 2px solid rgba(255, 255, 255, 0.1); border-top-color: var(--accent, #3b82f6); border-radius: 50%; animation: spin 0.8s linear infinite; margin-bottom: 12px;"></div>
                <div>正在读取 HDFS 目录树离线 FSImage 元数据...</div>
              </div>
            `
          : this.stats
          ? this.renderDashboard()
          : nothing}
      </div>
    `;
  }

  private renderDashboard() {
    const stats = this.stats!;

    return html`
      <div class="hdfs-dashboard-head">
        <div>
          <div class="hdfs-dashboard-title">${stats.namespace} 容量与元数据画像</div>
          <div class="hdfs-dashboard-subtitle">基于离线 FSImage 的 namespace 级容量、小文件、目录深度和冷热数据分析</div>
        </div>
        <button
          class="ai-namespace-link"
          type="button"
          ?disabled=${this.aiPreparing}
          @click=${() => this.askAiForAllNamespaces()}
        >
          🤖 ${this.aiPreparing ? "分析中..." : "对比全部 NS"}
        </button>
      </div>

      <div class="summary-grid">
        <div class="summary-card">
          <div class="summary-num">${stats.totalRecords}</div>
          <div class="summary-lbl">元数据总记录数</div>
        </div>
        <div class="summary-card">
          <div class="summary-num">${stats.totalFiles}</div>
          <div class="summary-lbl">文件总数</div>
        </div>
        <div class="summary-card">
          <div class="summary-num">${stats.totalDirs}</div>
          <div class="summary-lbl">目录总数</div>
        </div>
        <div class="summary-card">
          <div class="summary-num">${stats.totalSize}</div>
          <div class="summary-lbl">存储容量总大小</div>
        </div>
        <div class="summary-card">
          <div class="summary-num">${stats.avgFileSize}</div>
          <div class="summary-lbl">平均文件大小</div>
        </div>
        <div class="summary-card">
          <div class="summary-num">${stats.maxDepth}</div>
          <div class="summary-lbl">最大目录树深度</div>
        </div>
      </div>

      <div class="warning-banners">
        <div class="warning-banner-card warn">
          <div style="font-size: 24px;">🗑️</div>
          <div>
            <div class="warning-title">已删除回收站 (Trash) 未清理</div>
            <div class="warning-desc">包含被标记删除但仍占用元数据内存的记录 ${stats.trashFiles} 条。</div>
          </div>
        </div>
        <div class="warning-banner-card critical">
          <div style="font-size: 24px;">⚠️</div>
          <div>
            <div class="warning-title">空闲的零字节文件风险</div>
            <div class="warning-desc">当前命名空间下累计存在零大小（0 Byte）无效文件 ${stats.zeroByteFiles} 个。</div>
          </div>
        </div>
      </div>

      <div class="sections-grid">
        <!-- 目录深度分布 -->
        <div class="section-card">
          ${this.renderSectionTitle("HDFS 目录深度分布 (Directory Depth)", () =>
            this.askAiForChart(
              "directory-depth",
              "HDFS 目录深度分布",
              stats.depthData.map((d) => `目录深度 ${d.depth}: ${d.count.toLocaleString()} 个 (${d.percent}%)`),
              ["目录深度风险解释", "NameNode 元数据压力判断", "需要治理的深层目录", "清理/归档建议"],
            ))}
          <div class="bar-chart-list">
            ${stats.depthData.map((d) => html`
              <div class="bar-chart-item">
                <div class="bar-info-row">
                  <span>${d.depth}</span>
                  <span class="bar-value">${d.count.toLocaleString()} 个 (${d.percent}%)</span>
                </div>
                <div class="bar-track">
                  <div class="bar-fill" style="width: ${d.percent}%; background: linear-gradient(90deg, #10b981 0%, #34d399 100%)"></div>
                </div>
              </div>
            `)}
          </div>
        </div>

        <!-- 文件大小细分 -->
        <div class="section-card">
          ${this.renderSectionTitle("文件大小区间占比 (File Size Segments)", () =>
            this.askAiForChart(
              "file-size-segments",
              "文件大小区间占比",
              stats.sizeData.map((s) => `文件大小 ${s.size}: ${s.count.toLocaleString()} 个 (${s.percent}%)`),
              ["小文件风险判断", "空间利用效率", "治理优先级", "合并/归档策略"],
            ))}
          <div class="bar-chart-list">
            ${stats.sizeData.map((s) => html`
              <div class="bar-chart-item">
                <div class="bar-info-row">
                  <span>${s.size}</span>
                  <span class="bar-value">${s.count.toLocaleString()} 个 (${s.percent}%)</span>
                </div>
                <div class="bar-track">
                  <div class="bar-fill" style="width: ${s.percent}%"></div>
                </div>
              </div>
            `)}
          </div>
        </div>
      </div>

      <div class="sections-grid">
        <!-- 用户存储排行 -->
        <div class="section-card">
          ${this.renderSectionTitle("大租户/用户资源使用占比", () =>
            this.askAiForChart(
              "tenant-usage",
              "大租户/用户资源使用占比",
              stats.userData.map((u) => `用户 ${u.user}: 文件 ${u.files.toLocaleString()}, 占比 ${u.percent}%, 大小 ${u.size}`),
              ["Top 租户风险", "异常占用判断", "责任对象", "治理沟通建议"],
              "similar",
            ))}
          <table class="ops-table">
            <thead>
              <tr>
                <th>用户</th>
                <th>文件数目</th>
                <th>数目比例</th>
                <th>占用存储大小</th>
              </tr>
            </thead>
            <tbody>
              ${stats.userData.map((u) => html`
                <tr>
                  <td style="font-weight: 600; font-family: monospace;">${u.user}</td>
                  <td>${u.files.toLocaleString()}</td>
                  <td>${u.percent}%</td>
                  <td style="font-weight: 600;">${u.size}</td>
                </tr>
              `)}
            </tbody>
          </table>
        </div>

        <!-- 修改访问冷热周期 -->
        <div class="section-card">
          ${this.renderSectionTitle("冷热数据生命周期分析 (冷数据治理)", () =>
            this.askAiForChart(
              "cold-data-lifecycle",
              "冷热数据生命周期分析",
              stats.modifyData.slice(0, 5).flatMap((m, idx) => {
                const accessItem = stats.accessData[idx] || { period: m.period, count: 0, percent: 0 };
                return [
                  `修改周期 ${m.period}: ${m.count.toLocaleString()} (${m.percent}%)`,
                  `访问周期 ${accessItem.period ?? m.period}: ${accessItem.count?.toLocaleString?.() ?? 0} (${accessItem.percent}%)`,
                ];
              }),
              ["冷数据识别", "可归档/清理对象", "业务影响风险", "分阶段治理建议"],
              "action",
            ))}
          <div class="bar-chart-list">
            ${stats.modifyData.slice(0, 5).map((m, idx) => {
              const accessItem = stats.accessData[idx] || { percent: 0 };
              return html`
                <div class="bar-chart-item">
                  <div class="bar-info-row" style="margin-bottom: 2px;">
                    <span>数据修改周期 [${m.period}]</span>
                    <span class="bar-value">修改: ${m.percent}% | 访问: ${accessItem.percent}%</span>
                  </div>
                  <div style="display: flex; flex-direction: column; gap: 3px;">
                    <div class="bar-track" style="height: 4px;">
                      <div class="bar-fill" style="width: ${m.percent}%; background: #ef4444"></div>
                    </div>
                    <div class="bar-track" style="height: 4px;">
                      <div class="bar-fill" style="width: ${accessItem.percent}%; background: #3b82f6"></div>
                    </div>
                  </div>
                </div>
              `;
            })}
          </div>
        </div>
      </div>
    `;
  }
}
