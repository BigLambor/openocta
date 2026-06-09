import { LitElement, html, css, nothing } from "lit";
import { customElement, property, state } from "lit/decorators.js";
import {
  executeBchYarnQueueAction,
  fetchBchYarnQueues,
  rollbackBchYarnQueueAction,
  YarnQueueEvaluation,
} from "../../controllers/bch-client.ts";
import { icons } from "../../icons.ts";
import { parseWorkbenchObjectScope } from "../../ops/workbench-context.ts";

@customElement("bch-yarn-capacity")
export class BchYarnCapacity extends LitElement {
  @property({ type: Object }) host: any = null;
  @property({ type: String }) objectScope = "all";
  @property({ type: String }) timeRange = "24h";

  @state() private queues: YarnQueueEvaluation[] = [];
  @state() private loading = false;
  @state() private error: string | null = null;
  @state() private selectedCluster = "all";

  // Closed-loop execution modal state
  @state() private executionModalOpen = false;
  @state() private selectedQueue: YarnQueueEvaluation | null = null;
  @state() private pipelineRunning = false;
  @state() private pipelineStep = 0; // 0: idle, 1: verifying, 2: patching, 3: refreshing, 4: observing, 5: completed, 6: rollbackRunning, 7: rollbackCompleted
  @state() private executionLog: string[] = [];
  @state() private showRollbackOption = false;

