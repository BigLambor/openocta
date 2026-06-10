package ops

import (
	"encoding/json"
	"fmt"
	"sync"
)

// MockBchService implements BchService.
type MockBchService struct{}

func NewMockBchService() BchService {
	return &MockBchService{}
}

func (s *MockBchService) GetClustersHealth() ([]BchClusterHealth, error) {
	clusters := []BchClusterHealth{
		{
			ID:             "cluster-prod-a",
			Name:           "哈池 BCH 生产集群 A (prod-a)",
			Region:         "哈池",
			Status:         "healthy",
			Score:          98,
			ScoreStatus:    "ok",
			Coverage:       1.0,
			Freshness:      "ok",
			PresentSources: []string{"gbase_sql", "metrics"},
			MissingSources: []string{},
			NodeCount:      120,
			ActiveAlerts:   0,
			CpuUsedPercent: 62.5,
			MemUsedPercent: 78.2,
			DfsUsedPercent: 54.1,
			Metrics: map[string]interface{}{
				"activeNodes":      120,
				"decommission":     0,
				"totalBlocks":      91699026,
				"activeContainers": 1840,
			},
		},
		{
			ID:             "cluster-prod-b",
			Name:           "呼池 BCH 生产集群 A (prod-b)",
			Region:         "呼池",
			Status:         "warning",
			Score:          82,
			ScoreStatus:    "warning",
			Coverage:       1.0,
			Freshness:      "ok",
			PresentSources: []string{"gbase_sql", "metrics"},
			MissingSources: []string{},
			NodeCount:      80,
			ActiveAlerts:   2,
			CpuUsedPercent: 88.0,
			MemUsedPercent: 91.5,
			DfsUsedPercent: 81.3,
			Metrics: map[string]interface{}{
				"activeNodes":      78,
				"decommission":     2,
				"totalBlocks":      131486447,
				"activeContainers": 2560,
			},
		},
	}

	// Augment with real HealthSnapshot if available
	for i, c := range clusters {
		if snap, ok := GetHealthSnapshot(c.ID); ok {
			if snap.Score != nil {
				clusters[i].Score = *snap.Score
			}
			clusters[i].ScoreStatus = snap.ScoreStatus
			clusters[i].Coverage = snap.Coverage
			clusters[i].MissingSources = snap.MissingSources
			clusters[i].PresentSources = snap.PresentSources

			// If the snapshot has signals, we can compute freshness overall
			// For simplicity, we just use the global snapshot freshness check
			freshness := "ok"
			for _, sig := range snap.Signals {
				if sig.Freshness == "expired" {
					freshness = "expired"
					break
				}
			}
			clusters[i].Freshness = freshness

			// map status to our UI logic
			switch snap.ScoreStatus {
			case ScoreStatusCritical, ScoreStatusDegraded:
				clusters[i].Status = "critical"
			case ScoreStatusWarning, ScoreStatusPartial:
				clusters[i].Status = "warning"
			case ScoreStatusOK:
				clusters[i].Status = "healthy"
			}
		}
	}

	return clusters, nil
}

