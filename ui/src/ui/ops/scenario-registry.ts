import type { WorkbenchView } from "../views/workbench.ts";
import type { OpsDomainKey } from "../components/domain-filter.ts";
import type { icons } from "../icons.ts";

export type OpsScenarioMaturity = "planned" | "beta" | "connected" | "automated";
export type OpsScenarioAutomationLevel = "manual" | "recommendation" | "approval" | "closed-loop";
export type OpsScenarioTrigger = "manual" | "schedule" | "alert" | "change";

export type OpsScenario = {
  id: string;
  title: string;
  domain: Exclude<OpsDomainKey, "all">;
  center: WorkbenchView;
  icon: keyof typeof icons;
  summary: string;
  objectTypes: string[];
  triggers: OpsScenarioTrigger[];
  inputs: string[];
  outputs: string[];
  maturity: OpsScenarioMaturity;
  automationLevel: OpsScenarioAutomationLevel;
  owner: string;
  primaryMetric?: string;
  secondaryMetric?: string;
  recommendedActions: string[];
  runbooks: string[];
};

export type WorkbenchTimeRange = "1h" | "24h" | "7d" | "30d";

export type OpsScenarioCatalogStats = {
  total: number;
  domains: Record<string, number>;
  centers: Partial<Record<WorkbenchView, number>>;
  maturity: Record<OpsScenarioMaturity, number>;
};

export type OpsScenarioMaturityFilter = "all" | OpsScenarioMaturity;

export const WORKBENCH_TIME_RANGES: Array<{ id: WorkbenchTimeRange; label: string }> = [
  { id: "1h", label: "近 1 小时" },
  { id: "24h", label: "近 24 小时" },
  { id: "7d", label: "近 7 天" },
  { id: "30d", label: "近 30 天" },
];