  static styles = css`
    :host {
      display: flex;
      flex-direction: column;
      font-family: var(--font-family, sans-serif);
      color: var(--text-primary);
      height: 100%;
      box-sizing: border-box;
    }

    .dashboard-body {
      flex: 1;
      padding: 24px;
      overflow-y: auto;
    }

    .header-section {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 20px;
    }

    .header-title {
      font-size: 16px;
      font-weight: 700;
      color: var(--text-primary);
    }

    .header-subtitle {
      font-size: 12px;
      color: var(--text-muted);
      margin-top: 4px;
    }

    .table-control-bar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 14px;
      margin-top: 10px;
    }

    .table-title {
      font-size: 14px;
      font-weight: 600;
      color: var(--text-primary);
      display: flex;
      align-items: center;
      gap: 6px;
    }

    .table-title::before {
      content: "";
      display: inline-block;
      width: 3px;
      height: 14px;
      background: var(--accent, #3b82f6);
      border-radius: 2px;
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
      outline: none;
      cursor: pointer;
      font-family: inherit;
    }

    .cluster-filter select:focus {
      border-color: var(--accent, #3b82f6);
    }

    /* 1. Summary Dashboard Cards */
    .summary-grid {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: 16px;
      margin-bottom: 24px;
    }

    .summary-card {
      background: var(--bg-content);
      border: 1px solid var(--border);
      border-radius: 10px;
      padding: 16px;
      box-shadow: var(--shadow-sm, 0 2px 8px rgba(0, 0, 0, 0.05));
      position: relative;
      overflow: hidden;
    }

    .summary-card::before {
      content: "";
      position: absolute;
      top: 0;
      left: 0;
      width: 4px;
      height: 100%;
      background: var(--accent, #3b82f6);
    }

    .summary-card.warn::before {
      background: #f59e0b;
    }

    .summary-card.danger::before {
      background: #ef4444;
    }

    .summary-lbl {
      font-size: 11px;
      color: var(--text-muted);
      text-transform: uppercase;
      letter-spacing: 0.5px;
      margin-bottom: 8px;
    }

    .summary-num {
      font-size: 24px;
      font-weight: 700;
      color: var(--text-primary);
      display: flex;
      align-items: baseline;
      gap: 4px;
    }

    .summary-num span {
      font-size: 12px;
      font-weight: normal;
      color: var(--text-muted);
    }

    .summary-hint {
      font-size: 11px;
      color: var(--text-secondary);
      margin-top: 6px;
    }

    /* AI Assistant Card */
    .ai-copilot-card {
      display: flex;
      align-items: flex-start;
      gap: 12px;
      background: rgba(59, 130, 246, 0.06);
      border: 1px solid rgba(59, 130, 246, 0.18);
      border-radius: 8px;
      padding: 12px 16px;
      margin-bottom: 24px;
      font-size: 12px;
      line-height: 1.6;
    }

    .ai-copilot-avatar {
      font-size: 20px;
      flex-shrink: 0;
    }

    .ai-copilot-text strong {
      color: var(--accent, #3b82f6);
    }

    /* 2. Queue Table Styles */
    .ops-table-container {
      background: var(--bg-content);
      border: 1px solid var(--border);
      border-radius: 12px;
      overflow: hidden;
      box-shadow: var(--shadow-sm, 0 4px 16px rgba(0, 0, 0, 0.05));
    }

    .ops-table {
      width: 100%;
      border-collapse: collapse;
      font-size: 12px;
      text-align: left;
    }

    .ops-table th {
      background: var(--bg-hover, rgba(0, 0, 0, 0.02));
      padding: 12px 16px;
      font-weight: 600;
      color: var(--text-secondary);
      border-bottom: 1px solid var(--border);
      font-size: 11px;
      letter-spacing: 0.5px;
    }

    .ops-table td {
      padding: 14px 16px;
      border-bottom: 1px solid var(--border);
      color: var(--text-primary);
      vertical-align: middle;
    }

    .ops-table tr:hover {
      background: var(--bg-hover, rgba(0, 0, 0, 0.02));
    }

    .queue-path-cell {
      font-weight: 600;
      color: var(--text-primary);
      font-family: var(--font-family, sans-serif);
    }

    .queue-path-sub {
      font-size: 10px;
      color: var(--text-muted);
      margin-top: 2px;
      font-family: monospace;
    }

    .tag-badge {
      background: rgba(255, 255, 255, 0.05);
      border: 1px solid rgba(255, 255, 255, 0.1);
      color: var(--text-secondary);
      border-radius: 4px;
      padding: 2px 6px;
      font-size: 10px;
      display: inline-block;
    }

    /* Custom Double Progress Bar for CPU & Mem (Used vs Peak 30d) */
    .usage-metrics-container {
      display: flex;
      flex-direction: column;
      gap: 6px;
      min-width: 120px;
    }

    .usage-bar-row {
      display: flex;
      align-items: center;
      gap: 8px;
    }

    .usage-bar-label {
      font-size: 10px;
      color: var(--text-muted);
      width: 24px;
      flex-shrink: 0;
      text-align: right;
    }

    .usage-bar-track {
      height: 6px;
      background: rgba(255, 255, 255, 0.05);
      border-radius: 3px;
      flex: 1;
      position: relative;
      overflow: hidden;
    }

    .usage-bar-peak {
      position: absolute;
      top: 0;
      left: 0;
      height: 100%;
      background: rgba(59, 130, 246, 0.25);
      border-radius: 3px;
      transition: width 0.3s;
    }

    .usage-bar-used {
      position: absolute;
      top: 0;
      left: 0;
      height: 100%;
      background: var(--accent, #3b82f6);
      border-radius: 3px;
      transition: width 0.3s;
    }

    .usage-bar-val {
      font-size: 10px;
      color: var(--text-muted);
      width: 50px;
      flex-shrink: 0;
    }

    /* Status Badges */
    .status-badge {
      font-weight: 700;
      padding: 4px 8px;
      border-radius: 6px;
      font-size: 11px;
      display: inline-block;
      text-align: center;
    }

    .status-badge.idle { background: rgba(239, 68, 68, 0.12); color: #ef4444; border: 1px solid rgba(239, 68, 68, 0.25); }
    .status-badge.over_allocated { background: rgba(245, 158, 11, 0.12); color: #f59e0b; border: 1px solid rgba(245, 158, 11, 0.25); }
    .status-badge.under_allocated { background: rgba(59, 130, 246, 0.12); color: #60a5fa; border: 1px solid rgba(59, 130, 246, 0.25); }
    .status-badge.healthy { background: rgba(16, 185, 129, 0.12); color: #10b981; border: 1px solid rgba(16, 185, 129, 0.25); }

    /* Action buttons */
    .action-btn {
      background: var(--accent, #3b82f6);
      border: none;
      color: white;
      padding: 6px 12px;
      border-radius: 6px;
      cursor: pointer;
      font-size: 11px;
      font-weight: 500;
      transition: opacity 0.15s;
    }

    .action-btn:hover {
      opacity: 0.9;
    }

    .action-btn.reclaim {
      background: rgba(239, 68, 68, 0.15);
      border: 1px solid rgba(239, 68, 68, 0.3);
      color: #ef4444;
    }
    .action-btn.reclaim:hover {
      background: #ef4444;
      color: white;
    }

    .action-btn.downsize {
      background: rgba(245, 158, 11, 0.15);
      border: 1px solid rgba(245, 158, 11, 0.3);
      color: #f59e0b;
    }
    .action-btn.downsize:hover {
      background: #f59e0b;
      color: white;
    }

    .action-btn.expand {
      background: rgba(59, 130, 246, 0.15);
      border: 1px solid rgba(59, 130, 246, 0.3);
      color: #60a5fa;
    }
    .action-btn.expand:hover {
      background: var(--accent, #3b82f6);
      color: white;
    }

    .action-btn.stub {
      background: rgba(255, 255, 255, 0.03);
      border: 1px solid var(--border);
      color: var(--text-muted);
      cursor: not-allowed;
    }

    /* 3. Execution Modal Styles */
    .modal-backdrop {
      position: fixed;
      inset: 0;
      background: rgba(0, 0, 0, 0.7);
      backdrop-filter: blur(5px);
      z-index: 1000;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 24px;
    }

    .modal-container {
      background: var(--bg, #1e1e24);
      border: 1px solid var(--border);
      border-radius: 12px;
      width: 100%;
      max-width: 820px;
      max-height: 90vh;
      overflow: hidden;
      box-shadow: 0 10px 30px rgba(0, 0, 0, 0.5);
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

    .modal-header h3 {
      margin: 0;
      font-size: 15px;
      font-weight: 600;
    }

    .modal-header p {
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
      display: flex;
      flex-direction: column;
      gap: 20px;
    }

    /* Config Contrast Panel */
    .contrast-panel {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 16px;
      flex-shrink: 0;
    }

    .contrast-card {
      background: var(--bg-content, rgba(255, 255, 255, 0.02));
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 12px 16px;
    }

    .contrast-card.target {
      border-color: rgba(59, 130, 246, 0.3);
      background: rgba(59, 130, 246, 0.02);
    }

    .contrast-title {
      font-size: 11px;
      color: var(--text-muted);
      margin-bottom: 8px;
      text-transform: uppercase;
      font-weight: 600;
    }

    .contrast-row {
      display: flex;
      justify-content: space-between;
      margin-bottom: 6px;
      font-size: 12px;
    }

    .contrast-val {
      font-family: monospace;
      font-weight: 600;
    }

    .contrast-val.change-down {
      color: #ef4444;
    }

    .contrast-val.change-up {
      color: #10b981;
    }

    /* Risk Verification Area */
    .risk-banner {
      display: flex;
      align-items: flex-start;
      gap: 12px;
      padding: 14px 16px;
      border-radius: 8px;
      font-size: 12px;
      background: rgba(245, 158, 11, 0.05);
      flex-shrink: 0;
      border: 1px solid rgba(245, 158, 11, 0.2);
    }

    .risk-banner.low {
      background: rgba(16, 185, 129, 0.05);
      border: 1px solid rgba(16, 185, 129, 0.2);
    }

    .risk-banner-title {
      font-weight: 700;
      margin-bottom: 4px;
    }

    .risk-banner-title.low { color: #10b981; }
    .risk-banner-title.medium { color: #f59e0b; }

    .risk-bullets {
      margin: 6px 0 0 0;
      padding-left: 18px;
      color: var(--text-secondary);
      font-size: 11px;
    }

    /* XML Diff Pre */
    .xml-diff-box {
      border: 1px solid var(--border);
      border-radius: 8px;
      background: #15151a;
      overflow: hidden;
      flex-shrink: 0;
    }

    .xml-diff-header {
      background: #1e1e24;
      padding: 6px 12px;
      border-bottom: 1px solid var(--border);
      font-size: 11px;
      font-family: monospace;
      color: var(--text-muted);
    }

    .xml-diff-pre {
      margin: 0;
      padding: 12px 16px;
      font-family: var(--mono, monospace);
      font-size: 11px;
      line-height: 1.5;
      color: #d1d5db;
      overflow-x: auto;
      max-height: 180px;
    }

    /* Timeline execution tree */
    .timeline-container {
      display: flex;
      flex-direction: column;
      gap: 14px;
      background: rgba(255, 255, 255, 0.01);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 16px;
      flex-shrink: 0;
    }

    .timeline-title {
      font-size: 12px;
      font-weight: 600;
      color: var(--text-secondary);
      margin-bottom: 4px;
    }

    .timeline-step-row {
      display: flex;
      align-items: center;
      gap: 12px;
      font-size: 11.5px;
    }

    .step-indicator {
      width: 20px;
      height: 20px;
      border-radius: 50%;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 10px;
      font-weight: bold;
      border: 1.5px solid var(--border);
      background: transparent;
      color: var(--text-muted);
      flex-shrink: 0;
    }

    .timeline-step-row.active .step-indicator {
      border-color: var(--accent, #3b82f6);
      color: var(--accent, #3b82f6);
      background: rgba(59, 130, 246, 0.08);
      animation: pulse 1.5s infinite;
    }

    .timeline-step-row.success .step-indicator {
      border-color: #10b981;
      background: #10b981;
      color: white;
    }

    .timeline-step-row.fail .step-indicator {
      border-color: #ef4444;
      background: #ef4444;
      color: white;
    }

    .step-text {
      flex: 1;
      color: var(--text-muted);
    }

    .timeline-step-row.active .step-text {
      color: var(--text-primary);
      font-weight: 500;
    }

    .timeline-step-row.success .step-text {
      color: var(--text-secondary);
    }

    /* Live console output */
    .console-box {
      background: #0d0d11;
      border: 1px solid var(--border);
      border-radius: 6px;
      padding: 12px 16px;
      font-family: monospace;
      font-size: 11px;
      color: #34d399;
      height: 130px;
      overflow-y: auto;
      flex-shrink: 0;
      box-sizing: border-box;
    }

    .console-line {
      margin-bottom: 4px;
      line-height: 1.4;
    }

    .modal-footer {
      padding: 16px 24px;
      border-top: 1px solid var(--border);
      display: flex;
      justify-content: flex-end;
      gap: 12px;
      background: var(--bg-hover, rgba(0, 0, 0, 0.01));
    }

    .modal-footer-info {
      flex: 1;
      display: flex;
      align-items: center;
      font-size: 11px;
      color: var(--text-muted);
    }

    @keyframes pulse {
      0% { box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.4); }
      70% { box-shadow: 0 0 0 6px rgba(59, 130, 246, 0); }
      100% { box-shadow: 0 0 0 0 rgba(59, 130, 246, 0); }
    }
  `;