func (s *MockBchService) ListFlinkJobs() ([]FlinkJob, error) {
	return []FlinkJob{
		ComputeFlinkJobAnalysis("job_tx_core", "交易核心链路 (Trade_Analysis)", "cui.chao", "prod-a", FlinkMetricInput{LagTrend: 0, MaxLag: 10, AvgLag: 8, CpuMax: 60, CpuAvg: 55, HeapMax: 60}),
		ComputeFlinkJobAnalysis("job_log_sink", "日志归档 (Log_ES_Sink)", "lu.yang", "prod-a", FlinkMetricInput{LagTrend: 500, MaxLag: 8000, AvgLag: 7800, IsBP: true, CpuMax: 25, CpuAvg: 15, HeapMax: 40}),
		ComputeFlinkJobAnalysis("job_risk_calc", "风控实时计算 (Risk_Model)", "tom", "prod-b", FlinkMetricInput{LagTrend: 800, MaxLag: 12000, AvgLag: 11500, IsBP: true, CpuMax: 98, CpuAvg: 95, HeapMax: 70}),
		ComputeFlinkJobAnalysis("job_user_tag", "用户画像流 (User_Tagging)", "peter", "prod-b", FlinkMetricInput{LagTrend: 50, MaxLag: 50000, AvgLag: 2000, IsBP: true, CpuMax: 99, CpuAvg: 45, HeapMax: 65}),
		ComputeFlinkJobAnalysis("job_click_heat", "点击热力图 (Click_Heatmap)", "zhang.san", "prod-a", FlinkMetricInput{CpuMax: 15, CpuAvg: 10, HeapMax: 20}),
		ComputeFlinkJobAnalysis("job_state_heavy", "大促状态机 (Promo_State)", "li.si", "prod-b", FlinkMetricInput{LagTrend: 1200, MaxLag: 15000, AvgLag: 14000, IsBP: true, CpuMax: 75, CpuAvg: 40, HeapMax: 95, FullGc: 2, Restarts: 1}),
	}, nil
}

