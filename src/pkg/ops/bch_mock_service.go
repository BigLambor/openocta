package ops

import (
	"encoding/json"
	"fmt"
	"math"
)

// MockBchService implements BchService.
type MockBchService struct{}

func NewMockBchService() BchService {
	return &MockBchService{}
}

func (s *MockBchService) GetClustersHealth() ([]BchClusterHealth, error) {
	return []BchClusterHealth{
		{
			ID:             "cluster-prod-a",
			Name:           "哈池 BCH 生产集群 A (prod-a)",
			Region:         "哈池",
			Status:         "healthy",
			Score:          98,
			NodeCount:      120,
			ActiveAlerts:   0,
			CpuUsedPercent: 62.5,
			MemUsedPercent: 78.2,
			DfsUsedPercent: 54.1,
			Metrics: map[string]interface{}{
				"activeNodes":   120,
				"decommission":  0,
				"totalBlocks":   91699026,
				"activeContainers": 1840,
			},
		},
		{
			ID:             "cluster-prod-b",
			Name:           "呼池 BCH 生产集群 A (prod-b)",
			Region:         "呼池",
			Status:         "warning",
			Score:          82,
			NodeCount:      80,
			ActiveAlerts:   2,
			CpuUsedPercent: 88.0,
			MemUsedPercent: 91.5,
			DfsUsedPercent: 81.3,
			Metrics: map[string]interface{}{
				"activeNodes":   78,
				"decommission":  2,
				"totalBlocks":   131486447,
				"activeContainers": 2560,
			},
		},
	}, nil
}