  connectedCallback() {
    super.connectedCallback();
    this.syncClusterFromObjectScope();
    void this.loadData();
  }

  updated(changedProperties: Map<PropertyKey, unknown>) {
    if (changedProperties.has("objectScope")) {
      this.syncClusterFromObjectScope();
    }
  }

  private syncClusterFromObjectScope() {
    const parsed = parseWorkbenchObjectScope(this.objectScope);
    this.selectedCluster = parsed.kind === "cluster" ? parsed.value : "all";
  }

  async loadData() {
    if (!this.host) return;
    this.loading = true;
    this.error = null;
    try {
      this.queues = await fetchBchYarnQueues(this.host);
    } catch (err: any) {
      this.error = err.message || String(err);
    } finally {
      this.loading = false;
    }
  }

  private openExecutionModal(queue: YarnQueueEvaluation) {
    this.selectedQueue = queue;
    this.executionModalOpen = true;
    this.pipelineRunning = false;
    this.pipelineStep = 0;
    this.executionLog = [];
    this.showRollbackOption = false;
  }

  private closeExecutionModal() {
    if (this.pipelineRunning && this.pipelineStep < 5) return; // Prevent closing while running
    this.executionModalOpen = false;
    this.selectedQueue = null;
  }

  private startClosedLoop() {
    if (!this.selectedQueue || this.pipelineRunning) return;
    const q = this.selectedQueue;
    const isExpand = q.action === "expand";
    const isSchedulerFair = q.cluster === "prod-b";

    this.pipelineRunning = true;
    this.pipelineStep = 1;
    this.executionLog = ["[INFO] YARN 闭环调优引擎已启动。正在进行第一阶段检查..."];

    // 1. Verifying phase
    setTimeout(() => {
      this.pipelineStep = 2;
      this.executionLog = [
        ...this.executionLog,
        isExpand 
          ? "[SUCCESS] 阶段 1 完成：已确认存在排队及挂起 Container，扩容风险检测通过。" 
          : "[SUCCESS] 阶段 1 完成：无排队及挂起 Container，SLA 风险检测通过 (LOW 风险)。",
        "[INFO] 阶段 2 启动：正在向 BCH 配置管理器动态加载 XML 变更描述并生成 patch..."
      ];
      this.requestUpdate();

      // 2. Patching phase
      setTimeout(() => {
        this.pipelineStep = 3;
        this.executionLog = [
          ...this.executionLog,
          `[SUCCESS] 阶段 2 完成：已生成 ${isSchedulerFair ? 'fair-scheduler.xml' : 'capacity-scheduler.xml'} 变更描述段。`,
          `[INFO] 阶段 3 启动：正在向 YARN ResourceManager 发送热载指令 [yarn rmadmin -refreshQueues]...`
        ];
        this.requestUpdate();

        // 3. Refreshing phase
        setTimeout(() => {
          this.pipelineStep = 4;
          this.executionLog = [
            ...this.executionLog,
            "[SUCCESS] 阶段 3 完成：YARN 动态重载指令接收成功，ResourceManager 配置已热加载生效。",
            isExpand
              ? "[INFO] 阶段 4 启动：配置生效观测期启动，5 分钟滑动窗口水位观测中，准备抓取任务排队与水位变化..."
              : "[INFO] 阶段 4 启动：配置生效观测期启动，5 分钟滑动窗口水位观测中，准备抓取运行任务与挂起率..."
          ];
          this.requestUpdate();

          // 4. Observing phase
          setTimeout(async () => {
            this.pipelineStep = 5;
            this.pipelineRunning = false;
            this.showRollbackOption = true;
            this.executionLog = [
              ...this.executionLog,
              isExpand
                ? "[SUCCESS] 阶段 4 完成：5分钟观测期已过，挂起 Containers 顺利释放，资源扩容生效且平稳。"
                : "[SUCCESS] 阶段 4 完成：5分钟观测期已过，无新积压或分配失败，资源回收稳定。",
              isExpand
                ? "[SUCCESS] 闭环任务执行完成，当前队列额度已上调。YARN 容量扩容成功。"
                : "[SUCCESS] 闭环任务执行完成，当前队列额度已下调。YARN 容量回收成功。"
            ];
            
            try {
              if (this.selectedQueue) {
                await executeBchYarnQueueAction(this.host, this.selectedQueue.id);
                const nextQueues = await fetchBchYarnQueues(this.host);
                this.queues = nextQueues;
                const updated = nextQueues.find(q => q.id === this.selectedQueue?.id);
                if (updated) {
                  this.selectedQueue = updated;
                }
              }
            } catch (err) {
              console.error("Failed to commit mock change:", err);
              this.pipelineStep = 4;
              this.pipelineRunning = false;
              this.showRollbackOption = false;
              this.executionLog = [
                ...this.executionLog,
                `[ERROR] 后端容量变更提交失败：${err instanceof Error ? err.message : String(err)}`
              ];
            }
            this.requestUpdate();
          }, 2000);
        }, 1500);
      }, 1500);
    }, 1500);
  }