export const OPS_SCENARIOS: OpsScenario[] = [
  {
    id: "bch-flink-health",
    title: "Flink 作业健康度",
    domain: "hadoop",
    center: "diagnosis",
    icon: "activity",
    summary: "监控 Flink 实时作业运行状态，定位背压、Checkpoint 失败、OOM 与资源浪费。",
    objectTypes: ["flink_job", "cluster"],
    triggers: ["manual", "schedule", "alert"],
    inputs: ["Flink REST/JMX", "Prometheus 指标", "作业配置", "告警组"],
    outputs: ["健康分", "根因证据", "异常作业列表", "处置建议"],
    maturity: "beta",
    automationLevel: "recommendation",
    owner: "BCH 运维团队",
    primaryMetric: "背压 / Checkpoint / 重启",
    secondaryMetric: "5 分钟级滑动窗口巡检",
    recommendedActions: ["定位背压链路", "检查 Checkpoint 失败原因", "评估 TaskManager 资源配比"],
    runbooks: ["Flink 流量积压溯源 SOP", "Checkpoint 连续失败排查 SOP"],
  },
  {
    id: "bch-spark-tuning",
    title: "Spark 作业调优",
    domain: "hadoop",
    center: "governance",
    icon: "zap",
    summary: "识别 Spark 批作业资源浪费、数据倾斜和慢节点，输出参数调优建议。",
    objectTypes: ["spark_job", "yarn_queue", "cluster"],
    triggers: ["manual", "schedule"],
    inputs: ["Spark History", "YARN 指标", "任务运行历史", "作业配置"],
    outputs: ["调优候选", "参数建议", "资源节省预估", "治理任务"],
    maturity: "beta",
    automationLevel: "recommendation",
    owner: "BCH 运维团队",
    primaryMetric: "资源浪费 / 数据倾斜",
    secondaryMetric: "近 24h 完成任务画像",
    recommendedActions: ["识别数据倾斜 stage", "生成 Spark 参数调优建议", "评估可释放 Core 闲置算力"],
    runbooks: ["Spark 数据倾斜治理 SOP", "Spark Executor 参数调优 SOP"],
  },
  {
    id: "bch-hdfs-capacity",
    title: "HDFS 容量调优",
    domain: "hadoop",
    center: "capacity",
    icon: "server",
    summary: "分析 HDFS FSImage、小文件、目录深度和容量水位，定位存储治理机会。",
    objectTypes: ["hdfs_namespace", "hdfs_directory", "cluster"],
    triggers: ["manual", "schedule"],
    inputs: ["FSImage", "NameNode JMX", "容量水位", "目录元数据"],
    outputs: ["容量风险", "小文件治理建议", "目录热点", "扩容/清理建议"],
    maturity: "beta",
    automationLevel: "recommendation",
    owner: "BCH 运维团队",
    primaryMetric: "容量 / 小文件 / 目录深度",
    secondaryMetric: "离线 FSImage 深度分析",
    recommendedActions: ["扫描小文件热点目录", "识别 Trash 未清理空间", "生成 namespace 清理建议"],
    runbooks: ["HDFS 小文件治理 SOP", "HDFS namespace 容量巡检 SOP"],
  },
  {
    id: "gbase-slow-sql",
    title: "GBase 慢 SQL 诊断",
    domain: "gbase",
    center: "diagnosis",
    icon: "database",
    summary: "分析慢 SQL、执行计划、索引命中和等待事件，定位数据库性能瓶颈。",
    objectTypes: ["database_instance", "sql", "session"],
    triggers: ["manual", "alert"],
    inputs: ["慢 SQL 样本", "执行计划", "会话等待", "实例指标"],
    outputs: ["根因候选", "SQL 优化建议", "索引建议", "影响面"],
    maturity: "planned",
    automationLevel: "recommendation",
    owner: "GBase 运维团队",
    primaryMetric: "慢 SQL / 执行计划 / 等待事件",
    secondaryMetric: "待接入数据库观测数据源",
    recommendedActions: ["采集慢 SQL 样本", "对比执行计划", "检查索引和锁等待"],
    runbooks: ["GBase 慢 SQL 诊断 SOP", "GBase 执行计划分析 SOP"],
  },
  {
    id: "gbase-capacity-watermark",
    title: "GBase 容量水位",
    domain: "gbase",
    center: "capacity",
    icon: "usageBars",
    summary: "跟踪实例存储、水位趋势和热点表膨胀，输出扩容和清理建议。",
    objectTypes: ["database_instance", "table", "tablespace"],
    triggers: ["manual", "schedule", "alert"],
    inputs: ["实例容量", "表空间水位", "增长趋势", "热点表"],
    outputs: ["容量风险", "扩容建议", "清理候选", "趋势说明"],
    maturity: "planned",
    automationLevel: "recommendation",
    owner: "GBase 运维团队",
    primaryMetric: "实例容量 / 表空间 / 热点表",
    secondaryMetric: "待接入容量趋势数据",
    recommendedActions: ["识别高水位表空间", "评估增长趋势", "生成扩容或清理建议"],
    runbooks: ["GBase 容量巡检 SOP", "GBase 表空间治理 SOP"],
  },
  {
    id: "gbase-lock-wait",
    title: "GBase 锁等待诊断",
    domain: "gbase",
    center: "diagnosis",
    icon: "alertTriangle",
    summary: "分析会话阻塞链路、锁等待事件和长事务，定位数据库并发瓶颈和业务影响面。",
    objectTypes: ["database_instance", "session", "transaction"],
    triggers: ["manual", "alert"],
    inputs: ["锁等待事件", "阻塞会话", "事务快照", "实例指标"],
    outputs: ["阻塞链路", "影响会话", "处置建议", "回滚风险"],
    maturity: "planned",
    automationLevel: "recommendation",
    owner: "GBase 运维团队",
    primaryMetric: "锁等待 / 阻塞链路 / 长事务",
    secondaryMetric: "待接入会话与事务快照",
    recommendedActions: ["识别头部阻塞会话", "评估长事务回滚风险", "生成 kill/等待/业务确认建议"],
    runbooks: ["GBase 锁等待诊断 SOP", "GBase 长事务处置 SOP"],
  },
  {
    id: "fi-service-health",
    title: "FI 服务健康巡检",
    domain: "fi",
    center: "inspection",
    icon: "building",
    summary: "围绕 FI 服务、组件和节点状态执行健康巡检，识别服务降级和依赖异常。",
    objectTypes: ["fi_service", "component", "node"],
    triggers: ["manual", "schedule", "alert"],
    inputs: ["服务状态", "组件指标", "节点健康", "告警组"],
    outputs: ["巡检摘要", "风险项", "处置建议", "服务影响"],
    maturity: "planned",
    automationLevel: "recommendation",
    owner: "FI 运维团队",
    primaryMetric: "服务 / 组件 / 节点健康",
    secondaryMetric: "待接入 FI Manager 数据",
    recommendedActions: ["检查服务降级组件", "定位异常节点", "确认依赖服务影响"],
    runbooks: ["FI 服务健康巡检 SOP", "FI 组件异常处置 SOP"],
  },
  {
    id: "fi-component-diagnosis",
    title: "FI 组件异常诊断",
    domain: "fi",
    center: "diagnosis",
    icon: "settings",
    summary: "围绕 FI 核心组件、依赖服务和节点指标定位异常来源，输出恢复建议。",
    objectTypes: ["component", "fi_service", "node"],
    triggers: ["manual", "alert"],
    inputs: ["组件状态", "依赖探测", "节点指标", "告警组"],
    outputs: ["异常组件", "依赖影响", "恢复建议", "升级路径"],
    maturity: "planned",
    automationLevel: "recommendation",
    owner: "FI 运维团队",
    primaryMetric: "组件状态 / 依赖探测 / 节点负载",
    secondaryMetric: "待接入 FI Manager 组件视图",
    recommendedActions: ["定位异常组件实例", "检查上游依赖可用性", "生成重启或扩容建议"],
    runbooks: ["FI 组件异常诊断 SOP", "FI 依赖服务恢复 SOP"],
  },
  {
    id: "governance-metadata-lineage",
    title: "元数据血缘影响分析",
    domain: "governance",
    center: "governance",
    icon: "layout",
    summary: "分析元数据变更、血缘链路和影响面，支撑治理整改和变更评估。",
    objectTypes: ["metadata_asset", "lineage", "owner_team"],
    triggers: ["manual", "change"],
    inputs: ["元数据资产", "血缘关系", "变更记录", "责任团队"],
    outputs: ["影响面", "治理建议", "责任对象", "整改优先级"],
    maturity: "planned",
    automationLevel: "recommendation",
    owner: "开发治理团队",
    primaryMetric: "元数据 / 血缘 / 影响面",
    secondaryMetric: "待接入元数据平台",
    recommendedActions: ["识别血缘下游影响", "定位责任团队", "生成治理整改建议"],
    runbooks: ["元数据血缘影响分析 SOP", "元数据治理整改 SOP"],
  },
  {
    id: "governance-config-compliance",
    title: "配置合规扫描",
    domain: "governance",
    center: "governance",
    icon: "checkCircle",
    summary: "扫描任务、表、权限和发布配置的合规风险，生成整改优先级和责任清单。",
    objectTypes: ["config_item", "metadata_asset", "owner_team"],
    triggers: ["manual", "schedule", "change"],
    inputs: ["配置基线", "发布记录", "权限策略", "资产元数据"],
    outputs: ["违规项", "影响范围", "整改建议", "责任团队"],
    maturity: "planned",
    automationLevel: "recommendation",
    owner: "开发治理团队",
    primaryMetric: "配置基线 / 权限 / 发布合规",
    secondaryMetric: "待接入配置治理数据源",
    recommendedActions: ["比对配置基线", "识别高风险权限", "生成整改任务清单"],
    runbooks: ["配置合规扫描 SOP", "权限风险整改 SOP"],
  },
  {
    id: "dataapps-sla-escort",
    title: "数据 App SLA 护航",
    domain: "dataapps",
    center: "change",
    icon: "activity",
    summary: "围绕数据应用链路、调度任务和 SLA 执行变更前后护航。",
    objectTypes: ["data_app", "schedule_chain", "dataset", "sla"],
    triggers: ["manual", "schedule", "change", "alert"],
    inputs: ["调度链路", "任务状态", "SLA 目标", "数据集依赖"],
    outputs: ["SLA 风险", "链路影响面", "护航建议", "回滚点"],
    maturity: "planned",
    automationLevel: "approval",
    owner: "数据 App 运维团队",
    primaryMetric: "SLA / 调度链路 / 数据集依赖",
    secondaryMetric: "待接入调度和 SLA 数据",
    recommendedActions: ["检查关键链路任务", "评估 SLA 风险", "生成变更护航清单"],
    runbooks: ["数据 App SLA 护航 SOP", "调度链路异常处置 SOP"],
  },
  {
    id: "dataapps-schedule-failure",
    title: "调度失败诊断",
    domain: "dataapps",
    center: "diagnosis",
    icon: "activity",
    summary: "分析调度失败、依赖阻塞和数据集延迟，定位数据应用链路中断原因。",
    objectTypes: ["schedule_task", "schedule_chain", "dataset"],
    triggers: ["manual", "alert"],
    inputs: ["调度任务状态", "依赖链路", "失败日志", "数据集产出"],
    outputs: ["失败根因", "阻塞任务", "影响 SLA", "恢复建议"],
    maturity: "planned",
    automationLevel: "recommendation",
    owner: "数据 App 运维团队",
    primaryMetric: "调度失败 / 依赖阻塞 / SLA 延迟",
    secondaryMetric: "待接入调度平台事件",
    recommendedActions: ["定位首个失败任务", "检查上游依赖阻塞", "生成补数和恢复建议"],
    runbooks: ["调度失败诊断 SOP", "数据 App 链路恢复 SOP"],
  },
];

