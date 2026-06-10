package ops

import "math"

// FlinkMetricInput holds raw metrics for rule-based Flink scoring.
type FlinkMetricInput struct {
	LagTrend  int
	MaxLag    int64
	AvgLag    int64
	IsBP      bool
	CpuMax    int
	CpuAvg    int
	HeapMax   int
	FullGc    int
	Restarts  int
}

// ComputeFlinkJobAnalysis scores one Flink job (flink_doctor.html parity).
func ComputeFlinkJobAnalysis(id, name, owner, cluster string, in FlinkMetricInput) FlinkJob {
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

	if in.Restarts > 3 {
		stabilityPenalty += 100
		penalties = append(penalties, FlinkPenalty{Item: "频繁重启 (>3次/h)", Deduction: 100, Type: "fatal"})
		rootCause = "Fatal"
		rootCauseText = "频繁重启"
		diagnosis = "作业陷入死亡循环。"
		actions = []string{"检查 JobManager 日志", "排查代码逻辑错误"}
		step3 = FlinkCotStep{Text: "检测到 1h 内重启 > 3次，触发一票否决。", State: "critical"}
	} else if in.FullGc > 0 {
		p := in.FullGc * 20
		stabilityPenalty += p
		penalties = append(penalties, FlinkPenalty{Item: "GC 雪崩", Deduction: p, Type: "fatal"})
		rootCause = "S2"
		rootCauseText = "内存不足/GC停顿"
		diagnosis = "发生 Full GC，面临 OOM 风险，业务线程停顿。"
		actions = []string{"扩容 TaskManager Heap 内存", "检查 State TTL 设置"}
		step3 = FlinkCotStep{Text: "检测到 Full GC，内存触发红线。", State: "critical"}
	}

	hasLag := in.LagTrend > 0
	hasSkew := in.AvgLag > 0 && in.MaxLag > (5*in.AvgLag)

	if hasLag {
		perfPenalty += 20
		penalties = append(penalties, FlinkPenalty{Item: "积压恶化 (LagTrend>0)", Deduction: 20, Type: "fatal"})
		step1 = FlinkCotStep{Text: "Lag 趋势向上，出现严重积压。", State: "critical"}
	}
	if hasSkew {
		perfPenalty += 15
		penalties = append(penalties, FlinkPenalty{Item: "数据倾斜 (Max>5*Avg)", Deduction: 15, Type: "warning"})
		rootCause = "S4"
		rootCauseText = "数据倾斜"
		diagnosis = "Max Lag 远超 Avg Lag，存在严重单点倾斜。"
		actions = []string{"开启 LocalKeyBy 预聚合", "增加随机盐 (Salting) 打散数据"}
		step1 = FlinkCotStep{Text: "Max Lag 远大于 Avg Lag，存在单点倾斜。", State: "warning"}
	}
	if in.IsBP {
		perfPenalty += 10
		penalties = append(penalties, FlinkPenalty{Item: "持续反压", Deduction: 10, Type: "warning"})
		step2 = FlinkCotStep{Text: "检测到持续反压，瓶颈在 Flink 内部或下游。", State: "warning"}
	} else if hasLag && !in.IsBP {
		step2 = FlinkCotStep{Text: "有积压但无反压，判定为 Source 端读取瓶颈。", State: "warning"}
	}

	if rootCause == "S0" || rootCause == "S4" {
		if hasLag && in.IsBP && in.CpuMax > 90 {
			rootCause = "S1"
			rootCauseText = "计算资源瓶颈"
			diagnosis = "CPU被打满，算力不足导致积压反压。"
			actions = []string{"增加并行度 (Parallelism)", "Profile 热点代码优化"}
			effPenalty += 5
			penalties = append(penalties, FlinkPenalty{Item: "计算过载 (CPU>90%)", Deduction: 5, Type: "info"})
			step3 = FlinkCotStep{Text: "CPU Max > 90%，结合反压判定为计算瓶颈。", State: "critical"}
		} else if hasLag && in.IsBP && in.CpuMax < 40 {
			rootCause = "S3"
			rootCauseText = "外部 IO 阻塞"
			diagnosis = "CPU 闲置但存在反压，说明线程在等待下游 IO 响应。"
			actions = []string{"检查 Sink 端数据库负载", "开启批量写入 (Batch Flush)"}
			step3 = FlinkCotStep{Text: "CPU 较低但存在反压，判定为 IO 阻塞。", State: "warning"}
		} else if !hasLag && !in.IsBP && in.CpuMax < 30 {
			rootCause = "S7"
			rootCauseText = "资源过度配置"
			diagnosis = "无积压无反压且峰值 CPU 极低，资源可能浪费。"
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
			LagTrend:        in.LagTrend,
			MaxLag:          in.MaxLag,
			AvgLag:          in.AvgLag,
			IsBackpressured: in.IsBP,
			CpuMax:          in.CpuMax,
			CpuAvg:          in.CpuAvg,
			HeapMax:         in.HeapMax,
			FullGcCount:     in.FullGc,
			Restarts:        in.Restarts,
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

func flinkJobToMetricInput(j FlinkJob) FlinkMetricInput {
	return FlinkMetricInput{
		LagTrend: j.Metrics.LagTrend,
		MaxLag:   j.Metrics.MaxLag,
		AvgLag:   j.Metrics.AvgLag,
		IsBP:     j.Metrics.IsBackpressured,
		CpuMax:   j.Metrics.CpuMax,
		CpuAvg:   j.Metrics.CpuAvg,
		HeapMax:  j.Metrics.HeapMax,
		FullGc:   j.Metrics.FullGcCount,
		Restarts: j.Metrics.Restarts,
	}
}

func flinkStatusFromScore(score int) string {
	switch {
	case score >= 90:
		return HealthStatusHealthy
	case score >= 70:
		return HealthStatusWarning
	default:
		return HealthStatusCritical
	}
}