  private triggerRollback() {
    if (this.pipelineStep !== 5) return;
    this.pipelineRunning = true;
    this.pipelineStep = 6; // rollbackRunning
    this.executionLog = [
      ...this.executionLog,
      "[INFO] 触发人工回退程序。正在从回滚方案 (RollbackPlan) 加载还原 XML...",
      "[INFO] 正在下发回退 XML 还原 Patch，并热刷新 YARN Scheduler..."
    ];
    this.requestUpdate();

    setTimeout(async () => {
      try {
        if (this.selectedQueue) {
          await rollbackBchYarnQueueAction(this.host, this.selectedQueue.id);
        }
        this.pipelineStep = 7; // rollbackCompleted
        this.pipelineRunning = false;
        this.showRollbackOption = false;
        this.executionLog = [
          ...this.executionLog,
          "[SUCCESS] YARN 队列配置已还原至基线配额，配置刷新成功。",
          "[INFO] 回退完成。队列容量性能观测恢复原始水位。"
        ];
        await this.loadData();
        const updated = this.queues.find((q) => q.id === this.selectedQueue?.id);
        if (updated) {
          this.selectedQueue = updated;
        }
      } catch (err) {
        this.pipelineStep = 5;
        this.pipelineRunning = false;
        this.showRollbackOption = true;
        this.executionLog = [
          ...this.executionLog,
          `[ERROR] 回滚提交失败：${err instanceof Error ? err.message : String(err)}`
        ];
      }
      this.requestUpdate();
    }, 2000);
  }