// Internal helper to calculate scores for Flink Jobs (matching JS engine in flink_doctor.html)
func computeFlinkJobAnalysis(id, name, owner, cluster string, lagTrend int, maxLag, avgLag int64, isBP bool, cpuMax, cpuAvg, heapMax, fullGc, restarts int) FlinkJob {
	score := 100
	stabilityPenalty := 0
	perfPenalty := 0
	effPenalty := 0
	var penalties []FlinkPenalty

	rootCause := "S0"
	rootCauseText := "运行健康"
	diagnosis := "各项指标正常，无积压。"
	actions := []string{"无需干预，持续观察。"}

	step1 := FlinkCotStep{Text: "Lag 趋势平稳，无数据倾斜。", State: "active"}
	step2 := FlinkCotStep{Text: "无反压现象，数据流通顺畅。", State: "active"}
	step3 := FlinkCotStep{Text: "计算资源与内存利用率处于健康区间。", State: "active"}

	// 1. Stability
	if restarts > 3 {
		stabilityPenalty += 100
		penalties = append(penalties, FlinkPenalty{Item: "频繁重启 (>3次/h)", Deduction: 100, Type: "fatal"})
		rootCause = "Fatal"
		rootCauseText = "频繁重启"
		diagnosis = "作业陷入死亡循环。"
		actions = []string{"检查 JobManager 日志", "排查代码逻辑错误"}
		step3 = FlinkCotStep{Text: "检测到 1h 内重启 > 3次，触发一票否决。", State: "critical"}
	} else if fullGc > 0 {
		p := fullGc * 20
		stabilityPenalty += p
		penalties = append(penalties, FlinkPenalty{Item: fmt.Sprintf("GC 雪崩 (%d次)", fullGc), Deduction: p, Type: "fatal"})
		rootCause = "S2"
		rootCauseText = "内存不足/GC停顿"
		diagnosis = "发生 Full GC，面临 OOM 风险，业务线程停顿。"
		actions = []string{"扩容 TaskManager Heap 内存", "检查 State TTL 设置"}
		step3 = FlinkCotStep{Text: fmt.Sprintf("检测到 %d 次 Full GC，内存触发红线。", fullGc), State: "critical"}
	}

	// 2. Performance
	hasLag := lagTrend > 0
	hasSkew := avgLag > 0 && maxLag > (5*avgLag)

	if hasLag {
		perfPenalty += 20
		penalties = append(penalties, FlinkPenalty{Item: "积压恶化 (LagTrend>0)", Deduction: 20, Type: "fatal"})
		step1 = FlinkCotStep{Text: fmt.Sprintf("Lag 趋势向上 (斜率 %d)，出现严重积压。", lagTrend), State: "critical"}
	}
	if hasSkew {
		perfPenalty += 15
		penalties = append(penalties, FlinkPenalty{Item: "数据倾斜 (Max>5*Avg)", Deduction: 15, Type: "warning"})
		rootCause = "S4"
		rootCauseText = "数据倾斜"
		diagnosis = "Max Lag 远超 Avg Lag，存在严重单点倾斜。"
		actions = []string{"开启 LocalKeyBy 预聚合", "增加随机盐 (Salting) 打散数据"}
		step1 = FlinkCotStep{Text: fmt.Sprintf("Max Lag (%d) 远大于 Avg Lag (%d)，存在单点倾斜。", maxLag, avgLag), State: "warning"}
	}
	if isBP {
		perfPenalty += 10
		penalties = append(penalties, FlinkPenalty{Item: "持续反压", Deduction: 10, Type: "warning"})
		step2 = FlinkCotStep{Text: "检测到持续反压，瓶颈在 Flink 内部或下游。", State: "warning"}
	} else if hasLag && !isBP {
		step2 = FlinkCotStep{Text: "有积压但无反压，判定为 Source 端读取瓶颈。", State: "warning"}
	}

	// 3. Efficiency
	if rootCause == "S0" || rootCause == "S4" {
		if hasLag && isBP && cpuMax > 90 {
			rootCause = "S1"
			rootCauseText = "计算资源瓶颈"
			diagnosis = "CPU被打满，算力不足导致积压反压。"
			actions = []string{"增加并行度 (Parallelism)", "Profile 热点代码优化"}
			effPenalty += 5
			penalties = append(penalties, FlinkPenalty{Item: "计算过载 (CPU>90%)", Deduction: 5, Type: "info"})
			step3 = FlinkCotStep{Text: "CPU Max > 90%，结合反压判定为计算瓶颈。", State: "critical"}
		} else if hasLag && isBP && cpuMax < 40 {
			rootCause = "S3"
			rootCauseText = "外部 IO 阻塞"
			diagnosis = "CPU 闲置但存在反压，说明线程在等待下游 IO 响应。"
			actions = []string{"检查 Sink 端数据库负载", "开启批量写入 (Batch Flush)"}
			step3 = FlinkCotStep{Text: fmt.Sprintf("CPU Max仅 %d%% 但存在反压，判定为假性空闲 (IO 阻塞)，严禁缩容。", cpuMax), State: "warning"}
		} else if !hasLag && !isBP && cpuMax < 30 {
			rootCause = "S7"
			rootCauseText = "资源过度配置"
			diagnosis = "无积压无反压且峰值 CPU 极低，资源严重浪费。"
			actions = []string{"降低作业并行度", "减少 TaskManager 数量以节约成本"}
			effPenalty += 10
			penalties = append(penalties, FlinkPenalty{Item: "资源浪费 (闲置)", Deduction: 10, Type: "info"})
			step3 = FlinkCotStep{Text: "无积压且 CPU < 30%，判定为资源过度配置。", State: "warning"}
		}
	}

	score = 100 - stabilityPenalty - perfPenalty - effPenalty
	if score < 0 {
		score = 0
	}

	sScore := 100 - int(math.Min(100.0, float64(stabilityPenalty)*2.5))
	pScore := 100 - int(math.Min(100.0, float64(perfPenalty)*2.8))
	eScore := 100 - int(math.Min(100.0, float64(effPenalty)*4.0))

	return FlinkJob{
		ID:            id,
		Name:          name,
		Owner:         owner,
		Cluster:       cluster,
		Status:        "RUNNING",
		Score:         score,
		SScore:        sScore,
		PScore:        pScore,
		EScore:        eScore,
		Metrics: FlinkJobMetric{
			LagTrend:        lagTrend,
			MaxLag:          maxLag,
			AvgLag:          avgLag,
			IsBackpressured: isBP,
			CpuMax:          cpuMax,
			CpuAvg:          cpuAvg,
			HeapMax:         heapMax,
			FullGcCount:     fullGc,
			Restarts:        restarts,
		},
		Penalties:     penalties,
		Diagnosis:     diagnosis,
		RootCause:     rootCause,
		RootCauseText: rootCauseText,
		Actions:       actions,
		CotSteps: FlinkCotSteps{
			Step1: step1,
			Step2: step2,
			Step3: step3,
		},
	}
}

func (s *MockBchService) ListFlinkJobs() ([]FlinkJob, error) {
	return []FlinkJob{
		computeFlinkJobAnalysis("job_tx_core", "交易核心链路 (Trade_Analysis)", "cui.chao", "prod-a", 0, 10, 8, false, 60, 55, 60, 0, 0),
		computeFlinkJobAnalysis("job_log_sink", "日志归档 (Log_ES_Sink)", "lu.yang", "prod-a", 500, 8000, 7800, true, 25, 15, 40, 0, 0),
		computeFlinkJobAnalysis("job_risk_calc", "风控实时计算 (Risk_Model)", "tom", "prod-b", 800, 12000, 11500, true, 98, 95, 70, 0, 0),
		computeFlinkJobAnalysis("job_user_tag", "用户画像流 (User_Tagging)", "peter", "prod-b", 50, 50000, 2000, true, 99, 45, 65, 0, 0),
		computeFlinkJobAnalysis("job_click_heat", "点击热力图 (Click_Heatmap)", "zhang.san", "prod-a", 0, 0, 0, false, 15, 10, 20, 0, 0),
		computeFlinkJobAnalysis("job_state_heavy", "大促状态机 (Promo_State)", "li.si", "prod-b", 1200, 15000, 14000, true, 75, 40, 95, 2, 1),
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