func (s *MockBchService) GetFlinkJobConfig(id string) (string, error) {
	configMap := map[string]interface{}{
		"_info":          "Configuration fetched via YARN REST API",
		"job_id":         id,
		"application_id": fmt.Sprintf("application_1704067200000_%s", id[len(id)-4:]),
		"environment": map[string]string{
			"flink.version": "1.16.2",
			"java.version":  "1.8.0_312",
		},
		"jobmanager.config": map[string]string{
			"jobmanager.memory.process.size": "2048m",
			"jobmanager.rpc.address":         "ip-10-0-12-85",
		},
		"taskmanager.config": map[string]string{
			"parallelism.default":             "16",
			"taskmanager.numberOfTaskSlots":   "4",
			"taskmanager.memory.process.size": "4096m",
			"taskmanager.memory.managed.fraction": "0.4",
		},
		"state.config": map[string]string{
			"state.backend":                  "rocksdb",
			"state.checkpoints.dir":          "hdfs://namenode:8020/flink/checkpoints",
			"execution.checkpointing.interval": "180000",
			"execution.checkpointing.min-pause": "60000",
		},
		"jvm.args": "-XX:+UseParallelGC -XX:MaxGCPauseMillis=200 -XX:+HeapDumpOnOutOfMemoryError",
	}

	bytes, err := json.MarshalIndent(configMap, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (s *MockBchService) DiagnoseFlinkJob(id string) (*FlinkJob, error) {
	jobs, err := s.ListFlinkJobs()
	if err != nil {
		return nil, err
	}
	for _, j := range jobs {
		if j.ID == id {
			return &j, nil
		}
	}
	return nil, fmt.Errorf("job not found")
}

func (s *MockBchService) ListSparkJobs() ([]SparkJob, error) {
	return []SparkJob{
		{
			ID:          "spark_bill_calc",
			Name:        "账单日终对账 (Daily_Billing_Reconcile)",
			Owner:       "wang.wu",
			Cluster:     "prod-a",
			Status:      "SUCCEEDED",
			Labels:      []string{"数据倾斜", "资源过度配置"},
			DurationSec: 3600,
			Metrics: SparkJobMetric{
				ExecutorMemoryOverheadMB: 2048,
				MaxTaskDurationSec:       320,
				AvgTaskDurationSec:       25,
				TotalTasks:               1000,
				FailedTasks:              0,
				CpuSkewRatio:             5.4,
				MemorySkewRatio:          1.2,
				InputBytes:               536870912000, // 500 GB
				ShuffleReadBytes:         107374182400, // 100 GB
				ShuffleWriteBytes:        107374182400,
			},
			TuningAdvice: "1. 开启自适应查询执行 (Adaptive Query Execution, AQE): `spark.sql.adaptive.enabled=true`\n2. 增加 Shuffle 分区数，防止单 Task 倾斜严重: `spark.sql.shuffle.partitions=200` (当前配置 50)\n3. 优化 Key 散列，增加盐值打散热点数据。",
		},
		{
			ID:          "spark_user_retention",
			Name:        "用户留存统计 (User_Retention_Cohort)",
			Owner:       "zhao.liu",
			Cluster:     "prod-b",
			Status:      "FAILED",
			Labels:      []string{"内存溢出 (OOM)"},
			DurationSec: 1200,
			Metrics: SparkJobMetric{
				ExecutorMemoryOverheadMB: 512,
				MaxTaskDurationSec:       180,
				AvgTaskDurationSec:       30,
				TotalTasks:               500,
				FailedTasks:              12,
				CpuSkewRatio:             1.5,
				MemorySkewRatio:          4.8,
				InputBytes:               214748364800, // 200 GB
				ShuffleReadBytes:         53687091200,  // 50 GB
				ShuffleWriteBytes:        53687091200,
			},
			TuningAdvice: "1. 提高 Executor 堆外内存比例以支持高内存 Shuffle: `spark.executor.memoryOverhead=2048` (当前为 512m)\n2. 发现 executor 发生 GC 雪崩或 OOM，建议增加 Executor 内存: `spark.executor.memory=8g` (当前 4g)\n3. 调整 Join 策略，避免大表 Broadcast 导致 Driver/Executor 内存耗尽。",
		},
		{
			ID:          "spark_click_join",
			Name:        "点击流宽表关联 (Click_Stream_Wide_Join)",
			Owner:       "sun.qi",
			Cluster:     "prod-a",
			Status:      "RUNNING",
			Labels:      []string{"长尾 Task"},
			DurationSec: 1800,
			Metrics: SparkJobMetric{
				ExecutorMemoryOverheadMB: 1024,
				MaxTaskDurationSec:       850,
				AvgTaskDurationSec:       45,
				TotalTasks:               2000,
				FailedTasks:              2,
				CpuSkewRatio:             8.2,
				MemorySkewRatio:          2.1,
				InputBytes:               1073741824000, // 1 TB
				ShuffleReadBytes:         322122547200,  // 300 GB
				ShuffleWriteBytes:        322122547200,
			},
			TuningAdvice: "1. 开启慢 Task 推测执行，规避慢节点长尾效应: `spark.speculation=true`\n2. 排查运行超时的节点 `ip-10-0-12-32` 是否存在 CPU 节流或磁盘 IO 坏道。\n3. 调整并行度，使 Task 分配更均匀: `spark.default.parallelism=120`。",
		},
	}, nil
}

func (s *MockBchService) TuneSparkJob(id string) (*SparkJob, error) {
	jobs, err := s.ListSparkJobs()
	if err != nil {
		return nil, err
	}
	for _, j := range jobs {
		if j.ID == id {
			return &j, nil
		}
	}
	return nil, fmt.Errorf("job not found")
}

func (s *MockBchService) GetHdfsFsImage(namespace string) (*HdfsFsImageStats, error) {
	// Static high-fidelity data sourced from hadoop_fsimage_dashboard_depth.html
	switch namespace {
	case "NS2", "ns2":
		return &HdfsFsImageStats{
			Namespace:       "NS2",
			TotalRecords:    "123008820",
			TotalFiles:      "119.4 M",
			TotalDirs:       "3.6 M",
			TotalSize:       "10.33 PB",
			AvgFileSize:     "92.94 MB",
			MaxDepth:        "20",
			ProcessingTime:  "9115.10",
			ProcessingSpeed: "13495",
			DepthData: []HdfsFsImageDepthStats{
				{"0 级", 16, 0.0},
				{"3 级", 167513, 4.6},
				{"4 级", 156901, 4.3},
				{"7 级", 544200, 15.0},
				{"8 级", 1793026, 49.4},
				{"9 级", 357339, 9.8},
			},
			SizeData: []HdfsFsImageSizeStats{
				{"0B", 22033, 0.0},
				{"<1KB", 14787125, 12.4},
				{"1KB-10KB", 28498359, 23.9},
				{"10KB-100KB", 16606227, 13.9},
				{"100KB-1MB", 8073560, 6.8},
				{"1MB-10MB", 8083433, 6.8},
				{"10MB-100MB", 19770859, 16.6},
				{"100MB-1GB", 21855615, 18.3},
				{"1GB-10GB", 1680684, 1.4},
				{">10GB", 161, 0.0},
			},
			UserData: []HdfsFsImageUserStats{
				{"nrdc_admin", 49038331, 39.9, "5.89 PB"},
				{"dpi_admin", 43747265, 35.6, "544.47 TB"},
				{"production", 20281256, 16.5, "1.00 PB"},
				{"production_bonc", 4123584, 3.4, "1.60 PB"},
				{"flume_collector", 2953113, 2.4, "1003.06 TB"},
			},
			ModifyData: []HdfsFsImageTimeStats{
				{"<1周", 41111046, 33.4},
				{"1周-1月", 14633381, 11.9},
				{"1-3月", 28669401, 23.3},
				{"3月-1年", 35580300, 28.9},
				{"1-2年", 135586, 0.1},
				{"2年以上", 2859946, 2.4},
			},
			AccessData: []HdfsFsImageTimeStats{
				{"<1周", 40512399, 33.8},
				{"1周-1月", 14336641, 12.0},
				{"1-3月", 27583880, 23.0},
				{"3月-1年", 34549170, 28.9},
				{"1-2年", 62732, 0.1},
				{"2年以上", 2696220, 2.2},
			},
			FileTypeData: []HdfsFsImageFileTypeStats{
				{".orc", 11180396, 9.4},
				{".sgn", 9828947, 8.2},
				{".gz", 9319482, 7.8},
				{".txt", 1827453, 1.5},
				{".c000", 228465, 0.2},
			},
			PathData: []HdfsFsImagePathPattern{
				{"/user/bdoc", 106667681, 86.7},
				{"/originaldata/oss", 14369504, 11.7},
				{"/benchmarks/nnbench1", 700720, 0.6},
				{"/originaldata/uosp", 513127, 0.4},
				{"/user/production", 241930, 0.2},
			},
			ZeroByteFiles: 22033,
			TrashFiles:    1771,
		}, nil
	default:
		// Default to NS1
		return &HdfsFsImageStats{
			Namespace:       "NS1",
			TotalRecords:    "93030336",
			TotalFiles:      "77.1 M",
			TotalDirs:       "15.9 M",
			TotalSize:       "7.08 PB",
			AvgFileSize:     "98.53 MB",
			MaxDepth:        "18",
			ProcessingTime:  "8136.93",
			ProcessingSpeed: "11433",
			DepthData: []HdfsFsImageDepthStats{
				{"0 级", 25, 0.0},
				{"3 级", 5070698, 31.8},
				{"4 级", 5233618, 32.9},
				{"5 级", 734711, 4.6},
				{"6 级", 618682, 3.9},
				{"7 级", 1869959, 11.7},
				{"8 级", 1390568, 8.7},
			},
			SizeData: []HdfsFsImageSizeStats{
				{"0B", 1053, 0.0},
				{"<1KB", 14054646, 18.2},
				{"1KB-10KB", 3408852, 4.4},
				{"10KB-100KB", 2815077, 3.7},
				{"100KB-1MB", 3543341, 4.6},
				{"1MB-10MB", 12157110, 15.8},
				{"10MB-100MB", 23127579, 30.0},
				{"100MB-1GB", 17301907, 22.4},
				{"1GB-10GB", 681462, 0.9},
				{">10GB", 17228, 0.0},
			},
			UserData: []HdfsFsImageUserStats{
				{"production", 49778480, 53.5, "5.56 PB"},
				{"zj_zstp", 15288982, 16.4, "478.22 TB"},
				{"nrdc", 13048364, 14.0, "236.61 TB"},
				{"custom", 5434882, 5.8, "524.69 TB"},
				{"production_bonc", 5416138, 5.8, "78.40 TB"},
			},
			ModifyData: []HdfsFsImageTimeStats{
				{"<1周", 31974807, 34.4},
				{"1周-1月", 14459684, 15.5},
				{"1-3月", 15159871, 16.3},
				{"3月-1年", 20086425, 21.6},
				{"1-2年", 7242600, 7.8},
				{"2年以上", 3889373, 4.2},
			},
			AccessData: []HdfsFsImageTimeStats{
				{"<1周", 30224464, 39.1},
				{"1周-1月", 13267290, 17.2},
				{"1-3月", 13828198, 17.9},
				{"3月-1年", 13452138, 17.4},
				{"1-2年", 3777981, 4.9},
				{"2年以上", 2602364, 3.5},
			},
			FileTypeData: []HdfsFsImageFileTypeStats{
				{".orc", 29733113, 38.6},
				{".gz", 8635516, 11.2},
				{".sgn", 8600541, 11.2},
				{".txt", 1797788, 2.3},
				{".c000", 3057553, 4.0},
			},
			PathData: []HdfsFsImagePathPattern{
				{"/user/bdoc", 40240488, 43.3},
				{"/user/bdoc587961664", 12996533, 14.0},
				{"/tmp/hive", 10408868, 11.2},
				{"/originaldata/oss", 8617951, 9.3},
				{"/user/bdoc663000", 8397233, 9.0},
			},
			ZeroByteFiles: 1053,
			TrashFiles:    495519,
		}, nil
	}
}

func (s *MockBchService) ListEmployees() ([]BchEmployee, error) {
	return []BchEmployee{
		{
			ID:          "emp_bch_inspect",
			Name:        "BCH 深度巡检数字员工",
			Status:      "idle",
			StatusDesc:  "就绪",
			Description: "负责对开源 Hadoop 集群、YARN 资源队列及 HDFS 进行全天候深度巡检与健康打分。",
			Skills:      []string{"Hadoop 定时自动巡检 SOP", "HDFS 小文件风险诊断", "JVM 堆内存溢出预警"},
			Tools:       []string{"query_vm_metrics", "query_hadoop_jmx"},
			RecentTasks: []BchEmployeeTask{
				{"2小时前", "对哈池 BCH 生产集群执行例行深度巡检", "成功生成巡检报告，得分 98分"},
				{"14小时前", "对呼池 BCH 生产集群执行例行深度巡检", "巡检完成，发现 2 项警告，得分 82分"},
			},
		},
		{
			ID:          "emp_bch_diagnose",
			Name:        "BCH 作业诊断数字员工",
			Status:      "working",
			StatusDesc:  "正在诊断流作业...",
			Description: "专注于大数据 Flink / Spark 作业稳定性诊断、资源配额调整以及代码热点性能优化。",
			Skills:      []string{"Flink 消费 Lag 积压根因定位", "Spark 数据倾斜与长尾 Task 调优", "三角验证假性空闲排查"},
			Tools:       []string{"query_vm_metrics", "query_yarn_config", "query_spark_history"},
			RecentTasks: []BchEmployeeTask{
				{"10分钟前", "诊断流作业: 风控实时计算 (Risk_Model)", "定位为计算过载瓶颈，输出并行度调整参数"},
				{"1小时前", "提取 YARN 任务配置: 交易核心链路 (Trade_Analysis)", "成功导出运行配置 JSON 文档"},
				{"4小时前", "分析离线批作业 OOM 根因: 用户留存统计", "定位为 Shuffle 堆外内存过剩，输出调优建议"},
			},
		},
		{
			ID:          "emp_bch_duty",
			Name:        "BCH 值班运维数字员工",
			Status:      "idle",
			StatusDesc:  "正在监听告警流...",
			Description: "负责实时接入集群与组件告警，在 5 分钟滑动窗口内执行降噪合并、根因推导与自动升级。",
			Skills:      []string{"告警时间窗口聚类算法", "AI 故障根因关联推导", "飞书/钉钉渠道多级告警分派"},
			Tools:       []string{"query_vm_metrics", "ops_ack_alert"},
			RecentTasks: []BchEmployeeTask{
				{"30分钟前", "处理 NameNode GC 停顿引发的次生告警组", "成功降噪合并 12 条原始事件，降噪比 91%"},
				{"2小时前", "处理 DataNode 网络阻塞告警组", "已合并 6 条事件，判定为物理硬件故障，标记已处理"},
			},
		},
	}, nil
}

var (
	mockYarnQueues     []YarnQueueEvaluation
	mockYarnQueuesOnce sync.Once
	mockYarnMutex      sync.Mutex
)

func initMockYarnQueues() {
	mockYarnQueues = mockYarnQueueDefaults()
}

func mockYarnQueueDefaults() []YarnQueueEvaluation {
	return []YarnQueueEvaluation{
		{
			ID:                 "root.default",
			Name:               "默认队列",
			Cluster:            "prod-a",
			Status:             "healthy",
			RiskLevel:          "low",
			CurrentCapacity:    10.0,
			MaxCapacity:        100.0,
			UsedCapacity:       2.5,
			Metrics:            YarnQueueMetric{AvgCpuPercent: 2.5, MaxCpuPercent: 4.8, AvgMemPercent: 3.1, MaxMemPercent: 5.2, ActiveApps: 8},
			PendingContainers:  0,
			WaitingApps:        0,
			PeakUsage30d:       4.8,
			LastActiveTime:     "2026-06-09 07:30",
			Reasons:            []string{"队列资源配置与历史使用量匹配良好。"},
			Advice:             "运行状态健康，无需调整配置。",
			Action:             "none",
			TargetCapacity:     10.0,
			TargetMaxCapacity:  100.0,
			ConfigPatch:        "",
			RollbackPlan:       "",
		},
		{
			ID:                 "root.dev.flink",
			Name:               "Flink 实时开发队列",
			Cluster:            "prod-a",
			Status:             "healthy",
			RiskLevel:          "low",
			CurrentCapacity:    25.0,
			MaxCapacity:        60.0,
			UsedCapacity:       12.0,
			Metrics:            YarnQueueMetric{AvgCpuPercent: 12.0, MaxCpuPercent: 45.0, AvgMemPercent: 15.0, MaxMemPercent: 52.0, ActiveApps: 2},
			PendingContainers:  5,
			WaitingApps:        1,
			PeakUsage30d:       45.0,
			LastActiveTime:     "2026-06-09 07:55",
			Reasons:            []string{"开发队列在高峰期使用率符合预期，水位合理。"},
			Advice:             "运行状态健康，建议维持当前配额以支撑后续流式任务迭代。",
			Action:             "none",
			TargetCapacity:     25.0,
			TargetMaxCapacity:  60.0,
			ConfigPatch:        "",
			RollbackPlan:       "",
		},
		{
			ID:                 "root.prod.spark",
			Name:               "Spark 生产队列",
			Cluster:            "prod-b",
			Status:             "under_allocated",
			RiskLevel:          "medium",
			CurrentCapacity:    35.0,
			MaxCapacity:        80.0,
			UsedCapacity:       78.5,
			Metrics:            YarnQueueMetric{AvgCpuPercent: 78.5, MaxCpuPercent: 95.0, AvgMemPercent: 72.0, MaxMemPercent: 88.0, ActiveApps: 14},
			PendingContainers:  120,
			WaitingApps:        3,
			PeakUsage30d:       80.0,
			LastActiveTime:     "2026-06-09 07:58",
			Reasons:            []string{"队列资源使用率长期处于极高水位", "队列内作业存在排队情况，当前有 120 个 Pending Container 和 3 个等待中的 Application"},
			Advice:             "由于近期有大量日终对账和宽表关联批作业运行，该队列资源严重受限，建议从 root.offline.batch 回收资源并为该队列扩容至 50% 份额。",
			Action:             "expand",
			TargetCapacity:     50.0,
			TargetMaxCapacity:  90.0,
			ConfigPatch:        "<!-- fair-scheduler.xml allocations patch -->\n<queue name=\"prod\">\n  <queue name=\"spark\">\n    <weight>5.0</weight>\n    <maxResources>512000 mb, 200 vcores</maxResources>\n  </queue>\n</queue>",
			RollbackPlan:       "<!-- fair-scheduler.xml rollback patch -->\n<queue name=\"prod\">\n  <queue name=\"spark\">\n    <weight>3.5</weight>\n    <maxResources>358400 mb, 140 vcores</maxResources>\n  </queue>\n</queue>",
		},
		{
			ID:                 "root.offline.batch",
			Name:               "离线批处理队列",
			Cluster:            "prod-a",
			Status:             "over_allocated",
			RiskLevel:          "low",
			CurrentCapacity:    20.0,
			MaxCapacity:        80.0,
			UsedCapacity:       2.1,
			Metrics:            YarnQueueMetric{AvgCpuPercent: 2.1, MaxCpuPercent: 5.5, AvgMemPercent: 3.5, MaxMemPercent: 8.0, ActiveApps: 0},
			PendingContainers:  0,
			WaitingApps:        0,
			PeakUsage30d:       5.5,
			LastActiveTime:     "2026-06-09 05:00",
			Reasons:            []string{"近 30 天最高资源峰值仅为 5.5%，均值低于 3%", "当前队列中无作业运行或排队，存在较大闲置空间"},
			Advice:             "检测到队列容量过剩，建议缩容至 5% 份额，释放的 15% 份额可回收到主资源池以提供给其他高负载队列。",
			Action:             "downsize",
			TargetCapacity:     5.0,
			TargetMaxCapacity:  50.0,
			ConfigPatch:        "<!-- capacity-scheduler.xml capacity downsize patch -->\n<property>\n  <name>yarn.scheduler.capacity.root.offline.batch.capacity</name>\n  <value>5.0</value>\n</property>\n<property>\n  <name>yarn.scheduler.capacity.root.offline.batch.maximum-capacity</name>\n  <value>50.0</value>\n</property>",
			RollbackPlan:       "<!-- capacity-scheduler.xml rollback patch -->\n<property>\n  <name>yarn.scheduler.capacity.root.offline.batch.capacity</name>\n  <value>20.0</value>\n</property>\n<property>\n  <name>yarn.scheduler.capacity.root.offline.batch.maximum-capacity</name>\n  <value>80.0</value>\n</property>",
		},
		{
			ID:                 "root.test",
			Name:               "临时测试队列",
			Cluster:            "prod-a",
			Status:             "idle",
			RiskLevel:          "low",
			CurrentCapacity:    8.0,
			MaxCapacity:        20.0,
			UsedCapacity:       0.1,
			Metrics:            YarnQueueMetric{AvgCpuPercent: 0.1, MaxCpuPercent: 0.1, AvgMemPercent: 0.2, MaxMemPercent: 0.2, ActiveApps: 0},
			PendingContainers:  0,
			WaitingApps:        0,
			PeakUsage30d:       0.1,
			LastActiveTime:     "2026-05-12 14:20",
			Reasons:            []string{"队列已超过 20 天无活跃作业提交", "近 30 天峰值资源利用率低于 0.5%", "测试任务流量已大部分迁移至临时沙箱集群"},
			Advice:             "队列属于长期未使用状态，建议回收其 90% 的资源（配额下调至 1%），保留最低测试限额以防后续零星测试报错。",
			Action:             "reclaim",
			TargetCapacity:     1.0,
			TargetMaxCapacity:  5.0,
			ConfigPatch:        "<!-- capacity-scheduler.xml capacity reclaim patch -->\n<property>\n  <name>yarn.scheduler.capacity.root.test.capacity</name>\n  <value>1.0</value>\n</property>\n<property>\n  <name>yarn.scheduler.capacity.root.test.maximum-capacity</name>\n  <value>5.0</value>\n</property>",
			RollbackPlan:       "<!-- capacity-scheduler.xml rollback patch -->\n<property>\n  <name>yarn.scheduler.capacity.root.test.capacity</name>\n  <value>8.0</value>\n</property>\n<property>\n  <name>yarn.scheduler.capacity.root.test.maximum-capacity</name>\n  <value>20.0</value>\n</property>",
		},
		{
			ID:                 "root.temp_sandbox",
			Name:               "沙箱实验队列",
			Cluster:            "prod-b",
			Status:             "idle",
			RiskLevel:          "low",
			CurrentCapacity:    2.0,
			MaxCapacity:        10.0,
			UsedCapacity:       0.0,
			Metrics:            YarnQueueMetric{AvgCpuPercent: 0.0, MaxCpuPercent: 0.0, AvgMemPercent: 0.0, MaxMemPercent: 0.0, ActiveApps: 0},
			PendingContainers:  0,
			WaitingApps:        0,
			PeakUsage30d:       0.0,
			LastActiveTime:     "未见活跃",
			Reasons:            []string{"队列历史使用率为 0%", "最后活跃时间无记录，该队列创建后长期处于空跑状态"},
			Advice:             "建议完全回收该队列的所有资源，释放配额并清除配置。",
			Action:             "reclaim",
			TargetCapacity:     0.0,
			TargetMaxCapacity:  0.0,
			ConfigPatch:        "<!-- fair-scheduler.xml allocations delete patch -->\n<!-- 删除 root.temp_sandbox 节点 -->\n<queue name=\"temp_sandbox\" operation=\"delete\" />",
			RollbackPlan:       "<!-- fair-scheduler.xml rollback patch -->\n<queue name=\"temp_sandbox\">\n  <weight>0.2</weight>\n  <maxResources>20480 mb, 8 vcores</maxResources>\n</queue>",
		},
	}
}

func (s *MockBchService) ListYarnQueues() ([]YarnQueueEvaluation, error) {
	mockYarnQueuesOnce.Do(initMockYarnQueues)
	mockYarnMutex.Lock()
	defer mockYarnMutex.Unlock()
	res := make([]YarnQueueEvaluation, len(mockYarnQueues))
	copy(res, mockYarnQueues)
	return res, nil
}

func (s *MockBchService) ExecuteYarnQueueAction(id string) (bool, error) {
	mockYarnQueuesOnce.Do(initMockYarnQueues)
	mockYarnMutex.Lock()
	defer mockYarnMutex.Unlock()
	for i, q := range mockYarnQueues {
		if q.ID == id {
			mockYarnQueues[i].Status = "healthy"
			mockYarnQueues[i].CurrentCapacity = q.TargetCapacity
			mockYarnQueues[i].MaxCapacity = q.TargetMaxCapacity
			mockYarnQueues[i].Action = "none"
			actionDesc := "缩容/回收"
			if q.Action == "expand" {
				actionDesc = "扩容"
				mockYarnQueues[i].PendingContainers = 0
				mockYarnQueues[i].WaitingApps = 0
			}
			mockYarnQueues[i].Reasons = []string{"闭环调优已成功执行", fmt.Sprintf("队列已%s至配置目标: %v%%", actionDesc, q.TargetCapacity)}
			mockYarnQueues[i].Advice = "配置已动态重载，运行状态健康。"
			mockYarnQueues[i].ConfigPatch = ""
			return true, nil
		}
	}
	return false, fmt.Errorf("queue %s not found", id)
}

func (s *MockBchService) RollbackYarnQueueAction(id string) (bool, error) {
	mockYarnQueuesOnce.Do(initMockYarnQueues)
	mockYarnMutex.Lock()
	defer mockYarnMutex.Unlock()
	for _, baseline := range mockYarnQueueDefaults() {
		if baseline.ID != id {
			continue
		}
		for i := range mockYarnQueues {
			if mockYarnQueues[i].ID == id {
				mockYarnQueues[i] = baseline
				return true, nil
			}
		}
		mockYarnQueues = append(mockYarnQueues, baseline)
		return true, nil
	}
	return false, fmt.Errorf("queue %s not found", id)
}
