import { LitElement, html, css } from "lit";
import { customElement, property, state } from "lit/decorators.js";
import { fetchBchClustersHealth, BchClusterHealth } from "../../controllers/bch-client.ts";
import { icons } from "../../icons.ts";

@customElement("bch-cluster-overview")
export class BchClusterOverview extends LitElement {
  @property({ type: Object }) host: any = null;

  @state() private clusters: BchClusterHealth[] = [];
  @state() private loading = false;
  @state() private error: string | null = null;

  static styles = css`
    :host {
      display: block;
      padding: 24px;
      font-family: var(--font-family, sans-serif);
      color: var(--text-primary);
      overflow-y: auto;
      height: 100%;
      box-sizing: border-box;
    }

    .overview-header {
      margin-bottom: 24px;
    }

    .overview-header h2 {
      margin: 0 0 6px 0;
      font-size: 18px;
      font-weight: 600;
    }

    .overview-header p {
      margin: 0;
      font-size: 13px;
      color: var(--text-muted);
    }

    .stats-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
      gap: 16px;
      margin-bottom: 24px;
    }

    .ops-card {
      background: var(--bg-content);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 20px;
      box-shadow: var(--shadow-sm, 0 4px 20px rgba(0, 0, 0, 0.08));
      transition: transform 0.2s, border-color 0.2s;
    }

    .ops-card:hover {
      border-color: var(--accent, #3b82f6);
      transform: translateY(-2px);
      box-shadow: var(--shadow-md, 0 6px 24px rgba(0, 0, 0, 0.12));
    }

    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      margin-bottom: 16px;
    }

    .cluster-name {
      font-size: 15px;
      font-weight: 600;
      color: var(--text-primary);
    }

    .cluster-region {
      font-size: 11px;
      color: var(--text-muted);
      background: var(--bg-hover, rgba(0, 0, 0, 0.03));
      padding: 2px 6px;
      border-radius: 4px;
      margin-top: 4px;
      display: inline-block;
    }

    .health-score-badge {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      width: 52px;
      height: 52px;
      border-radius: 50%;
      font-weight: 700;
      font-size: 18px;
    }

    .health-score-badge.healthy {
      background: rgba(16, 185, 129, 0.12);
      color: #10b981;
      border: 2px solid rgba(16, 185, 129, 0.3);
    }

    .health-score-badge.warning {
      background: rgba(245, 158, 11, 0.12);
      color: #f59e0b;
      border: 2px solid rgba(245, 158, 11, 0.3);
    }

    .health-score-badge.critical {
      background: rgba(239, 68, 68, 0.12);
      color: #ef4444;
      border: 2px solid rgba(239, 68, 68, 0.3);
    }

    .score-label {
      font-size: 9px;
      font-weight: 400;
      opacity: 0.8;
      margin-top: -2px;
    }

    .metrics-list {
      display: flex;
      flex-direction: column;
      gap: 12px;
    }

    .metric-item {
      font-size: 12px;
    }

    .metric-label-row {
      display: flex;
      justify-content: space-between;
      color: var(--text-secondary);
      margin-bottom: 4px;
    }

    .metric-value {
      font-weight: 600;
      color: var(--text-primary);
    }

    .progress-bar-bg {
      height: 6px;
      background: var(--bg, rgba(0, 0, 0, 0.05));
      border-radius: 3px;
      overflow: hidden;
    }

    .progress-bar-fill {
      height: 100%;
      border-radius: 3px;
    }

    .progress-bar-fill.healthy {
      background: #10b981;
    }

    .progress-bar-fill.warning {
      background: #f59e0b;
    }

    .progress-bar-fill.critical {
      background: #ef4444;
    }

    .card-footer-stats {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: 8px;
      margin-top: 16px;
      padding-top: 16px;
      border-top: 1px solid var(--border);
      text-align: center;
    }

    .stat-box-num {
      font-size: 14px;
      font-weight: 600;
      color: var(--text-primary);
    }

    .stat-box-lbl {
      font-size: 10px;
      color: var(--text-muted);
      margin-top: 2px;
    }

    .alert-banner {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 8px 12px;
      border-radius: 6px;
      font-size: 11px;
      margin-top: 12px;
    }

    .alert-banner.warning {
      background: rgba(245, 158, 11, 0.08);
      color: #f59e0b;
      border: 1px solid rgba(245, 158, 11, 0.15);
    }

    .alert-banner.healthy {
      background: rgba(16, 185, 129, 0.08);
      color: #10b981;
      border: 1px solid rgba(16, 185, 129, 0.15);
    }

    .loading-container {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      height: 200px;
      color: var(--text-muted);
      gap: 12px;
      font-size: 13px;
    }

    .spinner {
      width: 24px;
      height: 24px;
      border: 2px solid rgba(255, 255, 255, 0.1);
      border-top-color: var(--accent, #3b82f6);
      border-radius: 50%;
      animation: spin 0.8s linear infinite;
    }

    @keyframes spin {
      to {
        transform: rotate(360deg);
      }
    }
  `;

