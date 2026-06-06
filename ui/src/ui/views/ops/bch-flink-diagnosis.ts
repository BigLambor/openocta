import { LitElement, html, css, nothing, type TemplateResult } from "lit";
import { customElement, property, state } from "lit/decorators.js";
import { unsafeHTML } from "lit/directives/unsafe-html.js";
import {
  fetchBchClustersHealth,
  fetchBchFlinkJobs,
  fetchBchFlinkJobConfig,
  diagnoseBchFlinkJob,
  fetchBchSparkJobs,
  tuneBchSparkJob,
  BchClusterHealth,
  FlinkJob,
  SparkJob,
  chatBchFlinkJob,
} from "../../controllers/bch-client.ts";
import { icons } from "../../icons.ts";
import { toSanitizedMarkdownHtml } from "../../markdown.ts";
import {
  averageRadarFromFlinkJobs,
  bucketJobsByScore,
  renderBchJobHealthOverview,
} from "./bch-job-health-overview.ts";

@customElement("bch-flink-diagnosis")
export class BchFlinkDiagnosis extends LitElement {
  @property({ type: Object }) host: any = null;
  @property({ type: String }) selectedCluster = "all";
  @property({ type: String }) timeRange = "24h";

  
  @state() private flinkJobs: FlinkJob[] = [];
  @state() private sparkJobs: SparkJob[] = [];
  @state() private clusters: BchClusterHealth[] = [];
  @state() private loading = false;
  @state() private error: string | null = null;

  // Flink Config Modal State
  @state() private configModalOpen = false;
  @state() private configLoading = false;
  @state() private configContent = "";
  @state() private configTargetJobName = "";

  // Flink Diagnosis Modal State
  @state() private diagnoseModalOpen = false;
  @state() private selectedFlinkJob: FlinkJob | null = null;
  @state() private showTrendCopilot = false;
  @state() private showBarCopilot = false;

  // Flink Copilot Chat Q&A State
  @state() private copilotInput = "";
  @state() private copilotMessages: Array<{ sender: "user" | "ai"; text: string }> = [];

  // Spark Detail Modal State
  @state() private sparkModalOpen = false;
  @state() private selectedSparkJob: SparkJob | null = null;

