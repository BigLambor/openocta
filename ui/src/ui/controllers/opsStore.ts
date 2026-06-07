import { ReactiveController, ReactiveControllerHost } from "lit";
import type { OpsClusterRecord, OpsHealthSnapshot, OpsDashboardSummary } from "./ops-clusters.ts";
import type { OpsInspectionIMStatus } from "./ops-inspection.ts";
import type { FlinkJob, SparkJob } from "./bch-client.ts";
import type { BchDomainScenarioSummary } from "./bch-scenario-summary.ts";
import type { DashboardAlertHighlight, DashboardInspectionRun } from "./ops-dashboard-feed.ts";
import type { mapAlertGroupForUI } from "./ops-alerts.ts";

export class OpsStore implements ReactiveController {
  host: ReactiveControllerHost;

  opsActiveSubTabs: Record<
    string,
    | "overview"
    | "assetTopology"
    | "observability"
    | "inspection"
    | "jobGovernance"
    | "diagnosis"
    | "governance"
    | "capacity"
    | "change"
    | "employees"
  > = {
    hadoop: "overview",
    fi: "overview",
    gbase: "overview",
    governance: "overview",
    dataapps: "overview",
  };

  opsSelectedAlertGroupIds: Record<string, string | null> = {
    hadoop: null,
    fi: null,
    gbase: null,
    governance: null,
    dataapps: null,
  };

  opsSelectedInspectionIds: Record<string, string | null> = {
    hadoop: null,
    fi: null,
    gbase: null,
    governance: null,
    dataapps: null,
  };

  opsAlertsByDomain: Record<string, ReturnType<typeof mapAlertGroupForUI>[]> = {};

  opsAlertsStats: Record<
    string,
    { originalTotal: number; reductionRate: number; mergedTotal: number }
  > = {};

  opsAlertsLoading: Record<string, boolean> = {};
  opsAlertsError: Record<string, string | null> = {};
  opsInspectionImStatus: OpsInspectionIMStatus | null = null;

  opsIsInspecting: Record<string, boolean> = {
    hadoop: false,
    fi: false,
    gbase: false,
    governance: false,
    dataapps: false,
  };

  opsSelectedEntityIds: Record<string, string> = {
    hadoop: "all",
    fi: "all",
    gbase: "all",
    governance: "all",
    dataapps: "all",
  };

  opsDomainClusters: Record<string, OpsClusterRecord[]> = {};
  opsDomainClustersLoading: Record<string, boolean> = {};

  opsEntitySelectorOpen: Record<string, boolean> = {
    hadoop: false,
    fi: false,
    gbase: false,
    governance: false,
    dataapps: false,
  };

  opsClusters: OpsClusterRecord[] = [];
  opsClustersLoading = false;
  opsClustersError: string | null = null;

  opsHealthSnapshots: OpsHealthSnapshot[] = [];
  opsHealthSnapshotsLoading = false;
  opsHealthSnapshotsError: string | null = null;

  opsFlinkJobs: FlinkJob[] = [];
  opsFlinkJobsLoading = false;

  opsSparkJobs: SparkJob[] = [];
  opsSparkJobsLoading = false;

  opsDashboardSummary: OpsDashboardSummary | null = null;
  opsDashboardLoading = false;
  opsDashboardError: string | null = null;

  opsBchScenarioSummary: BchDomainScenarioSummary | null = null;
  opsBchScenarioSummaryLoading = false;
  opsBchScenarioSummaryError: string | null = null;

  opsGlobalInspecting = false;
  opsDashboardToast: string | null = null;
  opsDashboardFeedLoading = false;
  opsDashboardFeedError: string | null = null;
  opsDashboardAlertHighlights: DashboardAlertHighlight[] = [];
  opsDashboardDomainPendingAlerts: Record<string, number> = {};
  opsDashboardRecentInspections: DashboardInspectionRun[] = [];

  constructor(host: ReactiveControllerHost) {
    this.host = host;
    host.addController(this);
  }

  hostConnected() {}
  hostDisconnected() {}
}
