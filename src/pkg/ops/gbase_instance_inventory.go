package ops

// GBaseInstance represents one GBase database instance for batch L0.
type GBaseInstance struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Cluster           string  `json:"cluster"`
	ActiveConnections int     `json:"activeConnections"`
	MaxConnections    int     `json:"maxConnections"`
	SlowSQLCount      int     `json:"slowSqlCount"`
	QPS               float64 `json:"qps"`
	Status            string  `json:"status"`
}

// ListGBaseInstances returns registered GBase instances (mock inventory).
func ListGBaseInstances() ([]GBaseInstance, error) {
	return []GBaseInstance{
		{
			ID: "gbase-prod-primary", Name: "GBase 生产主库", Cluster: "cluster-gbase-prod",
			ActiveConnections: 42, MaxConnections: 200, SlowSQLCount: 2, QPS: 1250, Status: "healthy",
		},
		{
			ID: "gbase-prod-standby", Name: "GBase 生产备库", Cluster: "cluster-gbase-prod",
			ActiveConnections: 18, MaxConnections: 200, SlowSQLCount: 0, QPS: 420, Status: "healthy",
		},
		{
			ID: "gbase-report-01", Name: "GBase 报表库", Cluster: "cluster-gbase-prod",
			ActiveConnections: 178, MaxConnections: 200, SlowSQLCount: 24, QPS: 890, Status: "warning",
		},
	}, nil
}