  static styles = css`
    :host {
      display: flex;
      flex-direction: column;
      font-family: var(--font-family, sans-serif);
      color: var(--text-primary);
      height: 100%;
      box-sizing: border-box;
    }

    .sub-nav {
      display: flex;
      border-bottom: 1px solid var(--border);
      background: var(--bg-content);
      padding: 0 24px;
    }

    .sub-nav-btn {
      background: transparent;
      border: none;
      color: var(--text-muted);
      padding: 12px 16px;
      font-size: 13px;
      cursor: pointer;
      font-weight: 500;
      border-bottom: 2px solid transparent;
      transition: all 0.2s;
    }

    .sub-nav-btn:hover {
      color: var(--text-primary);
    }

    .sub-nav-btn.active {
      color: var(--accent, #3b82f6);
      border-bottom-color: var(--accent, #3b82f6);
    }

    .governance-content {
      flex: 1;
      padding: 24px;
      overflow-y: auto;
    }

    .sec-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 20px;
    }

    .sec-header h2 {
      margin: 0;
      font-size: 16px;
      font-weight: 600;
    }

    .cluster-filter {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      font-size: 12px;
      color: var(--text-muted);
    }

    .cluster-filter select {
      min-width: 160px;
      padding: 6px 10px;
      border: 1px solid var(--border);
      border-radius: 6px;
      background: var(--bg-content);
      color: var(--text-primary);
      font-size: 12px;
    }

    /* Overall Summary Cards */
    .flink-summary-grid {
      display: grid;
      grid-template-columns: repeat(4, 1fr);
      gap: 16px;
      margin-bottom: 20px;
    }

    .summary-card {
      background: var(--bg-content);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 16px;
      box-shadow: var(--shadow-sm, 0 2px 10px rgba(0, 0, 0, 0.05));
    }

    .summary-lbl {
      font-size: 12px;
      color: var(--text-muted);
      margin-bottom: 4px;
    }

    .summary-val {
      font-size: 22px;
      font-weight: 700;
    }

    .summary-val.healthy { color: #10b981; }
    .summary-val.warning { color: #f59e0b; }
    .summary-val.critical { color: #ef4444; }
    .summary-val.info { color: #3b82f6; }

    /* Table Styles */
    .ops-table-container {
      background: var(--bg-content);
      border: 1px solid var(--border);
      border-radius: 12px;
      overflow: hidden;
      box-shadow: var(--shadow-sm, 0 4px 20px rgba(0, 0, 0, 0.05));
    }

    .ops-table {
      width: 100%;
      border-collapse: collapse;
      font-size: 13px;
      text-align: left;
    }

    .ops-table th {
      background: var(--bg-hover, rgba(0, 0, 0, 0.02));
      padding: 12px 16px;
      font-weight: 600;
      color: var(--text-secondary);
      border-bottom: 1px solid var(--border);
      text-transform: uppercase;
      font-size: 11px;
      letter-spacing: 0.5px;
    }

    .ops-table td {
      padding: 14px 16px;
      border-bottom: 1px solid var(--border);
      color: var(--text-primary);
    }

    .ops-table tr:hover {
      background: var(--bg-hover, rgba(0, 0, 0, 0.02));
    }

    .score-btn {
      background: transparent;
      border: none;
      font-weight: 700;
      padding: 4px 12px;
      border-radius: 12px;
      cursor: pointer;
      transition: transform 0.1s;
      font-size: 12px;
      box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    }

    .score-btn:hover {
      transform: scale(1.08);
    }

    .score-btn.healthy { background: rgba(16, 185, 129, 0.15); color: #10b981; border: 1px solid rgba(16, 185, 129, 0.3); }
    .score-btn.warning { background: rgba(245, 158, 11, 0.15); color: #f59e0b; border: 1px solid rgba(245, 158, 11, 0.3); }
    .score-btn.critical { background: rgba(239, 68, 68, 0.15); color: #ef4444; border: 1px solid rgba(239, 68, 68, 0.3); }

    .action-icon-btn {
      background: transparent;
      border: 1px solid var(--border);
      color: var(--text-muted);
      border-radius: 6px;
      padding: 4px 10px;
      cursor: pointer;
      font-size: 14px;
      transition: all 0.2s;
    }

    .action-icon-btn:hover {
      color: var(--accent, #3b82f6);
      border-color: var(--accent, #3b82f6);
      background: rgba(59, 130, 246, 0.05);
    }

    .diagnose-btn {
      background: rgba(59, 130, 246, 0.1);
      border: 1px solid rgba(59, 130, 246, 0.2);
      color: var(--accent, #3b82f6);
      padding: 6px 12px;
      border-radius: 6px;
      cursor: pointer;
      font-size: 11px;
      transition: all 0.2s;
    }

    .diagnose-btn:hover {
      background: var(--accent, #3b82f6);
      color: #fff;
    }

    .config-btn {
      background: rgba(16, 185, 129, 0.08);
      border: 1px solid rgba(16, 185, 129, 0.2);
      color: #10b981;
      padding: 6px 12px;
      border-radius: 6px;
      cursor: pointer;
      font-size: 11px;
      transition: all 0.2s;
      font-weight: 500;
    }

    .config-btn:hover {
      background: #10b981;
      color: #fff;
    }

    .tag-badge {
      background: rgba(255, 255, 255, 0.05);
      border: 1px solid rgba(255, 255, 255, 0.1);
      color: var(--text-secondary);
      border-radius: 4px;
      padding: 2px 8px;
      font-size: 11px;
      margin-right: 6px;
      display: inline-block;
    }

    .tag-badge.danger {
      background: rgba(239, 68, 68, 0.08);
      border-color: rgba(239, 68, 68, 0.15);
      color: #ef4444;
    }

    .tag-badge.warning {
      background: rgba(245, 158, 11, 0.08);
      border-color: rgba(245, 158, 11, 0.15);
      color: #f59e0b;
    }

    /* Modal Backdrop */
    .modal-backdrop {
      position: fixed;
      inset: 0;
      background: rgba(0, 0, 0, 0.6);
      backdrop-filter: blur(4px);
      z-index: 100;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 24px;
    }

    .modal-container {
      background: var(--bg);
      border: 1px solid var(--border);
      border-radius: 12px;
      width: 100%;
      max-width: 960px;
      max-height: 85vh;
      overflow-y: auto;
      box-shadow: var(--shadow-xl);
      display: flex;
      flex-direction: column;
    }

    .modal-header {
      padding: 16px 24px;
      border-bottom: 1px solid var(--border);
      display: flex;
      justify-content: space-between;
      align-items: center;
      background: var(--bg-hover, rgba(0, 0, 0, 0.02));
    }

    .modal-title h3 {
      margin: 0;
      font-size: 16px;
      font-weight: 600;
    }

    .modal-title p {
      margin: 4px 0 0 0;
      font-size: 11px;
      color: var(--text-muted);
    }

    .close-btn {
      background: transparent;
      border: none;
      color: var(--text-muted);
      font-size: 20px;
      cursor: pointer;
      padding: 4px;
    }

    .close-btn:hover {
      color: var(--text-primary);
    }

    .modal-body {
      padding: 24px;
      overflow-y: auto;
      flex: 1;
    }

    /* Flink Diagnosis Layout */
    .diag-summary-panel {
      background: var(--bg-hover, rgba(0, 0, 0, 0.02));
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 16px;
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 24px;
    }

    .diag-score-group {
      display: flex;
      align-items: baseline;
      gap: 12px;
    }

    .diag-score {
      font-size: 40px;
      font-weight: 800;
      line-height: 1;
    }

    .diag-badge {
      padding: 4px 8px;
      border-radius: 4px;
      font-size: 10px;
      font-weight: 700;
    }

    .diag-badge.healthy { background: rgba(16, 185, 129, 0.15); color: #10b981; }
    .diag-badge.warning { background: rgba(245, 158, 11, 0.15); color: #f59e0b; }
    .diag-badge.critical { background: rgba(239, 68, 68, 0.15); color: #ef4444; }

    .penalty-track {
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      gap: 6px;
      margin-top: 8px;
      font-size: 11px;
    }

    .penalty-pill {
      background: var(--bg);
      border: 1px solid var(--border);
      padding: 2px 8px;
      border-radius: 4px;
      font-family: monospace;
    }

    .penalty-pill.fatal { color: #ef4444; border-color: rgba(239, 68, 68, 0.2); }
    .penalty-pill.warning { color: #f59e0b; border-color: rgba(245, 158, 11, 0.2); }
    .penalty-pill.info { color: #3b82f6; border-color: rgba(59, 130, 246, 0.2); }

    .diag-grid-split {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 24px;
    }

    .diag-sub-header {
      font-size: 13px;
      font-weight: 600;
      border-bottom: 1px solid var(--border);
      padding-bottom: 8px;
      margin-bottom: 16px;
      color: var(--text-secondary);
    }

    .chart-container-box {
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 12px;
      background: var(--bg);
      margin-bottom: 16px;
      position: relative;
    }

    .copilot-btn {
      position: absolute;
      bottom: 8px;
      right: 8px;
      background: var(--bg-content);
      border: 1px solid var(--border);
      color: var(--text-secondary);
      border-radius: 12px;
      padding: 3px 8px;
      font-size: 10px;
      cursor: pointer;
      z-index: 5;
      box-shadow: 0 2px 6px rgba(0,0,0,0.1);
    }

    .copilot-btn:hover {
      border-color: var(--accent, #3b82f6);
      color: var(--text-primary);
    }

    .copilot-popover {
      position: absolute;
      bottom: 34px;
      right: 8px;
      background: var(--bg-content);
      border: 1px solid var(--border);
      border-radius: 6px;
      padding: 10px;
      font-size: 11px;
      max-width: 220px;
      z-index: 20;
      color: var(--text-primary);
      box-shadow: 0 4px 12px rgba(0,0,0,0.15);
    }

    .copilot-popover strong {
      color: var(--accent, #3b82f6);
      display: block;
      margin-bottom: 4px;
    }

    /* CoT Steps Timeline */
    .timeline-steps {
      display: flex;
      flex-direction: column;
      gap: 16px;
      background: var(--bg);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 16px;
    }

    .chain-step {
      border-left: 2px solid var(--border);
      padding-left: 14px;
      position: relative;
    }

    .chain-step::before {
      content: "";
      position: absolute;
      left: -5px;
      top: 2px;
      width: 8px;
      height: 8px;
      border-radius: 50%;
      background: rgba(255,255,255,0.1);
      border: 1px solid var(--border);
    }

    .chain-step.active::before { background: #3b82f6; }
    .chain-step.warning::before { background: #f59e0b; }
    .chain-step.critical::before { background: #ef4444; }

    .chain-step.active { border-left-color: rgba(59, 130, 246, 0.4); }
    .chain-step.warning { border-left-color: rgba(245, 158, 11, 0.4); }
    .chain-step.critical { border-left-color: rgba(239, 68, 68, 0.4); }

    .step-title {
      font-size: 12px;
      font-weight: 600;
      margin-bottom: 2px;
    }

    .step-desc {
      font-size: 11px;
      color: var(--text-muted);
    }

    .action-list {
      margin-top: 16px;
      padding: 12px 16px;
      background: rgba(59, 130, 246, 0.05);
      border: 1px solid rgba(59, 130, 246, 0.15);
      border-radius: 6px;
    }

    .action-list h5 {
      margin: 0 0 8px 0;
      font-size: 12px;
      color: var(--accent, #3b82f6);
    }

    .action-list ul {
      margin: 0;
      padding-left: 16px;
      font-size: 11px;
      color: var(--text-secondary);
      line-height: 1.6;
    }

    /* Copilot Chat Layout */
    .copilot-chat-layout {
      display: flex;
      border: 1px solid var(--border);
      background: var(--bg-content);
      border-radius: 8px;
      margin-top: 24px;
      height: 280px;
      overflow: hidden;
    }

    .copilot-profile-pane {
      width: 240px;
      border-right: 1px solid var(--border);
      background: var(--bg-hover, rgba(0, 0, 0, 0.01));
      padding: 16px;
      display: flex;
      flex-direction: column;
      gap: 12px;
      flex-shrink: 0;
    }

    .copilot-profile-header {
      display: flex;
      align-items: center;
      gap: 8px;
    }

    .copilot-avatar {
      font-size: 24px;
    }

    .copilot-profile-name {
      font-size: 13px;
      font-weight: 600;
      color: var(--text-primary);
    }

    .copilot-status-dot {
      display: inline-block;
      width: 6px;
      height: 6px;
      border-radius: 50%;
      background: #10b981;
      margin-right: 4px;
    }

    .copilot-status-text {
      font-size: 11px;
      color: var(--text-secondary);
    }

    .copilot-skills-list {
      margin: 0;
      padding-left: 14px;
      font-size: 11px;
      color: var(--text-secondary);
      display: flex;
      flex-direction: column;
      gap: 4px;
    }

    .copilot-chat-pane {
      flex: 1;
      display: flex;
      flex-direction: column;
      background: var(--bg);
    }

    .copilot-chat-messages {
      flex: 1;
      overflow-y: auto;
      padding: 16px;
      display: flex;
      flex-direction: column;
      gap: 10px;
      font-size: 12px;
    }

    .chat-bubble {
      max-width: 85%;
      padding: 10px 14px;
      border-radius: 8px;
      line-height: 1.6;
      font-size: 12px;
      box-sizing: border-box;
      word-wrap: break-word;
      word-break: break-word;
    }

    .chat-bubble.user {
      align-self: flex-end;
      background: var(--accent);
      color: white;
    }

    .chat-bubble.ai {
      align-self: flex-start;
      background: var(--bg-hover, rgba(0, 0, 0, 0.02));
      border: 1px solid var(--border);
      color: var(--text-primary);
    }

    /* Style markdown elements in AI response */
    .chat-bubble.ai p {
      margin: 0 0 8px 0;
    }
    .chat-bubble.ai p:last-child {
      margin-bottom: 0;
    }
    .chat-bubble.ai strong {
      color: var(--text-primary);
      font-weight: 600;
    }
    .chat-bubble.ai code {
      font-family: var(--mono, monospace);
      background: rgba(127, 127, 127, 0.1);
      padding: 2px 4px;
      border-radius: 4px;
      font-size: 11px;
      color: var(--accent);
    }
    .chat-bubble.ai pre {
      background: var(--bg-content, #1e1e1e);
      border: 1px solid var(--border);
      border-radius: 6px;
      padding: 10px;
      margin: 8px 0;
      overflow-x: auto;
    }
    .chat-bubble.ai pre code {
      background: transparent;
      padding: 0;
      border-radius: 0;
      color: var(--text-primary);
      font-size: 11px;
      display: block;
    }
    .chat-bubble.ai ul, .chat-bubble.ai ol {
      margin: 0 0 8px 0;
      padding-left: 20px;
    }
    .chat-bubble.ai li {
      margin-bottom: 4px;
    }
    .chat-bubble.ai h1, .chat-bubble.ai h2, .chat-bubble.ai h3, .chat-bubble.ai h4 {
      margin: 12px 0 6px 0;
      font-size: 13px;
      font-weight: 600;
      color: var(--text-primary);
    }
    .chat-bubble.ai h1:first-child, .chat-bubble.ai h2:first-child, .chat-bubble.ai h3:first-child {
      margin-top: 0;
    }

    .copilot-input-area {
      display: flex;
      align-items: flex-end;
      border-top: 1px solid var(--border);
      background: var(--bg-content);
      padding: 10px 16px;
      gap: 12px;
    }

    .copilot-textarea {
      flex: 1;
      background: var(--bg);
      border: 1px solid var(--border);
      border-radius: 6px;
      padding: 8px 12px;
      color: var(--text-primary);
      font-size: 12px;
      resize: none;
      height: 40px;
      line-height: 1.5;
      box-sizing: border-box;
      font-family: inherit;
    }

    .copilot-textarea:focus {
      outline: none;
      border-color: var(--accent);
    }

    .copilot-send-button {
      background: var(--accent);
      color: var(--accent-foreground, #fff);
      border: none;
      padding: 8px 16px;
      border-radius: 6px;
      cursor: pointer;
      font-size: 12px;
      font-weight: 500;
      transition: opacity 0.15s;
      height: 40px;
      display: flex;
      align-items: center;
      justify-content: center;
    }

    .copilot-send-button:hover {
      opacity: 0.9;
    }

    /* Configuration Modal Specific */
    .config-pre {
      background: var(--bg-content);
      color: var(--text-primary);
      font-family: var(--mono);
      padding: 16px;
      border-radius: 8px;
      overflow: auto;
      font-size: 12px;
      margin: 0;
      border: 1px solid var(--border);
    }

    .config-loader {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 48px;
      color: var(--text-muted);
      gap: 12px;
      font-size: 12px;
    }

    /* Spark Grid Details */
    .spark-details-grid {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: 12px;
      margin-bottom: 20px;
    }

    .spark-detail-item {
      background: var(--bg);
      border: 1px solid var(--border);
      border-radius: 6px;
      padding: 12px;
    }

    .spark-lbl {
      font-size: 11px;
      color: var(--text-muted);
      margin-bottom: 4px;
    }

    .spark-val {
      font-size: 15px;
      font-weight: 600;
    }

    .tuning-advice-box {
      background: rgba(59, 130, 246, 0.04);
      border: 1px solid rgba(59, 130, 246, 0.15);
      border-radius: 8px;
      padding: 16px;
    }

    .tuning-advice-box h4 {
      margin: 0 0 10px 0;
      color: #93c5fd;
      font-size: 13px;
    }

    .tuning-advice-box p {
      margin: 0 0 12px 0;
      font-size: 12px;
      line-height: 1.6;
      color: var(--text-secondary);
    }

    .tuning-advice-box pre {
      background: var(--bg-content);
      padding: 10px 14px;
      border-radius: 6px;
      font-family: var(--mono);
      font-size: 11px;
      color: var(--text-primary);
      overflow-x: auto;
      margin: 0;
      border: 1px solid var(--border);
    }
  `;

  

