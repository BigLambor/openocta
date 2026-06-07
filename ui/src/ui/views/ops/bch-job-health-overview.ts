import { html, nothing, type TemplateResult } from "lit";

export type JobHealthBucket = {
  healthy: number;
  warning: number;
  critical: number;
  waste: number;
  total: number;
};

export type JobRadarScores = {
  stability: number;
  efficiency: number;
  performance: number;
};

export type JobHealthOverviewProps = {
  title?: string;
  jobKind: "Flink" | "Spark";
  clusterLabel?: string;
  buckets: JobHealthBucket;
  radar: JobRadarScores;
  agentSummary: string | TemplateResult;
  stabilityBaseline?: "normal" | "warning" | "critical";
  performanceBaseline?: "normal" | "warning" | "critical";
};

function baselineLabel(state: "normal" | "warning" | "critical"): string {
  switch (state) {
    case "warning":
      return "波动";
    case "critical":
      return "异常";
    default:
      return "正常";
  }
}

function baselineClass(state: "normal" | "warning" | "critical"): string {
  switch (state) {
    case "warning":
      return "warning";
    case "critical":
      return "critical";
    default:
      return "healthy";
  }
}

function radarPoint(cx: number, cy: number, r: number, angleDeg: number, norm: number): string {
  const rad = ((angleDeg - 90) * Math.PI) / 180;
  const x = cx + r * norm * Math.cos(rad);
  const y = cy + r * norm * Math.sin(rad);
  return `${x},${y}`;
}

