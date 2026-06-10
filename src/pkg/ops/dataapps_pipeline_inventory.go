package ops

// DataAppPipeline represents one data application pipeline for batch L0.
type DataAppPipeline struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Cluster      string `json:"cluster"`
	Status       string `json:"status"`
	SLABreach    bool   `json:"slaBreach"`
	FailedTasks  int    `json:"failedTasks"`
	DelaySeconds int    `json:"delaySeconds"`
	Owner        string `json:"owner"`
}

// ListDataAppPipelines returns registered pipelines (mock inventory).
func ListDataAppPipelines() ([]DataAppPipeline, error) {
	return []DataAppPipeline{
		{
			ID: "pipeline_daily_billing", Name: "日终账单链路", Cluster: "cluster-dataapp-scheduler",
			Status: "ok", SLABreach: false, FailedTasks: 0, DelaySeconds: 120, Owner: "wang.wu",
		},
		{
			ID: "pipeline_user_cohort", Name: "用户留存报表", Cluster: "cluster-dataapp-scheduler",
			Status: "delayed", SLABreach: true, FailedTasks: 0, DelaySeconds: 5400, Owner: "zhao.liu",
		},
		{
			ID: "pipeline_click_heat", Name: "点击热力图跑批", Cluster: "cluster-dataapp-scheduler",
			Status: "failed", SLABreach: true, FailedTasks: 3, DelaySeconds: 0, Owner: "sun.qi",
		},
		{
			ID: "pipeline_risk_sync", Name: "风控指标同步", Cluster: "cluster-dataapp-scheduler",
			Status: "ok", SLABreach: false, FailedTasks: 0, DelaySeconds: 60, Owner: "tom",
		},
	}, nil
}