  connectedCallback() {
    super.connectedCallback();
    if (this.initialSubTab) this.subTab = this.initialSubTab;
    this.loadData();
  }

  async loadData() {
    if (!this.host) return;
    this.loading = true;
    this.error = null;
    try {
      const [clusters, jobs] = await Promise.all([
        fetchBchClustersHealth(this.host).catch(() => [] as BchClusterHealth[]),
        fetchBchFlinkJobs(this.host),
      ]);
      this.clusters = clusters;
      this.flinkJobs = jobs;
    } catch (err: any) {
      this.error = err.message || String(err);
    } finally {
      this.loading = false;
    }
  }

  private clusterOptions(): Array<{ value: string; label: string }> {
    const names = new Set<string>();
    for (const cluster of this.clusters) {
      if (cluster.name) names.add(cluster.name);
    }
    for (const job of this.flinkJobs) {
      if (job.cluster) names.add(job.cluster);
    }
    return [
      { value: "all", label: "全部集群" },
      ...Array.from(names)
        .sort()
        .map((name) => ({ value: name, label: name })),
    ];
  }

  private filteredFlinkJobs(): FlinkJob[] {
    if (this.selectedCluster === "all") {
      return this.flinkJobs;
    }
    return this.flinkJobs.filter((job) => job.cluster === this.selectedCluster);
  }