export function renderBchJobHealthOverview(props: JobHealthOverviewProps) {
  const { buckets, radar } = props;
  const cx = 180;
  const cy = 145;
  const r = 96;
  const sNorm = Math.min(1, Math.max(0, radar.stability / 100));
  const eNorm = Math.min(1, Math.max(0, radar.efficiency / 100));
  const pNorm = Math.min(1, Math.max(0, radar.performance / 100));
  const poly = [
    radarPoint(cx, cy, r, -90, sNorm),
    radarPoint(cx, cy, r, 150, eNorm),
    radarPoint(cx, cy, r, 30, pNorm),
  ].join(" ");

  const stabilityBaseline =
    props.stabilityBaseline ?? (radar.stability >= 80 ? "normal" : radar.stability >= 60 ? "warning" : "critical");
  const performanceBaseline =
    props.performanceBaseline ?? (radar.performance >= 80 ? "normal" : radar.performance >= 60 ? "warning" : "critical");

  const outer = [
    radarPoint(cx, cy, r, -90, 1),
    radarPoint(cx, cy, r, 150, 1),
    radarPoint(cx, cy, r, 30, 1),
  ].join(" ");
  const mid = [
    radarPoint(cx, cy, r * 0.66, -90, 1),
    radarPoint(cx, cy, r * 0.66, 150, 1),
    radarPoint(cx, cy, r * 0.66, 30, 1),
  ].join(" ");
  const inner = [
    radarPoint(cx, cy, r * 0.33, -90, 1),
    radarPoint(cx, cy, r * 0.33, 150, 1),
    radarPoint(cx, cy, r * 0.33, 30, 1),
  ].join(" ");

  return html`
    <style>
      .bch-overview {
        margin-bottom: 22px;
      }

      .bch-overview__title {
        font-size: 16px;
        font-weight: 700;
        margin: 0 0 16px;
        color: var(--text-primary);
        letter-spacing: 0.01em;
      }

      .bch-overview__cards {
        display: grid;
        grid-template-columns: repeat(4, minmax(0, 1fr));
        gap: 16px;
        margin-bottom: 16px;
      }

      .bch-overview__card {
        background: #fff;
        border: 1px solid #e8ebf0;
        border-radius: 12px;
        padding: 18px 20px 20px;
        box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
        min-height: 96px;
        display: flex;
        flex-direction: column;
        justify-content: center;
      }

      :root[data-theme="dark"] .bch-overview__card {
        background: color-mix(in srgb, var(--bg-content) 92%, #fff);
        border-color: var(--border);
      }

      .bch-overview__card--highlight {
        background: linear-gradient(180deg, #edf4ff 0%, #f7faff 100%);
        border-color: #c9daf8;
      }

      :root[data-theme="dark"] .bch-overview__card--highlight {
        background: linear-gradient(180deg, rgba(59, 130, 246, 0.16), rgba(59, 130, 246, 0.06));
        border-color: rgba(59, 130, 246, 0.28);
      }

      .bch-overview__card-label {
        font-size: 12px;
        color: #6b7280;
        margin-bottom: 10px;
        line-height: 1.4;
      }

      .bch-overview__card-value {
        font-size: 34px;
        font-weight: 700;
        line-height: 1;
        letter-spacing: -0.02em;
      }

      .bch-overview__card-value.healthy { color: #10b981; }
      .bch-overview__card-value.warning { color: #f59e0b; }
      .bch-overview__card-value.critical { color: #ef4444; }
      .bch-overview__card-value.info { color: #2563eb; }

      .bch-overview__card-unit {
        font-size: 14px;
        font-weight: 600;
        color: #2563eb;
        margin-left: 6px;
      }

      .bch-overview__panels {
        display: grid;
        grid-template-columns: minmax(0, 1.08fr) minmax(0, 1fr);
        gap: 16px;
      }

      .bch-overview__panel {
        background: #fff;
        border: 1px solid #e8ebf0;
        border-radius: 12px;
        padding: 18px 20px 20px;
        min-height: 328px;
        box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
      }

      :root[data-theme="dark"] .bch-overview__panel {
        background: color-mix(in srgb, var(--bg-content) 92%, #fff);
        border-color: var(--border);
      }

      .bch-overview__panel-title {
        font-size: 13px;
        font-weight: 600;
        color: #374151;
        margin-bottom: 8px;
      }

      :root[data-theme="dark"] .bch-overview__panel-title {
        color: var(--text-secondary);
      }

      .bch-overview__radar-wrap {
        display: flex;
        align-items: center;
        justify-content: center;
        min-height: 286px;
      }

      .bch-overview__radar-label {
        font-size: 14px;
        fill: #6b7280;
        font-weight: 600;
      }

      .bch-overview__summary {
        font-size: 13px;
        line-height: 1.85;
        color: #4b5563;
        margin: 8px 0 18px;
      }

      :root[data-theme="dark"] .bch-overview__summary {
        color: var(--text-secondary);
      }

      .bch-overview__summary strong {
        color: #111827;
        font-weight: 700;
      }

      :root[data-theme="dark"] .bch-overview__summary strong {
        color: var(--text-primary);
      }

      .bch-overview__summary-link {
        color: #2563eb;
        text-decoration: none;
        font-weight: 600;
      }

      .bch-overview__baselines {
        display: flex;
        flex-direction: column;
        gap: 10px;
        padding-top: 4px;
      }

      .bch-overview__baseline {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 12px;
        color: #4b5563;
      }

      .bch-overview__dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        flex-shrink: 0;
      }

      .bch-overview__dot.healthy { background: #10b981; }
      .bch-overview__dot.warning { background: #f59e0b; }
      .bch-overview__dot.critical { background: #ef4444; }

      @media (max-width: 1100px) {
        .bch-overview__cards,
        .bch-overview__panels {
          grid-template-columns: 1fr;
        }
      }
    </style>

    <section class="bch-overview">
      ${props.title ? html`<h3 class="bch-overview__title">${props.title}</h3>` : nothing}

      <div class="bch-overview__cards">
        ${props.jobKind === "Spark"
          ? html`
              <div class="bch-overview__card">
                <div class="bch-overview__card-label">已分析作业</div>
                <div class="bch-overview__card-value info">${buckets.total}</div>
              </div>
              <div class="bch-overview__card">
                <div class="bch-overview__card-label">正常结束</div>
                <div class="bch-overview__card-value healthy">${buckets.healthy}</div>
              </div>
              <div class="bch-overview__card">
                <div class="bch-overview__card-label">运行失败</div>
                <div class="bch-overview__card-value critical">${buckets.critical}</div>
              </div>
              <div class="bch-overview__card bch-overview__card--highlight">
                <div class="bch-overview__card-label">待调优作业</div>
                <div class="bch-overview__card-value info">
                  ${buckets.waste}<span class="bch-overview__card-unit">Jobs</span>
                </div>
              </div>
            `
          : html`
              <div class="bch-overview__card">
                <div class="bch-overview__card-label">健康作业 (Health ≥ 90)</div>
                <div class="bch-overview__card-value healthy">${buckets.healthy}</div>
              </div>
              <div class="bch-overview__card">
                <div class="bch-overview__card-label">亚健康 (Health 60-89)</div>
                <div class="bch-overview__card-value warning">${buckets.warning}</div>
              </div>
              <div class="bch-overview__card">
                <div class="bch-overview__card-label">高危作业 (Health &lt; 60)</div>
                <div class="bch-overview__card-value critical">${buckets.critical}</div>
              </div>
              <div class="bch-overview__card bch-overview__card--highlight">
                <div class="bch-overview__card-label">识别资源浪费</div>
                <div class="bch-overview__card-value info">
                  ${buckets.waste}<span class="bch-overview__card-unit">Jobs</span>
                </div>
              </div>
            `}
      </div>

      <div class="bch-overview__panels">
        <div class="bch-overview__panel">
          <div class="bch-overview__panel-title">集群多维评估均值</div>
          <div class="bch-overview__radar-wrap">
            <svg width="100%" height="300" viewBox="0 0 360 300" aria-label="集群多维评估雷达图">
              <polygon points="${outer}" fill="none" stroke="#e5e7eb" stroke-width="1" />
              <polygon points="${mid}" fill="none" stroke="#eceff3" stroke-width="1" />
              <polygon points="${inner}" fill="none" stroke="#f1f3f6" stroke-width="1" />
              <line x1="${cx}" y1="${cy}" x2="${radarPoint(cx, cy, r, -90, 1).split(",")[0]}" y2="${radarPoint(cx, cy, r, -90, 1).split(",")[1]}" stroke="#e5e7eb" />
              <line x1="${cx}" y1="${cy}" x2="${radarPoint(cx, cy, r, 150, 1).split(",")[0]}" y2="${radarPoint(cx, cy, r, 150, 1).split(",")[1]}" stroke="#e5e7eb" />
              <line x1="${cx}" y1="${cy}" x2="${radarPoint(cx, cy, r, 30, 1).split(",")[0]}" y2="${radarPoint(cx, cy, r, 30, 1).split(",")[1]}" stroke="#e5e7eb" />
              <polygon points="${poly}" fill="rgba(37, 99, 235, 0.18)" stroke="#3b82f6" stroke-width="2" />
              <text x="${cx}" y="28" text-anchor="middle" class="bch-overview__radar-label">稳定性</text>
              <text x="58" y="274" text-anchor="middle" class="bch-overview__radar-label">效率</text>
              <text x="302" y="274" text-anchor="middle" class="bch-overview__radar-label">性能</text>
            </svg>
          </div>
        </div>

        <div class="bch-overview__panel">
          <div class="bch-overview__panel-title">智能体运作摘要</div>
          <div class="bch-overview__summary">${props.agentSummary}</div>
          <div class="bch-overview__baselines">
            <div class="bch-overview__baseline">
              <span class="bch-overview__dot ${baselineClass(stabilityBaseline)}"></span>
              <span>稳定性基准: ${baselineLabel(stabilityBaseline)}</span>
            </div>
            <div class="bch-overview__baseline">
              <span class="bch-overview__dot ${baselineClass(performanceBaseline)}"></span>
              <span>性能基准: ${baselineLabel(performanceBaseline)}</span>
            </div>
          </div>
        </div>
      </div>
    </section>
  `;
}