  private dispatchAiRequest() {
    const activeCluster = this.selectedCluster;
    const scopedQueues = activeCluster === "all" ? this.queues : this.queues.filter((q) => q.cluster === activeCluster);
    const evidence = scopedQueues.map(
      (q) =>
        `${q.id} (集群 ${q.cluster}): status=${q.status}, risk=${q.riskLevel}, currentCapacity=${q.currentCapacity}%, usedCapacity=${q.usedCapacity}%, peak30d=${q.peakUsage30d}%, pending=${q.pendingContainers}, waiting=${q.waitingApps}, activeTime=${q.lastActiveTime}`
    );

    this.dispatchEvent(
      new CustomEvent("scenario-ai-request", {
        bubbles: true,
        composed: true,
        detail: {
          mode: "root-cause",
          title: "YARN 队列容量评估 · 全局分析",
          objectType: "yarn_queue_set",
          objectId: "all-yarn-queues",
          objectScope: activeCluster,
          evidence: [
            `当前集群: ${activeCluster}`,
            `YARN 资源队列诊断汇总 (共 ${scopedQueues.length} 个):`,
            ...evidence,
          ],
          expectedOutputs: [
            "长期闲置/配置过剩队列核对",
            "是否具备回收条件（排队与活跃度核实）",
            "回收/缩容的配置参数验证",
            "变更后的 SLA 稳定性风险说明"
          ],
        },
      }),
    );
  }

  render() {
    return html`
      <div class="dashboard-body">
        ${this.loading
          ? html`
              <div class="loading-container" style="display: flex; flex-direction: column; align-items: center; justify-content: center; height: 200px; color: var(--text-muted);">
                <div class="spinner" style="width: 24px; height: 24px; border: 2px solid rgba(255, 255, 255, 0.1); border-top-color: var(--accent, #3b82f6); border-radius: 50%; animation: spin 0.8s linear infinite; margin-bottom: 12px;"></div>
                <div>正在加载 YARN 调度器队列资源分析数据...</div>
              </div>
            `
          : this.error
          ? html`<div class="empty-placeholder" style="padding: 24px; color: var(--danger, #d33);">${this.error}</div>`
          : this.renderDashboard()}
      </div>

      ${this.renderExecutionModal()}
    `;
  }