  private renderClusterFilter() {
    const options = this.clusterOptions();
    if (options.length <= 1) {
      return nothing;
    }
    return html`
      <label class="cluster-filter">
        <span>目标集群</span>
        <select
          .value=${this.selectedCluster}
          @change=${(e: Event) => {
            this.selectedCluster = (e.target as HTMLSelectElement).value;
          }}
        >
          ${options.map(
            (opt) => html`<option value=${opt.value} ?selected=${opt.value === this.selectedCluster}>${opt.label}</option>`,
          )}
        </select>
      </label>
    `;
  }

  async openConfigModal(job: FlinkJob) {
    this.configTargetJobName = job.name;
    this.configModalOpen = true;
    this.configLoading = true;
    this.configContent = "";
    try {
      const configObj = await fetchBchFlinkJobConfig(this.host, job.id);
      this.configContent = JSON.stringify(configObj, null, 2);
    } catch (err: any) {
      this.configContent = `Error: ${err.message || String(err)}`;
    } finally {
      this.configLoading = false;
    }
  }

  async openDiagnoseModal(job: FlinkJob) {
    this.selectedFlinkJob = job;
    this.diagnoseModalOpen = true;
    this.copilotMessages = [
      { sender: "ai", text: `您好！我是 BCH 作业诊断助手。我已经对作业「${job.name}」完成了智能诊断，您可以随时向我提问关于该作业的瓶颈、反压或扩容建议。` }
    ];
    try {
      this.selectedFlinkJob = await diagnoseBchFlinkJob(this.host, job.id);
    } catch (err) {
      console.error(err);
    }
  }