export function scenariosForWorkbench(domain: OpsDomainKey, center: WorkbenchView): OpsScenario[] {
  if (domain === "all") {
    return OPS_SCENARIOS.filter((scenario) => scenario.center === center);
  }
  return OPS_SCENARIOS.filter((scenario) => scenario.domain === domain && scenario.center === center);
}

export function findWorkbenchScenario(id: string | null | undefined): OpsScenario | undefined {
  if (!id) {
    return undefined;
  }
  return OPS_SCENARIOS.find((scenario) => scenario.id === id);
}

export function defaultScenarioForWorkbench(domain: OpsDomainKey, center: WorkbenchView): OpsScenario | undefined {
  return scenariosForWorkbench(domain, center)[0];
}

export function filterWorkbenchScenarios(
  scenarios: OpsScenario[],
  query: string | null | undefined,
  maturity: OpsScenarioMaturityFilter = "all",
): OpsScenario[] {
  const q = (query ?? "").trim().toLowerCase();
  return scenarios.filter((scenario) => {
    if (maturity !== "all" && scenario.maturity !== maturity) {
      return false;
    }
    if (!q) {
      return true;
    }
    const haystack = [
      scenario.id,
      scenario.title,
      scenario.domain,
      scenario.center,
      scenario.summary,
      scenario.owner,
      scenario.primaryMetric,
      scenario.secondaryMetric,
      ...scenario.objectTypes,
      ...scenario.inputs,
      ...scenario.outputs,
      ...scenario.recommendedActions,
      ...scenario.runbooks,
    ]
      .filter(Boolean)
      .join(" ")
      .toLowerCase();
    return haystack.includes(q);
  });
}

export function scenarioCatalogStats(domain: OpsDomainKey = "all"): OpsScenarioCatalogStats {
  const scenarios = domain === "all" ? OPS_SCENARIOS : OPS_SCENARIOS.filter((scenario) => scenario.domain === domain);
  const stats: OpsScenarioCatalogStats = {
    total: scenarios.length,
    domains: {},
    centers: {},
    maturity: {
      planned: 0,
      beta: 0,
      connected: 0,
      automated: 0,
    },
  };

  for (const scenario of scenarios) {
    stats.domains[scenario.domain] = (stats.domains[scenario.domain] ?? 0) + 1;
    stats.centers[scenario.center] = (stats.centers[scenario.center] ?? 0) + 1;
    stats.maturity[scenario.maturity] += 1;
  }

  return stats;
}