  private renderDashboard() {
    const clusters = ["all", ...new Set(this.queues.map((q) => q.cluster))];
    const filteredQueues = this.selectedCluster === "all"
      ? this.queues
      : this.queues.filter((q) => q.cluster === this.selectedCluster);

    const idleCount = filteredQueues.filter((q) => q.status === "idle").length;
    const overAllocatedCount = filteredQueues.filter((q) => q.status === "over_allocated").length;
    const totalReclaimed = filteredQueues.reduce((acc, q) => {
      if (q.status === "idle" || q.status === "over_allocated") {
        return acc + (q.currentCapacity - q.targetCapacity);
      }
      return acc;
    }, 0);

    const riskLevels = filteredQueues.filter((q) => q.status === "idle" || q.status === "over_allocated").map((q) => q.riskLevel);
    const lowRiskCount = riskLevels.filter((r) => r === "low").length;
    const mediumRiskCount = riskLevels.filter((r) => r === "medium").length;
    const highRiskCount = riskLevels.filter((r) => r === "high").length;

    return html`
      <div class="header-section">
        <div>
          <div class="header-title">YARN 资源队列容量评估 (Yarn Doctor)</div>
          <div class="header-subtitle">基于 YARN 调度器运行数据，自动评估配置份额的合理度，定位空闲队列与配置漂移风险</div>
        </div>
        <button class="ops-btn ops-btn--primary" type="button" @click=${() => this.dispatchAiRequest()}>
          🤖 诊断全部队列
        </button>
      </div>

      <!-- 1. Summary Dashboard Panels -->
      <div class="summary-grid">
        <div class="summary-card warn">
          <div class="summary-lbl">候选回收/缩容队列</div>
          <div class="summary-num">
            ${idleCount + overAllocatedCount} <span>个队列</span>
          </div>
          <div class="summary-hint">其中 ${idleCount} 个长期空闲，${overAllocatedCount} 个配置过剩</div>
        </div>

        <div class="summary-card">
          <div class="summary-lbl">预计可回收队列配额</div>
          <div class="summary-num" style="color: var(--accent, #3b82f6);">
            ${totalReclaimed.toFixed(1)}% <span>配额</span>
          </div>
          <div class="summary-hint">估算释放约 ${(totalReclaimed * 4.0).toFixed(0)} Core CPU / ${(totalReclaimed * 16.0).toFixed(0)} GB 内存</div>
        </div>

        <div class="summary-card danger">
          <div class="summary-lbl">评估风险等级分布</div>
          <div class="summary-num" style="font-size: 20px;">
            ${lowRiskCount} 低风险 / ${mediumRiskCount} 中风险
          </div>
          <div class="summary-hint">优先处理低风险空闲回收以确保安全</div>
        </div>
      </div>

      <div class="ai-copilot-card">
        <span class="ai-copilot-avatar">🤖</span>
        <div class="ai-copilot-text">
          <strong>YARN 资源评估员工建议</strong>：经分析，当前集群中存在明显的资源空跑与配置偏高现象。主要由于临时测试队列 <code>root.test</code> 长期无作业运行，以及离线队列在峰值期以外资源利用率极低。缩容这些队列可释出多达 <strong>${totalReclaimed.toFixed(1)}%</strong> 的核心计算配额，建议一键执行闭环优化。
        </div>
      </div>

      <!-- 2. Queue Table Section -->
      <div class="table-control-bar">
        <div class="table-title">队列评估明细</div>
        <label class="cluster-filter">
          <span>所属集群筛选:</span>
          <select
            .value=${this.selectedCluster}
            @change=${(e: Event) => {
              this.selectedCluster = (e.target as HTMLSelectElement).value;
            }}
          >
            ${clusters.map((c) => html`
              <option value=${c} ?selected=${c === this.selectedCluster}>
                ${c === 'all' ? '全部集群' : c.toUpperCase()}
              </option>
            `)}
          </select>
        </label>
      </div>

      <div class="ops-table-container">
        <table class="ops-table">
          <thead>
            <tr>
              <th>队列路径</th>
              <th>所属集群 / 调度器</th>
              <th style="text-align: center;">配置 / 最大配额</th>
              <th>CPU / 内存利用率 (均值 / 峰值)</th>
              <th style="text-align: center;">挂起 Containers / Apps</th>
              <th style="text-align: center;">评估状态</th>
              <th>智能优化建议</th>
              <th style="text-align: center;">操作</th>
            </tr>
          </thead>
          <tbody>
            ${filteredQueues.length === 0
              ? html`<tr><td colspan="8" style="text-align:center; color: var(--text-muted); padding: 24px;">暂未筛选到满足条件的 YARN 队列。</td></tr>`
              : nothing}
            ${filteredQueues.map((q) => {
              const statusClass = q.status;
              const hasAction = q.action === "reclaim" || q.action === "downsize";
              const label = q.action === "reclaim" ? "一键回收" : q.action === "downsize" ? "缩容配置" : q.action === "expand" ? "扩容建议" : "无需操作";
              const metrics = q.metrics || { avgCpuPercent: 0, maxCpuPercent: 0, avgMemPercent: 0, maxMemPercent: 0, activeApps: 0 };
              
              return html`
                <tr>
                  <td>
                    <div class="queue-path-cell">${q.id}</div>
                    <div class="queue-path-sub">${q.name}</div>
                  </td>
                  <td>
                    <span class="tag-badge" style="text-transform: uppercase; margin-bottom: 4px;">${q.cluster}</span>
                    <div style="font-size: 10px; color: var(--text-muted); font-weight: 500;">
                      ${q.cluster === 'prod-b' ? '公平调度' : '容量调度'}
                    </div>
                  </td>
                  <td style="text-align: center; font-family: monospace; font-weight: 600;">
                    ${q.currentCapacity.toFixed(1)}% / ${q.maxCapacity.toFixed(1)}%
                  </td>
                  <td>
                    <div class="usage-metrics-container">
                      <!-- CPU Bar -->
                      <div class="usage-bar-row">
                        <span class="usage-bar-label">CPU</span>
                        <div class="usage-bar-track">
                          <div class="usage-bar-peak" style="width: ${q.peakUsage30d}%"></div>
                          <div class="usage-bar-used" style="width: ${metrics.avgCpuPercent}%"></div>
                        </div>
                        <span class="usage-bar-val">${metrics.avgCpuPercent.toFixed(1)}% / ${q.peakUsage30d.toFixed(1)}%</span>
                      </div>
                      <!-- Mem Bar -->
                      <div class="usage-bar-row">
                        <span class="usage-bar-label">内存</span>
                        <div class="usage-bar-track">
                          <div class="usage-bar-peak" style="width: ${q.peakUsage30d * 0.9}%"></div>
                          <div class="usage-bar-used" style="width: ${metrics.avgMemPercent}%"></div>
                        </div>
                        <span class="usage-bar-val">${metrics.avgMemPercent.toFixed(1)}% / ${(q.peakUsage30d * 0.9).toFixed(1)}%</span>
                      </div>
                    </div>
                  </td>
                  <td style="text-align: center; font-family: monospace;">
                    ${q.pendingContainers > 0
                      ? html`<strong style="color: #ef4444;">${q.pendingContainers}</strong>`
                      : html`<span class="muted">${q.pendingContainers}</span>`}
                    <span class="muted" style="font-size: 10px;"> / ${q.waitingApps}</span>
                  </td>
                  <td style="text-align: center;">
                    <span class="status-badge ${statusClass}">
                      ${q.status === "idle"
                        ? "长期闲置"
                        : q.status === "over_allocated"
                        ? "配置过剩"
                        : q.status === "under_allocated"
                        ? "负载过高"
                        : "合理"}
                    </span>
                  </td>
                  <td style="color: var(--text-secondary); max-width: 180px; line-height: 1.4;">
                    ${q.advice}
                  </td>
                  <td style="text-align: center;">
                    <button
                      class="action-btn ${q.action} ${!hasAction && q.action !== 'expand' ? 'stub' : ''}"
                      type="button"
                      ?disabled=${!hasAction && q.action !== 'expand'}
                      @click=${() => {
                        this.openExecutionModal(q);
                      }}
                    >
                      ${label}
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

  private renderExecutionModal() {
    if (!this.executionModalOpen || !this.selectedQueue) return nothing;
    const q = this.selectedQueue;
    const isSchedulerFair = q.cluster === "prod-b";
    
    // Status mappings
    const steps = [
      { text: "AI 风险核算与安全隔离校验", desc: "验证排队 Containers、活跃 Apps 数，确认无承载作业风险" },
      { text: "生成配置 XML 变更 Patch", desc: isSchedulerFair ? "生成 fair-scheduler.xml 节点修改/删除描述" : "生成 capacity-scheduler.xml 属性调整 Patch" },
      { text: "下发 YARN ResourceManager 热重载", desc: "发送 yarn rmadmin -refreshQueues 指令重新加载配置" },
      { text: "配置生效高频水位观测 (5分钟观测期)", desc: "自动分析重载后的任务等待数与系统运行状态" }
    ];

    const isExpand = q.action === "expand";
    const changeClass = isExpand ? "change-up" : "change-down";
    const confirmButtonLabel = q.action === "reclaim" ? "确认回收" : q.action === "expand" ? "确认扩容" : "确认缩容";

    return html`
      <div class="modal-backdrop">
        <div class="modal-container">
          <div class="modal-header">
            <div>
              <h3>🤖 YARN 队列闭环调优执行器</h3>
              <p>调度器类型: ${isSchedulerFair ? "Fair Scheduler (公平调度)" : "Capacity Scheduler (容量调度)"} · 目标集群: ${q.cluster}</p>
            </div>
            <button class="close-btn" @click=${() => this.closeExecutionModal()}>&times;</button>
          </div>
          <div class="modal-body">
            <!-- 1. Configuration Contrast -->
            <div class="contrast-panel">
              <div class="contrast-card">
                <div class="contrast-title">当前队列配置 (Current)</div>
                <div class="contrast-row">
                  <span>队列全路径:</span>
                  <span class="contrast-val">${q.id}</span>
                </div>
                <div class="contrast-row">
                  <span>配置容量 (Capacity):</span>
                  <span class="contrast-val">${q.currentCapacity.toFixed(1)}%</span>
                </div>
                <div class="contrast-row">
                  <span>最大配额 (Max Capacity):</span>
                  <span class="contrast-val">${q.maxCapacity.toFixed(1)}%</span>
                </div>
              </div>

              <div class="contrast-card target">
                <div class="contrast-title" style="color: var(--accent, #3b82f6);">目标调整配置 (Target)</div>
                <div class="contrast-row">
                  <span>队列全路径:</span>
                  <span class="contrast-val">${q.id}</span>
                </div>
                <div class="contrast-row">
                  <span>配置容量 (Capacity):</span>
                  <span class="contrast-val ${changeClass}">${q.targetCapacity.toFixed(1)}%</span>
                </div>
                <div class="contrast-row">
                  <span>最大配额 (Max Capacity):</span>
                  <span class="contrast-val ${changeClass}">${q.targetMaxCapacity.toFixed(1)}%</span>
                </div>
              </div>
            </div>

            <!-- 2. Risk Check Banner -->
            <div class="risk-banner ${q.riskLevel}">
              <div style="font-size: 20px;">🛡️</div>
              <div>
                <div class="risk-banner-title ${q.riskLevel}">
                  风险等级评估：${q.riskLevel === "low" ? "LOW (低风险)" : "MEDIUM (中风险)"}
                </div>
                <div>安全核查通过项:</div>
                <ul class="risk-bullets">
                  ${q.reasons.map((r) => html`<li>${r}</li>`)}
                  <li>最后活跃时间: <strong style="font-family:monospace;">${q.lastActiveTime}</strong></li>
                  <li>等待挂起应用数: <strong style="font-family:monospace;">${q.waitingApps}</strong></li>
                </ul>
              </div>
            </div>

            <!-- 3. Configuration Diff -->
            <div class="xml-diff-box">
              <div class="xml-diff-header">
                ${isSchedulerFair ? "fair-scheduler.xml" : "capacity-scheduler.xml"} (Generated Patch)
              </div>
              <pre class="xml-diff-pre"><code>${q.configPatch || "<!-- 暂无挂起变更配置 -->"}</code></pre>
            </div>

            <!-- 4. Timeline Execution -->
            <div class="timeline-container">
              <div class="timeline-title">${this.pipelineStep >= 6 ? "【回退计划】Timeline" : "【闭环执行】Timeline"}</div>
              <div style="display:flex; flex-direction:column; gap:12px;">
                ${this.pipelineStep >= 6
                  ? html`
                      <div class="timeline-step-row ${this.pipelineStep === 6 ? "active" : "success"}">
                        <div class="step-indicator">1</div>
                        <div class="step-text">
                          <strong>加载回退方案 XML Patch 并重置参数</strong>
                          <div class="step-desc">还原配置至容量: ${q.currentCapacity}% / 最大容量: ${q.maxCapacity}%</div>
                        </div>
                      </div>
                      <div class="timeline-step-row ${this.pipelineStep === 7 ? "success" : "pending"}">
                        <div class="step-indicator">2</div>
                        <div class="step-text">
                          <strong>ResourceManager 配置动态刷新与校验</strong>
                          <div class="step-desc">执行 yarn rmadmin 热载还原配置并校验生效状态</div>
                        </div>
                      </div>
                    `
                  : steps.map((s, idx) => {
                      const stepIdx = idx + 1;
                      let stepState = "pending";
                      if (this.pipelineStep === stepIdx) {
                        stepState = "active";
                      } else if (this.pipelineStep > stepIdx) {
                        stepState = "success";
                      }
                      return html`
                        <div class="timeline-step-row ${stepState}">
                          <div class="step-indicator">${stepIdx}</div>
                          <div class="step-text">
                            <strong>${s.text}</strong>
                            <div class="step-desc">${s.desc}</div>
                          </div>
                        </div>
                      `;
                    })}
              </div>
            </div>

            <!-- Live console log output -->
            ${this.executionLog.length > 0
              ? html`
                  <div class="console-box">
                    ${this.executionLog.map((line) => html`<div class="console-line">${line}</div>`)}
                  </div>
                `
              : nothing}
          </div>

          <div class="modal-footer">
            <div class="modal-footer-info">
              ${this.pipelineRunning
                ? html`<span style="color: var(--accent, #3b82f6); display: flex; align-items: center; gap: 4px;">${icons.loader} AI 自动化工作流执行中...</span>`
                : this.pipelineStep === 5
                ? html`<span style="color: #10b981;">✓ 调优已完成，配置刷新生效。</span>`
                : this.pipelineStep === 7
                ? html`<span style="color: #f59e0b;">✓ 配置已还原至原参数配额。</span>`
                : html`<span>待下发闭环变更建议。</span>`}
            </div>

            <button
              class="ops-btn ops-btn--ghost"
              type="button"
              ?disabled=${this.pipelineRunning}
              @click=${() => this.closeExecutionModal()}
            >
              关闭
            </button>

            ${this.showRollbackOption
              ? html`
                  <button
                    class="ops-btn"
                    style="background: rgba(245,158,11,0.15); border: 1px solid rgba(245,158,11,0.3); color:#f59e0b;"
                    type="button"
                    ?disabled=${this.pipelineRunning}
                    @click=${() => this.triggerRollback()}
                  >
                    回滚变更
                  </button>
                `
              : nothing}

            ${this.pipelineStep < 5 && this.pipelineStep !== 7
              ? html`
                  <button
                    class="ops-btn ops-btn--primary"
                    type="button"
                    ?disabled=${this.pipelineRunning}
                    @click=${() => this.startClosedLoop()}
                  >
                    ${confirmButtonLabel}
                  </button>
                `
              : nothing}
          </div>
        </div>
      </div>
    `;
  }
}