  async sendCopilotMessage() {
    const text = this.copilotInput.trim();
    if (!text || !this.selectedFlinkJob) return;

    this.copilotMessages = [...this.copilotMessages, { sender: "user", text }];
    this.copilotInput = "";

    // Show a typing/loading indicator from AI
    const tempAiMessage = { sender: "ai" as const, text: "正在思考中..." };
    this.copilotMessages = [...this.copilotMessages, tempAiMessage];

    try {
      const reply = await chatBchFlinkJob(this.host, this.selectedFlinkJob.id, text);
      // Replace the loading message with the real reply
      this.copilotMessages = [
        ...this.copilotMessages.slice(0, -1),
        { sender: "ai", text: reply },
      ];
    } catch (err: any) {
      // Replace the loading message with the error message
      this.copilotMessages = [
        ...this.copilotMessages.slice(0, -1),
        { sender: "ai", text: `出错了: ${err.message || String(err)}` },
      ];
    }
  }

  openSparkTuningModal(job: SparkJob) {
    this.selectedSparkJob = job;
    this.sparkModalOpen = true;
  }

  updated(changedProperties: Map<PropertyKey, unknown>) {
    if (changedProperties.has("copilotMessages")) {
      const messagesContainer = this.renderRoot.querySelector(".copilot-chat-messages");
      if (messagesContainer) {
        messagesContainer.scrollTop = messagesContainer.scrollHeight;
      }
    }
  }

  switchSubTab(tab: "flink" | "spark") {
    if (this.subTab === tab) return;
    this.subTab = tab;
    this.loadData();
  }

  render() {
    return html`
      

      <div class="governance-content">
        ${this.loading
          ? html`
              <div class="loading-container" style="display: flex; flex-direction: column; align-items: center; justify-content: center; height: 200px; color: var(--text-muted);">
                <div class="spinner" style="width: 24px; height: 24px; border: 2px solid rgba(255, 255, 255, 0.1); border-top-color: var(--accent, #3b82f6); border-radius: 50%; animation: spin 0.8s linear infinite; margin-bottom: 12px;"></div>
                <div>正在加载作业治理数据...</div>
              </div>
            `
          : this.error
            ? html`<div class="empty-placeholder" style="padding: 24px; color: var(--danger, #d33);">${this.error}</div>`
            : this.renderFlinkContent()}
      </div>

      ${this.renderConfigModal()}
      ${this.renderDiagnoseModal()}
      ${this.renderSparkModal()}
    `;
  }

  private renderFlinkContent() {
    const jobs = this.filteredFlinkJobs();
    const clusterLabel = this.selectedCluster === "all" ? "全部集群" : this.selectedCluster;
    const buckets = bucketJobsByScore(
      jobs.map((j) => j.score),
      jobs.map((j) => j.rootCause === "S7"),
    );
    const radar = averageRadarFromFlinkJobs(jobs);
    const backpressureCount = jobs.filter((j) => j.metrics?.isBP).length;

    return html`
      ${renderBchJobHealthOverview({
        title: "全局健康概览",
        jobKind: "Flink",
        clusterLabel,
        buckets,
        radar,
        agentSummary: html`当前环境共接入 <strong>${buckets.total}</strong> 个 Flink 作业。Flink Doctor Agent 正在执行 5 分钟级滑动窗口巡检。通过 <span class="bch-overview__summary-link">三角验证逻辑</span>，本周期共排查出 <strong>${backpressureCount}</strong> 起背压事件与 <strong>${buckets.waste}</strong> 起资源浪费候选。`,
        stabilityBaseline: radar.stability >= 80 ? "normal" : "warning",
        performanceBaseline: radar.performance >= 75 ? "warning" : "critical",
      })}

      <div class="sec-header">
        <h2>在线流计算作业巡检 (Flink Doctor)</h2>
        ${this.renderClusterFilter()}
      </div>

      <div class="ops-table-container">
        <table class="ops-table">
          <thead>
            <tr>
              <th>作业名称</th>
              <th>Owner</th>
              <th>所属集群</th>
              <th style="text-align: center;">健康评分 (点击问诊)</th>
              <th style="text-align: center;">运行配置</th>
              <th>初步诊断</th>
            </tr>
          </thead>
          <tbody>
            ${jobs.length === 0
              ? html`<tr><td colspan="6" style="text-align:center; color: var(--text-muted); padding: 24px;">当前集群暂无 Flink 作业。</td></tr>`
              : nothing}
            ${jobs.map((job) => {
              const scoreClass = job.score >= 90 ? "healthy" : job.score >= 60 ? "warning" : "critical";
              return html`
                <tr>
                  <td style="font-weight: 600;">${job.name}</td>
                  <td style="font-family: monospace; font-size: 11px;">${job.owner}</td>
                  <td><span class="tag-badge">${job.cluster}</span></td>
                  <td style="text-align: center;">
                    <button class="score-btn ${scoreClass}" @click=${() => this.openDiagnoseModal(job)}>
                      ${job.score}分
                    </button>
                  </td>
                  <td style="text-align: center;">
                    <button class="config-btn" @click=${() => this.openConfigModal(job)}>
                      查看配置
                    </button>
                  </td>
                  <td style="font-size: 11px; color: var(--text-secondary);">
                    <strong>${job.rootCauseText}</strong> - ${job.diagnosis}
                  </td>
                </tr>
              `;
            })}
          </tbody>
        </table>
      </div>
    `;
  }



