import { html } from "lit";
import type { OpsScenario } from "./scenario-registry.ts";
import { parseWorkbenchObjectScope } from "./workbench-context.ts";
import "../views/ops/bch-flink-diagnosis.ts";
import "../views/ops/bch-spark-governance.ts";
import "../views/ops/bch-fsimage-dashboard.ts";

function renderScenarioSkeleton(scenario: OpsScenario) {
  return html`
    <section class="ops-card" style="margin-top: 14px;">
      <div class="column-header">${scenario.title} · 场景骨架</div>
      <div style="padding:16px;">
        <div class="ops-summary-cards">
          <article class="ops-card stat-card">
            <div class="stat-label">当前状态</div>
            <div class="stat-value info">规划中</div>
            <div class="muted">已进入场景目录，可执行 AI 分析和闭环记录。</div>
          </article>
          <article class="ops-card stat-card">
            <div class="stat-label">待接数据源</div>
            <div class="muted">${scenario.inputs.join(" / ")}</div>
          </article>
          <article class="ops-card stat-card">
            <div class="stat-label">目标输出</div>
            <div class="muted">${scenario.outputs.join(" / ")}</div>
          </article>
        </div>
        <div class="detail-section">
          <div class="detail-section__header">落地说明</div>
          <div class="detail-section__content highlight">
            当前专项先以统一场景模型承载，不承诺未接通的数据自动化。后续接入数据源后，此区域替换为专项详情视图。
          </div>
        </div>
      </div>
    </section>
  `;
}

export function renderScenarioComponent(
  scenario: OpsScenario,
  context: {
    host?: any;
    objectScope?: string;
    timeRange?: string;
  } = {},
) {
  const parsedScope = parseWorkbenchObjectScope(context.objectScope || "all");
  const selectedCluster = parsedScope.kind === "cluster" ? parsedScope.value : "all";
  switch (scenario.id) {
    case "bch-flink-health":
      return html`
        <bch-flink-diagnosis
          .host=${context.host}
          .selectedCluster=${selectedCluster}
          .objectScope=${context.objectScope ?? "all"}
          .timeRange=${context.timeRange ?? "24h"}
        ></bch-flink-diagnosis>
      `;
    case "bch-spark-tuning":
      return html`
        <bch-spark-governance
          .host=${context.host}
          .selectedCluster=${selectedCluster}
          .objectScope=${context.objectScope ?? "all"}
          .timeRange=${context.timeRange ?? "24h"}
        ></bch-spark-governance>
      `;
    case "bch-hdfs-capacity": {
      let activeNamespace = "NS1";
      let activeCluster = "all";
      if (parsedScope.kind === "namespace") {
        activeNamespace = parsedScope.value;
        activeCluster = parsedScope.cluster ?? "all";
      } else if (parsedScope.kind === "directory" && parsedScope.namespace) {
        activeNamespace = parsedScope.namespace;
        activeCluster = parsedScope.cluster ?? "all";
      } else if (parsedScope.kind === "cluster") {
        activeCluster = parsedScope.value;
      }
      return html`
        <bch-fsimage-dashboard
          .host=${context.host}
          .activeCluster=${activeCluster}
          .activeNamespace=${activeNamespace}
          .objectScope=${context.objectScope ?? "all"}
          .timeRange=${context.timeRange ?? "24h"}
        ></bch-fsimage-dashboard>
      `;
    }
    default:
      return renderScenarioSkeleton(scenario);
  }
}