export function bucketJobsByScore(
  scores: number[],
  wasteFlags: boolean[] = [],
): JobHealthBucket {
  let healthy = 0;
  let warning = 0;
  let critical = 0;
  let waste = 0;
  scores.forEach((score, idx) => {
    if (score >= 90) healthy++;
    else if (score >= 60) warning++;
    else critical++;
    if (wasteFlags[idx]) waste++;
  });
  return { healthy, warning, critical, waste, total: scores.length };
}

export function averageRadarFromFlinkJobs(
  jobs: Array<{ sScore: number; pScore: number; eScore: number }>,
): JobRadarScores {
  if (jobs.length === 0) {
    return { stability: 0, efficiency: 0, performance: 0 };
  }
  const sum = jobs.reduce(
    (acc, job) => ({
      stability: acc.stability + job.sScore,
      efficiency: acc.efficiency + job.eScore,
      performance: acc.performance + job.pScore,
    }),
    { stability: 0, efficiency: 0, performance: 0 },
  );
  const n = jobs.length;
  return {
    stability: Math.round(sum.stability / n),
    efficiency: Math.round(sum.efficiency / n),
    performance: Math.round(sum.performance / n),
  };
}

export function sparkHealthScore(job: {
  status: string;
  labels: string[];
  metrics: { failedTasks: number; totalTasks: number; cpuSkewRatio: number };
}): number {
  if (job.status === "FAILED") return 48;
  if (job.labels.some((l) => l.includes("OOM") || l.includes("内存溢出"))) return 52;
  if (job.labels.some((l) => l.includes("倾斜"))) return 68;
  if (job.labels.some((l) => l.includes("长尾"))) return 76;
  if (job.metrics.failedTasks > 0) return 62;
  if (job.metrics.cpuSkewRatio >= 4) return 74;
  if (job.status === "RUNNING") return 82;
  return 93;
}

export function averageRadarFromSparkJobs(
  jobs: Array<{
    status: string;
    labels: string[];
    metrics: { failedTasks: number; totalTasks: number; cpuSkewRatio: number; memorySkewRatio: number };
  }>,
): JobRadarScores {
  if (jobs.length === 0) {
    return { stability: 0, efficiency: 0, performance: 0 };
  }
  const scores = jobs.map((job) => {
    const failRate = job.metrics.totalTasks > 0 ? job.metrics.failedTasks / job.metrics.totalTasks : 0;
    const stability = Math.max(0, 100 - failRate * 100 - (job.status === "FAILED" ? 35 : 0));
    const efficiency = Math.max(
      0,
      100 - Math.min(40, job.metrics.cpuSkewRatio * 6) - (job.labels.some((l) => l.includes("资源过度配置")) ? 18 : 0),
    );
    const performance = Math.max(0, sparkHealthScore(job));
    return { stability, efficiency, performance };
  });
  return {
    stability: Math.round(scores.reduce((s, v) => s + v.stability, 0) / scores.length),
    efficiency: Math.round(scores.reduce((s, v) => s + v.efficiency, 0) / scores.length),
    performance: Math.round(scores.reduce((s, v) => s + v.performance, 0) / scores.length),
  };
}