  private renderConfigModal() {
    if (!this.configModalOpen) return nothing;
    return html`
      <div class="modal-backdrop">
        <div class="modal-container" style="max-width: 640px;">
          <div class="modal-header">
            <div class="modal-title">
              <h3>运行配置提取</h3>
              <p>数据源: YARN ResourceManager API (${this.configTargetJobName})</p>
            </div>
            <button class="close-btn" @click=${() => (this.configModalOpen = false)}>&times;</button>
          </div>
          <div class="modal-body" style="padding: 16px;">
            ${this.configLoading
              ? html`
                  <div class="config-loader">
                    <div class="spinner"></div>
                    <span style="animation: pulse 1.5s infinite;">正在向集群 API 发送提取请求并校验凭证...</span>
                  </div>
                `
              : html`<pre class="config-pre"><code>${this.configContent}</code></pre>`}
          </div>
        </div>
      </div>
    `;
  }

  private renderDiagnoseModal() {
    if (!this.diagnoseModalOpen || !this.selectedFlinkJob) return nothing;
    const job = this.selectedFlinkJob;
    const scoreClass = job.score >= 90 ? "healthy" : job.score >= 60 ? "warning" : "critical";

    // SVG Axis Center points
    const sScoreNorm = job.sScore / 100;
    const pScoreNorm = job.pScore / 100;
    const eScoreNorm = job.eScore / 100;

    // Radar coordinates calculations
    const cx = 120, cy = 90, r = 60;
    const sX = cx, sY = cy - r * sScoreNorm;
    const pX = cx + r * Math.sin((120 * Math.PI) / 180) * pScoreNorm;
    const pY = cy - r * Math.cos((120 * Math.PI) / 180) * pScoreNorm;
    const eX = cx + r * Math.sin((240 * Math.PI) / 180) * eScoreNorm;
    const eY = cy - r * Math.cos((240 * Math.PI) / 180) * eScoreNorm;

    return html`
      <div class="modal-backdrop">
        <div class="modal-container">
          <div class="modal-header">
            <div class="modal-title">
              <h3>🤖 AI 深度诊断报告</h3>
              <p>${job.name}</p>
            </div>
            <button class="close-btn" @click=${() => {
              this.diagnoseModalOpen = false;
              this.showTrendCopilot = false;
              this.showBarCopilot = false;
            }}>&times;</button>
          </div>
          <div class="modal-body">
            <div class="diag-summary-panel" style="display: flex; flex-direction: column; align-items: stretch; gap: 12px;">
              <div style="display: flex; justify-content: space-between; align-items: center;">
                <div>
                  <div style="font-size: 10px; font-weight: bold; color: var(--text-muted); text-transform: uppercase;">综合健康评分</div>
                  <div class="diag-score-group">
                    <span class="diag-score ${scoreClass}">${job.score}</span>
                    <span class="diag-badge ${scoreClass}">${scoreClass.toUpperCase()}</span>
                  </div>
                </div>
                <div style="text-align: right;">
                  <div style="font-size: 10px; font-weight: bold; color: var(--text-muted); text-transform: uppercase;">诊断定位</div>
                  <div style="font-size: 14px; font-weight: 700; color: var(--text-primary); margin-top: 4px;">
                    ${job.rootCauseText} (${job.rootCause})
                  </div>
                </div>
              </div>
              
              <div style="border-top: 1px solid var(--border); padding-top: 12px;">
                <div style="font-size: 11px; font-weight: bold; color: var(--text-secondary); margin-bottom: 8px;">健康度打分模型 (扣分明细)</div>
                <div class="penalty-track">
                  ${job.penalties && job.penalties.length > 0
                    ? html`
                        <span class="penalty-pill" style="background: rgba(16, 185, 129, 0.08); color: #10b981; border-color: rgba(16, 185, 129, 0.2);">100</span>
                        ${job.penalties.map(p => {
                          const penaltyClass = p.type === 'fatal' ? 'fatal' : (p.type === 'warning' ? 'warning' : 'info');
                          return html`
                            <span style="color: var(--text-placeholder); font-size: 10px;">➔</span>
                            <span class="penalty-pill ${penaltyClass}">${p.item} (-${p.deduction})</span>
                          `;
                        })}
                        <span style="color: var(--text-placeholder); font-size: 10px;">➔</span>
                        <span class="penalty-pill" style="background: var(--text-primary); color: var(--bg); font-weight: bold; border-color: var(--text-primary); font-size: 12px;">=${job.score}</span>
                      `
                    : html`<span class="penalty-pill" style="background: rgba(16, 185, 129, 0.08); color: #10b981; border-color: rgba(16, 185, 129, 0.2);">100 (满分，无扣分项)</span>`
                  }
                </div>
              </div>
            </div>

            <div class="diag-grid-split">
              <!-- Left Side: Data Evidence (Charts) -->
              <div>
                <div class="diag-sub-header">数据举证 (Metrics Context)</div>

                <!-- Radar Chart -->
                <div class="chart-container-box" style="display: flex; justify-content: center; height: 180px; align-items: center;">
                  <svg width="240" height="185">
                    <!-- Concentric Grid Triangles -->
                    ${[0.2, 0.4, 0.6, 0.8, 1].map((pct) => {
                      const gr = r * pct;
                      const x1 = cx, y1 = cy - gr;
                      const x2 = cx + gr * Math.sin((120 * Math.PI) / 180), y2 = cy - gr * Math.cos((120 * Math.PI) / 180);
                      const x3 = cx + gr * Math.sin((240 * Math.PI) / 180), y3 = cy - gr * Math.cos((240 * Math.PI) / 180);
                      return html`
                        <polygon points="${x1},${y1} ${x2},${y2} ${x3},${y3}" fill="none" stroke="var(--border)" stroke-opacity="0.6" />
                      `;
                    })}
                    <!-- Axes lines -->
                    <line x1="${cx}" y1="${cy}" x2="${cx}" y2="${cy - r}" stroke="var(--border)" stroke-opacity="0.8" />
                    <line x1="${cx}" y1="${cy}" x2="${cx + r * Math.sin((120 * Math.PI) / 180)}" y2="${cy - r * Math.cos((120 * Math.PI) / 180)}" stroke="var(--border)" stroke-opacity="0.8" />
                    <line x1="${cx}" y1="${cy}" x2="${cx + r * Math.sin((240 * Math.PI) / 180)}" y2="${cy - r * Math.cos((240 * Math.PI) / 180)}" stroke="var(--border)" stroke-opacity="0.8" />
                    
                    <!-- Labels -->
                    <text x="${cx}" y="${cy - r - 6}" text-anchor="middle" fill="var(--text-muted)" font-size="9">稳定性 (${job.sScore}分)</text>
                    <text x="${cx + r * Math.sin((120 * Math.PI) / 180) + 4}" y="${cy - r * Math.cos((120 * Math.PI) / 180) + 10}" text-anchor="start" fill="var(--text-muted)" font-size="9">性能 (${job.pScore}分)</text>
                    <text x="${cx + r * Math.sin((240 * Math.PI) / 180) - 4}" y="${cy - r * Math.cos((240 * Math.PI) / 180) + 10}" text-anchor="end" fill="var(--text-muted)" font-size="9">效率 (${job.eScore}分)</text>

                    <!-- Score Triangle Area -->
                    <polygon points="${sX},${sY} ${pX},${pY} ${eX},${eY}" fill="rgba(59, 130, 246, 0.25)" stroke="#3b82f6" stroke-width="1.5" />
                    <!-- Points -->
                    <circle cx="${sX}" cy="${sY}" r="3" fill="#3b82f6" />
                    <circle cx="${pX}" cy="${pY}" r="3" fill="#3b82f6" />
                    <circle cx="${eX}" cy="${eY}" r="3" fill="#3b82f6" />
                  </svg>
                </div>

                <!-- Lag / CPU trend chart -->
                <div class="chart-container-box" style="height: 180px;">
                  <button class="copilot-btn" @click=${() => (this.showTrendCopilot = !this.showTrendCopilot)}>🤖 问问 AI</button>
                  ${this.showTrendCopilot
                    ? html`
                        <div class="copilot-popover">
                          <strong>AI 诊断解读:</strong>
                          <span>
                            ${job.metrics.lagTrend > 0
                              ? `消费Lag呈阶梯式恶化（积压恶化比例高），且YARN内部呈现反压，判定数据输入超过单算子最大吞吐。`
                              : `消费Lag走势平稳，无堆积风险。`}
                          </span>
                        </div>
                      `
                    : nothing}
                  <!-- Custom SVG line graph -->
                  <svg width="100%" height="100%" viewBox="0 0 400 160" preserveAspectRatio="none">
                    <!-- Grid lines -->
                    <line x1="40" y1="20" x2="380" y2="20" stroke="var(--border)" stroke-opacity="0.4" />
                    <line x1="40" y1="70" x2="380" y2="70" stroke="var(--border)" stroke-opacity="0.4" />
                    <line x1="40" y1="120" x2="380" y2="120" stroke="var(--border)" stroke-opacity="0.4" />
                    <!-- Ticks & Axis -->
                    <line x1="40" y1="20" x2="40" y2="120" stroke="var(--border)" stroke-opacity="0.8" />
                    <line x1="40" y1="120" x2="380" y2="120" stroke="var(--border)" stroke-opacity="0.8" />
                    
                    <!-- Axis Labels -->
                    <text x="35" y="24" text-anchor="end" fill="var(--text-muted)" font-size="8">MAX</text>
                    <text x="35" y="123" text-anchor="end" fill="var(--text-muted)" font-size="8">0</text>
                    <text x="40" y="132" text-anchor="middle" fill="var(--text-muted)" font-size="8">-1h</text>
                    <text x="380" y="132" text-anchor="middle" fill="var(--text-muted)" font-size="8">现在</text>

                    <!-- Mock Trend Polyline (Lag: Red, CPU: Blue) -->
                    <path d="M 40 120 L 70 115 L 100 110 L 130 ${job.metrics.lagTrend > 0 ? "90" : "112"} L 160 ${job.metrics.lagTrend > 0 ? "70" : "110"} L 190 ${job.metrics.lagTrend > 0 ? "50" : "115"} L 220 ${job.metrics.lagTrend > 0 ? "35" : "108"} L 250 ${job.metrics.lagTrend > 0 ? "30" : "110"} L 280 ${job.metrics.lagTrend > 0 ? "25" : "112"} L 310 ${job.metrics.lagTrend > 0 ? "22" : "110"} L 340 ${job.metrics.lagTrend > 0 ? "20" : "108"} L 380 ${job.metrics.lagTrend > 0 ? "18" : "110"}" fill="none" stroke="#ef4444" stroke-width="2" />
                    <path d="M 40 100 L 70 95 L 100 ${job.metrics.cpuAvg > 60 ? "60" : "105"} L 130 ${job.metrics.cpuAvg > 60 ? "40" : "102"} L 160 ${job.metrics.cpuAvg > 60 ? "35" : "98"} L 190 ${job.metrics.cpuAvg > 60 ? "30" : "105"} L 220 ${job.metrics.cpuAvg > 60 ? "28" : "97"} L 250 ${job.metrics.cpuAvg > 60 ? "35" : "100"} L 280 ${job.metrics.cpuAvg > 60 ? "40" : "102"} L 310 ${job.metrics.cpuAvg > 60 ? "32" : "98"} L 340 ${job.metrics.cpuAvg > 60 ? "35" : "100"} L 380 ${job.metrics.cpuAvg > 60 ? "30" : "97"}" fill="none" stroke="#3b82f6" stroke-width="1.5" stroke-dasharray="3,3" />
                  </svg>
                </div>
              </div>

              <!-- Right Side: Chain of Thought Reasoning -->
              <div>
                <div class="diag-sub-header">思维链推理 (Chain of Thought)</div>
                <div class="timeline-steps">
                  <div class="chain-step ${job.cotSteps.step1.state}">
                    <div class="step-title">Step 1: 消费趋势诊断</div>
                    <div class="step-desc">${job.cotSteps.step1.text}</div>
                  </div>
                  <div class="chain-step ${job.cotSteps.step2.state}">
                    <div class="step-title">Step 2: 瓶颈定位 (三角验证)</div>
                    <div class="step-desc">${job.cotSteps.step2.text}</div>
                  </div>
                  <div class="chain-step ${job.cotSteps.step3.state}">
                    <div class="step-title">Step 3: 资源归因与内存红线</div>
                    <div class="step-desc">${job.cotSteps.step3.text}</div>
                  </div>
                </div>

                <div class="action-list">
                  <h5>AI 处方建议 (Actionable Advice)</h5>
                  <ul>
                    ${job.actions.map((act) => html`<li>${act}</li>`)}
                  </ul>
                </div>
              </div>
            </div>

            <!-- Digital Employee Copilot Chat Layout (Spans 100% width) -->
            <div class="copilot-chat-layout">
              <div class="copilot-profile-pane">
                <div class="copilot-profile-header">
                  <span class="copilot-avatar">🤖</span>
                  <div>
                    <div class="copilot-profile-name">BCH 诊断数字员工</div>
                    <div style="display: flex; align-items: center; margin-top: 2px;">
                      <span class="copilot-status-dot"></span>
                      <span class="copilot-status-text">在线值守</span>
                    </div>
                  </div>
                </div>
                <div style="font-size: 11px; color: var(--text-secondary); line-height: 1.4; border-top: 1px solid var(--border); padding-top: 10px; margin-top: 8px;">
                  授权 SOP 技能库：
                  <ul class="copilot-skills-list" style="margin-top: 4px;">
                    <li>Flink 流量积压溯源 SOP</li>
                    <li>YARN 内存与资源三角校验</li>
                    <li>JVM FullGC 根因诊断 SOP</li>
                  </ul>
                </div>
              </div>
              <div class="copilot-chat-pane">
                <div class="copilot-chat-messages">
                  ${this.copilotMessages.map(
                    (msg) => html`
                      <div class="chat-bubble ${msg.sender}">
                        ${msg.sender === 'user' ? msg.text : unsafeHTML(toSanitizedMarkdownHtml(msg.text))}
                      </div>
                    `
                  )}
                </div>
                <div class="copilot-input-area">
                  <textarea
                    class="copilot-textarea"
                    placeholder="请输入关于作业「${job.name}」的调优疑问..."
                    .value=${this.copilotInput}
                    @input=${(e: any) => (this.copilotInput = e.target.value)}
                    @keydown=${(e: KeyboardEvent) => {
                      if (e.key === "Enter" && !e.shiftKey) {
                        e.preventDefault();
                        this.sendCopilotMessage();
                      }
                    }}
                  ></textarea>
                  <button class="copilot-send-button" @click=${this.sendCopilotMessage}>发送</button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    `;
  }

