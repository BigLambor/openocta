import { LitElement, html, css, nothing } from "lit";
import { customElement, property, state } from "lit/decorators.js";
import {
  fetchBchFlinkJobs,
  fetchBchFlinkJobConfig,
  diagnoseBchFlinkJob,
  fetchBchSparkJobs,
  tuneBchSparkJob,
  FlinkJob,
  SparkJob,
} from "../../controllers/bch-client.ts";
import { icons } from "../../icons.ts";

@customElement("bch-job-governance")
export class BchJobGovernance extends LitElement {
  @property({ type: Object }) host: any = null;

  @state() private subTab: "flink" | "spark" = "flink";
  @state() private flinkJobs: FlinkJob[] = [];
  @state() private sparkJobs: SparkJob[] = [];
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
      background: var(--bg-card, #1c1c1e);
      border: 1px solid var(--border, rgba(255, 255, 255, 0.1));
      border-radius: 12px;
      width: 100%;
      max-width: 960px;
      max-height: 85vh;
      overflow-y: auto;
      box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3);
      display: flex;
      flex-direction: column;
    }

    .modal-header {
      padding: 16px 24px;
      border-bottom: 1px solid var(--border, rgba(255, 255, 255, 0.08));
      display: flex;
      justify-content: space-between;
      align-items: center;
      background: rgba(255, 255, 255, 0.01);
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
      color: #93c5fd;
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
      color: #93c5fd;
    }

    .action-list ul {
      margin: 0;
      padding-left: 16px;
      font-size: 11px;
      color: var(--text-secondary);
      line-height: 1.6;
    }

    /* Copilot Chat Console */
    .copilot-chat-console {
      border: 1px solid var(--border);
      background: var(--bg-content);
      border-radius: 8px;
      margin-top: 16px;
      display: flex;
      flex-direction: column;
      height: 180px;
    }

    .chat-header {
      font-size: 11px;
      font-weight: 600;
      padding: 6px 12px;
      background: var(--bg-hover, rgba(0, 0, 0, 0.02));
      border-bottom: 1px solid var(--border);
      color: var(--text-muted);
      display: flex;
      align-items: center;
      gap: 6px;
    }

    .chat-messages {
      flex: 1;
      overflow-y: auto;
      padding: 10px;
      display: flex;
      flex-direction: column;
      gap: 8px;
      font-size: 11px;
    }

    .chat-bubble {
      max-width: 85%;
      padding: 6px 10px;
      border-radius: 6px;
      line-height: 1.4;
    }

    .chat-bubble.user {
      align-self: flex-end;
      background: var(--accent, #3b82f6);
      color: white;
    }

    .chat-bubble.ai {
      align-self: flex-start;
      background: var(--bg);
      border: 1px solid var(--border);
      color: var(--text-primary);
    }

    .chat-input-row {
      display: flex;
      border-top: 1px solid var(--border);
    }

    .chat-input {
      flex: 1;
      background: transparent;
      border: none;
      padding: 8px 12px;
      color: var(--text-primary);
      font-size: 11px;
    }

    .chat-input:focus {
      outline: none;
    }

    .chat-send-btn {
      background: transparent;
      border: none;
      color: var(--accent, #3b82f6);
      padding: 0 12px;
      cursor: pointer;
      font-size: 11px;
      font-weight: 600;
    }

    .chat-send-btn:hover {
      color: white;
    }

    /* Configuration Modal Specific */
    .config-pre {
      background: #09090b;
      color: #4ade80;
      font-family: monospace;
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
      background: #09090b;
      padding: 10px 14px;
      border-radius: 6px;
      font-family: monospace;
      font-size: 11px;
      color: #60a5fa;
      overflow-x: auto;
      margin: 0;
      border: 1px solid rgba(255,255,255,0.05);
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
      if (this.subTab === "flink") {
        this.flinkJobs = await fetchBchFlinkJobs(this.host);
      } else {
        this.sparkJobs = await fetchBchSparkJobs(this.host);
      }
    } catch (err: any) {
      this.error = err.message || String(err);
    } finally {
      this.loading = false;
    }
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

  sendCopilotMessage() {
    const text = this.copilotInput.trim();
    if (!text || !this.selectedFlinkJob) return;

    this.copilotMessages = [...this.copilotMessages, { sender: "user", text }];
    this.copilotInput = "";

    // Generate mock expert responses based on the job context
    const job = this.selectedFlinkJob;
    setTimeout(() => {
      let reply = "";
      if (text.includes("配置") || text.includes("内存") || text.includes("GC")) {
        if (job.rootCause === "S2") {
          reply = `当前该作业存在 ${job.metrics.fullGcCount} 次 Full GC 记录。建议通过 YARN 调整配置，调大 TaskManager 内存：将 \`taskmanager.memory.process.size\` 从 4096m 增加至 8192m，并根据 State 规模增加堆外 RocksDB 缓存。`;
        } else {
          reply = `该作业的 JVM 参数包含了 RocksDB 状态后端配置，当前整体内存处于安全线内，暂不需要专门调大内存配置。`;
        }
      } else if (text.includes("Lag") || text.includes("积压") || text.includes("消费")) {
        if (job.metrics.lagTrend > 0) {
          reply = `根据监测，该作业 Consumer Lag 在过去1小时呈现恶化上升趋势（最大 Lag 达到 ${job.metrics.maxLag}）。结合反压判定，这是由于下游处理较慢。您可以点击左栏的 [实时问诊] 扩容并行度（Parallelism）。`;
        } else {
          reply = `目前作业 Consumer Lag 稳定维持在均值 ${job.metrics.avgLag} 左右，无积压压力，消费速度正常。`;
        }
      } else {
        reply = `对于作业「${job.name}」，AI 诊断建议主要为：${job.actions.join("；") || "持续观察，当前无明显性能瓶颈。"}`;
      }
      this.copilotMessages = [...this.copilotMessages, { sender: "ai", text: reply }];
    }, 600);
  }

  openSparkTuningModal(job: SparkJob) {
    this.selectedSparkJob = job;
    this.sparkModalOpen = true;
  }

  switchSubTab(tab: "flink" | "spark") {
    if (this.subTab === tab) return;
    this.subTab = tab;
    this.loadData();
  }

  render() {
    return html`
      <div class="sub-nav">
        <button
          class="sub-nav-btn ${this.subTab === "flink" ? "active" : ""}"
          @click=${() => this.switchSubTab("flink")}
        >
          Flink 作业健康度
        </button>
        <button
          class="sub-nav-btn ${this.subTab === "spark" ? "active" : ""}"
          @click=${() => this.switchSubTab("spark")}
        >
          Spark 作业调优
        </button>
      </div>

      <div class="governance-content">
        ${this.loading
          ? html`
              <div class="loading-container" style="display: flex; flex-direction: column; align-items: center; justify-content: center; height: 200px; color: var(--text-muted);">
                <div class="spinner" style="width: 24px; height: 24px; border: 2px solid rgba(255, 255, 255, 0.1); border-top-color: var(--accent, #3b82f6); border-radius: 50%; animation: spin 0.8s linear infinite; margin-bottom: 12px;"></div>
                <div>正在加载作业治理数据...</div>
              </div>
            `
          : this.subTab === "flink"
          ? this.renderFlinkContent()
          : this.renderSparkContent()}
      </div>

      ${this.renderConfigModal()}
      ${this.renderDiagnoseModal()}
      ${this.renderSparkModal()}
    `;
  }

  private renderFlinkContent() {
    const healthy = this.flinkJobs.filter((j) => j.score >= 90).length;
    const warning = this.flinkJobs.filter((j) => j.score >= 60 && j.score < 90).length;
    const critical = this.flinkJobs.filter((j) => j.score < 60).length;
    const waste = this.flinkJobs.filter((j) => j.rootCause === "S7").length;

    return html`
      <div class="flink-summary-grid">
        <div class="summary-card">
          <div class="summary-lbl">健康作业 (Score ≥ 90)</div>
          <div class="summary-val healthy">${healthy}</div>
        </div>
        <div class="summary-card">
          <div class="summary-lbl">亚健康作业 (Score 60-89)</div>
          <div class="summary-val warning">${warning}</div>
        </div>
        <div class="summary-card">
          <div class="summary-lbl">高危作业 (Score < 60)</div>
          <div class="summary-val critical">${critical}</div>
        </div>
        <div class="summary-card">
          <div class="summary-lbl">资源闲置作业</div>
          <div class="summary-val info">${waste} <span style="font-size: 12px; font-weight: normal; color: var(--text-muted)">个</span></div>
        </div>
      </div>

      <div class="sec-header">
        <h2>在线流计算作业巡检 (Flink Doctor)</h2>
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
            ${this.flinkJobs.map((job) => {
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
                    <button class="action-icon-btn" title="提取当前运行配置" @click=${() => this.openConfigModal(job)}>
                      ${icons.settings || "配置"}
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

  private renderSparkContent() {
    return html`
      <div class="sec-header">
        <h2>离线批处理作业治理 (Spark Tuning)</h2>
      </div>

      <div class="ops-table-container">
        <table class="ops-table">
          <thead>
            <tr>
              <th>作业名称</th>
              <th>负责人</th>
              <th>运行集群</th>
              <th>作业状态</th>
              <th>优化诊断</th>
              <th>运行时长</th>
              <th>调优处方</th>
            </tr>
          </thead>
          <tbody>
            ${this.sparkJobs.map((job) => {
              let statusColor = "color: #10b981; font-weight: bold;";
              if (job.status === "FAILED") {
                statusColor = "color: #ef4444; font-weight: bold;";
              } else if (job.status === "RUNNING") {
                statusColor = "color: #3b82f6; font-weight: bold;";
              }

              return html`
                <tr>
                  <td style="font-weight: 600;">${job.name}</td>
                  <td style="font-family: monospace; font-size: 11px;">${job.owner}</td>
                  <td><span class="tag-badge">${job.cluster}</span></td>
                  <td style="${statusColor}">${job.status}</td>
                  <td>
                    ${job.labels.map((lbl) => {
                      const c = lbl.includes("OOM") || lbl.includes("倾斜") ? "danger" : "warning";
                      return html`<span class="tag-badge ${c}">${lbl}</span>`;
                    })}
                  </td>
                  <td>${job.durationSec} 秒</td>
                  <td>
                    <button class="diagnose-btn" @click=${() => this.openSparkTuningModal(job)}>
                      查看调优参数
                    </button>
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
          <div class="modal-body" style="background: #09090b; padding: 16px;">
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
            <div class="diag-summary-panel">
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
                        <polygon points="${x1},${y1} ${x2},${y2} ${x3},${y3}" fill="none" stroke="rgba(255,255,255,0.05)" />
                      `;
                    })}
                    <!-- Axes lines -->
                    <line x1="${cx}" y1="${cy}" x2="${cx}" y2="${cy - r}" stroke="rgba(255,255,255,0.1)" />
                    <line x1="${cx}" y1="${cy}" x2="${cx + r * Math.sin((120 * Math.PI) / 180)}" y2="${cy - r * Math.cos((120 * Math.PI) / 180)}" stroke="rgba(255,255,255,0.1)" />
                    <line x1="${cx}" y1="${cy}" x2="${cx + r * Math.sin((240 * Math.PI) / 180)}" y2="${cy - r * Math.cos((240 * Math.PI) / 180)}" stroke="rgba(255,255,255,0.1)" />
                    
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
                    <line x1="40" y1="20" x2="380" y2="20" stroke="rgba(255,255,255,0.03)" />
                    <line x1="40" y1="70" x2="380" y2="70" stroke="rgba(255,255,255,0.03)" />
                    <line x1="40" y1="120" x2="380" y2="120" stroke="rgba(255,255,255,0.03)" />
                    <!-- Ticks & Axis -->
                    <line x1="40" y1="20" x2="40" y2="120" stroke="rgba(255,255,255,0.1)" />
                    <line x1="40" y1="120" x2="380" y2="120" stroke="rgba(255,255,255,0.1)" />
                    
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

                <!-- Digital Employee Copilot Console -->
                <div class="copilot-chat-console">
                  <div class="chat-header">
                    <span>🤖</span>
                    <span>BCH 作业诊断数字员工</span>
                  </div>
                  <div class="chat-messages">
                    ${this.copilotMessages.map(
                      (msg) => html`
                        <div class="chat-bubble ${msg.sender}">${msg.text}</div>
                      `
                    )}
                  </div>
                  <div class="chat-input-row">
                    <input
                      class="chat-input"
                      type="text"
                      placeholder="问问数字员工该作业的调优建议..."
                      .value=${this.copilotInput}
                      @input=${(e: any) => (this.copilotInput = e.target.value)}
                      @keydown=${(e: KeyboardEvent) => e.key === "Enter" && this.sendCopilotMessage()}
                    />
                    <button class="chat-send-btn" @click=${this.sendCopilotMessage}>发送</button>
                  </div>
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