  connectedCallback() {
    super.connectedCallback();
    this.loadData();
  }

  async loadData() {
    if (!this.host) return;
    this.loading = true;
    this.error = null;
    try {
      this.clusters = await fetchBchClustersHealth(this.host);
    } catch (err: any) {
      this.error = err.message || String(err);
    } finally {
      this.loading = false;
    }
  }

  render() {
    if (this.loading) {
      return html`
        <div class="loading-container">
          <div class="spinner"></div>
          <div>正在加载集群健康数据...</div>
        </div>
      `;
    }

    if (this.error) {
      return html`
        <div style="padding: 16px; color: var(--ops-health-critical, #ef4444);">
          ${this.error}
        </div>
      `;
    }

    return html`
      <div class="overview-header">
        <h2>集群健康概览</h2>
        <p>汇聚 BCH 大数据生态集群的核心负载、健康评分与实时容量矩阵。</p>
      </div>

      <div class="stats-grid">
        ${this.clusters.map((c) => {
          const scoreClass = c.score >= 90 ? "healthy" : c.score >= 70 ? "warning" : "critical";
          const alertClass = c.activeAlerts > 0 ? "warning" : "healthy";
          return html`
            <div class="ops-card">
              <div class="card-header">
                <div>
                  <div class="cluster-name">${c.name}</div>
                  <div class="cluster-region">${c.region}区域</div>
                </div>
                <div class="health-score-badge ${scoreClass}">
                  <div>${c.score}</div>
                  <div class="score-label">健康度</div>
                </div>
              </div>

              <div class="metrics-list">
                <div class="metric-item">
                  <div class="metric-label-row">
                    <span>CPU 使用率</span>
                    <span class="metric-value">${c.cpuUsedPercent}%</span>
                  </div>
                  <div class="progress-bar-bg">
                    <div
                      class="progress-bar-fill ${c.cpuUsedPercent > 85 ? "critical" : c.cpuUsedPercent > 70 ? "warning" : "healthy"}"
                      style="width: ${c.cpuUsedPercent}%"
                    ></div>
                  </div>
                </div>

                <div class="metric-item">
                  <div class="metric-label-row">
                    <span>内存使用率</span>
                    <span class="metric-value">${c.memUsedPercent}%</span>
                  </div>
                  <div class="progress-bar-bg">
                    <div
                      class="progress-bar-fill ${c.memUsedPercent > 90 ? "critical" : c.memUsedPercent > 75 ? "warning" : "healthy"}"
                      style="width: ${c.memUsedPercent}%"
                    ></div>
                  </div>
                </div>

                <div class="metric-item">
                  <div class="metric-label-row">
                    <span>HDFS 存储容量</span>
                    <span class="metric-value">${c.dfsUsedPercent}%</span>
                  </div>
                  <div class="progress-bar-bg">
                    <div
                      class="progress-bar-fill ${c.dfsUsedPercent > 80 ? "critical" : c.dfsUsedPercent > 60 ? "warning" : "healthy"}"
                      style="width: ${c.dfsUsedPercent}%"
                    ></div>
                  </div>
                </div>
              </div>

              <div class="card-footer-stats">
                <div>
                  <div class="stat-box-num">${c.nodeCount}</div>
                  <div class="stat-box-lbl">物理节点</div>
                </div>
                <div>
                  <div class="stat-box-num">${c.metrics.activeContainers || 0}</div>
                  <div class="stat-box-lbl">运行容器</div>
                </div>
                <div>
                  <div class="stat-box-num">${c.metrics.activeNodes || 0} / ${c.nodeCount}</div>
                  <div class="stat-box-lbl">在线节点</div>
                </div>
              </div>

              <div class="alert-banner ${alertClass}">
                <span style="font-size: 14px;">${c.activeAlerts > 0 ? "⚠️" : "🛡️"}</span>
                <span>
                  ${c.activeAlerts > 0 ? `当前活动告警: ${c.activeAlerts} 条` : "当前集群无活动告警，处于稳定状态"}
                </span>
              </div>
            </div>
          `;
        })}
      </div>
    `;
  }
}