  private renderSparkModal() {
    if (!this.sparkModalOpen || !this.selectedSparkJob) return nothing;
    const job = this.selectedSparkJob;

    return html`
      <div class="modal-backdrop">
        <div class="modal-container" style="max-width: 640px;">
          <div class="modal-header">
            <div class="modal-title">
              <h3>离线计算智能调优建议</h3>
              <p>${job.name}</p>
            </div>
            <button class="close-btn" @click=${() => (this.sparkModalOpen = false)}>&times;</button>
          </div>
          <div class="modal-body">
            <div class="spark-details-grid">
              <div class="spark-detail-item">
                <div class="spark-lbl">Task 耗时长尾比</div>
                <div class="spark-val" style="color: #f59e0b;">
                  ${(job.metrics.maxTaskDurationSec / job.metrics.avgTaskDurationSec).toFixed(1)} 倍
                </div>
              </div>
              <div class="spark-detail-item">
                <div class="spark-lbl">CPU 倾斜率 (Skew)</div>
                <div class="spark-val" style="color: ${job.metrics.cpuSkewRatio > 4 ? "#ef4444" : "#10b981"};">
                  ${job.metrics.cpuSkewRatio}
                </div>
              </div>
              <div class="spark-detail-item">
                <div class="spark-lbl">内存倾斜率 (Skew)</div>
                <div class="spark-val" style="color: ${job.metrics.memorySkewRatio > 4 ? "#ef4444" : "#10b981"};">
                  ${job.metrics.memorySkewRatio}
                </div>
              </div>
              <div class="spark-detail-item">
                <div class="spark-lbl">数据吞吐总量</div>
                <div class="spark-val">${(job.metrics.inputBytes / 1024 / 1024 / 1024).toFixed(0)} GB</div>
              </div>
              <div class="spark-detail-item">
                <div class="spark-lbl">Shuffle 读/写</div>
                <div class="spark-val">${(job.metrics.shuffleReadBytes / 1024 / 1024 / 1024).toFixed(0)} GB</div>
              </div>
              <div class="spark-detail-item">
                <div class="spark-lbl">失败 Task 数</div>
                <div class="spark-val" style="color: ${job.metrics.failedTasks > 0 ? "#ef4444" : "#10b981"};">
                  ${job.metrics.failedTasks} / ${job.metrics.totalTasks}
                </div>
              </div>
            </div>

            <div class="tuning-advice-box">
              <h4>Spark 调优处方</h4>
              <p>
                针对该作业的倾斜指标与失败情况，BCH 作业诊断智能体给出了如下优化参数，您可以将它们复制至 Spark Submit 配置项中：
              </p>
              <pre><code>${job.tuningAdvice}</code></pre>
            </div>
          </div>
        </div>
      </div>
    `;
  }
}
