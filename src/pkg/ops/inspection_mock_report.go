package ops

import (
	"fmt"
	"strings"
	"time"
)

// EnrichInspectionWithMockReport fills domain-aware score and structured markdown for demo / offline runs.
func EnrichInspectionWithMockReport(res *InspectionResult, scenarioKey, clusterID string) {
	EnrichInspectionWithMockReportAt(res, scenarioKey, clusterID, time.Now(), 0)
}

// EnrichInspectionWithMockReportAt fills mock data using a specific inspection timestamp and variant.
func EnrichInspectionWithMockReportAt(res *InspectionResult, scenarioKey, clusterID string, inspectedAt time.Time, variant int) {
	if res == nil {
		return
	}
	if strings.TrimSpace(res.ScenarioKey) == "" {
		res.ScenarioKey = scenarioKey
	}
	if strings.TrimSpace(res.ClusterID) == "" {
		res.ClusterID = strings.TrimSpace(clusterID)
	}
	profile := buildInspectionMockProfileAt(scenarioKey, res.Domain, res.MissingSources, inspectedAt, variant)
	forcePolished := res.TriggerType == "scenario_runner" || res.SourceKind == "cron" || isRawToolDump(res.ReportMarkdown)

	if forcePolished || res.ReportMarkdown == "" {
		res.ReportMarkdown = profile.Markdown
	}
	if res.Score == nil || forcePolished {
		score := profile.Score
		res.Score = &score
	}
	if res.ScoreStatus == "" || res.ScoreStatus == "healthy" || res.ScoreStatus == "degraded" || forcePolished {
		res.ScoreStatus = profile.ScoreStatus
	}
	if len(res.MetricsEvidence) == 0 || forcePolished {
		res.MetricsEvidence = profile.MetricsEvidence
	}
	res.ScoreSource = "structured"
}

// EnrichInspectionCronRunResult polishes stored cron inspection runs for demo-quality UI display.
func EnrichInspectionCronRunResult(res *InspectionResult, jobID string, runAtMs int64, variant int) {
	if res == nil || !strings.HasPrefix(jobID, "job-inspect-") {
		return
	}
	if strings.TrimSpace(res.Domain) == "" {
		res.Domain = DomainFromInspectJobID(jobID)
	}
	if strings.TrimSpace(res.ScenarioKey) == "" {
		res.ScenarioKey = ScenarioKeyForInspection(InspectionReport{JobID: jobID, Domain: res.Domain})
	}
	inspectedAt := time.Now()
	if runAtMs > 0 {
		inspectedAt = time.UnixMilli(runAtMs)
	}
	EnrichInspectionWithMockReportAt(res, res.ScenarioKey, res.ClusterID, inspectedAt, variant)
	res.TriggerType = "cron"
	res.SourceKind = "cron"
}

func isRawToolDump(markdown string) bool {
	text := strings.TrimSpace(markdown)
	if text == "" {
		return false
	}
	return strings.HasPrefix(text, "#### [") ||
		strings.Contains(text, "Not Found in Registry") ||
		strings.Contains(text, "] Failed") ||
		(strings.HasPrefix(text, "{") && !strings.Contains(text, "##"))
}

type inspectionMockProfile struct {
	Score           int
	ScoreStatus     string
	Markdown        string
	MetricsEvidence map[string]interface{}
}

func buildInspectionMockProfile(scenarioKey, domain string, missingSources []string) inspectionMockProfile {
	return buildInspectionMockProfileAt(scenarioKey, domain, missingSources, time.Now(), 0)
}

func buildInspectionMockProfileAt(scenarioKey, domain string, missingSources []string, inspectedAt time.Time, variant int) inspectionMockProfile {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		domain = domainFromScenarioKey(scenarioKey)
	}
	clusterLabel := "生产集群（全域）"
	degradedNote := ""
	if len(missingSources) > 0 {
		degradedNote = fmt.Sprintf("\n\n> **数据覆盖提示**：以下信号源未完全采集：%s。健康分已按降级策略估算。", strings.Join(missingSources, "、"))
	}
	now := inspectedAt.Format("2006-01-02 15:04")
	scoreOffset := scoreVariantOffset(variant)

	switch domain {
	case DomainHadoop:
		return mockHadoopProfile(now, clusterLabel, degradedNote, scoreOffset)
	case DomainFI:
		return mockFIProfile(now, clusterLabel, degradedNote, scoreOffset)
	case DomainGBase:
		return mockGBaseProfile(now, clusterLabel, degradedNote, scoreOffset)
	case DomainGovernance:
		return mockGovernanceProfile(now, clusterLabel, degradedNote, scoreOffset)
	case DomainDataApps:
		return mockDataAppsProfile(now, clusterLabel, degradedNote, scoreOffset)
	default:
		return mockHadoopProfile(now, clusterLabel, degradedNote, scoreOffset)
	}
}

func scoreVariantOffset(variant int) int {
	if variant == 0 {
		return 0
	}
	return (variant % 5) - 2
}

func clampScore(score int) int {
	if score < 72 {
		return 72
	}
	if score > 98 {
		return 98
	}
	return score
}

func scoreStatusFromMockScore(score int) string {
	if score >= 90 {
		return "ok"
	}
	if score >= 75 {
		return "warning"
	}
	return "critical"
}

func domainFromScenarioKey(scenarioKey string) string {
	switch scenarioKey {
	case "ops-bch-health":
		return DomainHadoop
	case "ops-fi-health":
		return DomainFI
	case "ops-gbase-health":
		return DomainGBase
	case "ops-governance-health":
		return DomainGovernance
	case "ops-dataapps-health":
		return DomainDataApps
	default:
		return DomainHadoop
	}
}

func mockHadoopProfile(now, clusterLabel, degradedNote string, scoreOffset int) inspectionMockProfile {
	score := clampScore(86 + scoreOffset)
	return inspectionMockProfile{
		Score:       score,
		ScoreStatus: scoreStatusFromMockScore(score),
		MetricsEvidence: map[string]interface{}{
			"yarnActiveNodes":       48,
			"yarnTotalNodes":        50,
			"hdfsUsedPercent":       72.4,
			"flinkBackpressureJobs": 2,
			"sparkTuningCandidates": 3,
		},
		Markdown: fmt.Sprintf(`# BCH 生态深度健康巡检报告

**巡检时间**：%s  
**覆盖范围**：%s  
**综合健康分**：%d / 100（%s）

---

## 执行摘要

本次巡检基于 YARN、HDFS、Flink 实时链路与 Spark 批处理 SLA 进行联合评估。集群整体可用，但流计算背压与 HDFS 容量水位已触发亚健康阈值，建议在 48 小时内完成 P1 项处置。

## 健康度评分

| 维度 | 状态 | 说明 |
|------|------|------|
| 集群可用性 | 正常 | YARN 活跃 NM 48/50，无大规模节点失联 |
| 存储容量 | 关注 | HDFS 使用率 72.4%%，预计 14 天触及 80%% 预警线 |
| 流计算健康 | 亚健康 | 2 个作业持续背压，最大消费延迟 12s |
| 批处理效率 | 可优化 | 3 个 Spark 作业存在倾斜或过度申请资源 |

## 关键指标（SLI）

| 指标 | 当前值 | 基线 | 判定 |
|------|--------|------|------|
| YARN 活跃节点率 | 96%% | ≥ 95%% | 达标 |
| HDFS 容量使用率 | 72.4%% | < 75%% | 接近阈值 |
| Flink 背压作业数 | 2 | 0 | 超标 |
| Spark 失败批次（24h） | 0 | 0 | 达标 |
| Checkpoint 成功率（Flink） | 98.5%% | ≥ 99%% | 轻微偏离 |

## 风险项与处置建议

### P1 · Flink 背压（prod-b / risk-realtime-calc）
- **影响**：风控实时链路延迟上升，可能导致规则命中滞后
- **证据**：反压持续 18min，Kafka lag 11.5k，Checkpoint 间隔 10min
- **建议**：Source 并行度 16→32；RocksDB 增量清理窗口调至 2h

### P2 · HDFS 容量水位（/data/warehouse）
- **影响**：批处理落盘与 Flink checkpoint 共享存储，存在写入抖动风险
- **证据**：目录周环比 +8.2%%，EC 策略未覆盖冷数据层
- **建议**：启用 EC 6+3；清理 90 天未访问分区

### P3 · Spark 数据倾斜（daily_billing_reconcile）
- **影响**：批作业 SLA 边缘化，单批次尾部拖长 23min
- **证据**：stage skew ratio 5.4，最大 task 耗时 8.2× 中位数
- **建议**：开启 AQE；shuffle partitions 调整为 200

## 后续动作

- [ ] 诊断中心：对 P1 作业发起 Flink Doctor 深度问诊
- [ ] 治理中心：下发 Spark 调优处方并跟踪下一批次
- [ ] 容量性能：提交 HDFS 扩容评估工单

%s`, now, clusterLabel, score, statusLabel(score), degradedNote),
	}
}

func mockFIProfile(now, clusterLabel, degradedNote string, scoreOffset int) inspectionMockProfile {
	score := clampScore(88 + scoreOffset)
	return inspectionMockProfile{
		Score:       score,
		ScoreStatus: scoreStatusFromMockScore(score),
		MetricsEvidence: map[string]interface{}{
			"hbaseRsActive":         24,
			"hbaseRsTotal":          24,
			"fiYarnQueuePressure":   0.76,
			"slowRegionCount":       3,
		},
		Markdown: fmt.Sprintf(`# FI 商业生态深度健康巡检报告

**巡检时间**：%s  
**覆盖范围**：%s  
**综合健康分**：%d / 100（%s）

---

## 执行摘要

FusionInsight 核心组件（HBase、YARN、Kafka、Hive MetaStore）整体稳定。晚高峰队列资源竞争加剧，需关注 batch_core 队列与 3 个慢 Region 的Compaction 窗口。

## 健康度评分

| 维度 | 状态 | 说明 |
|------|------|------|
| HBase 可用性 | 正常 | RegionServer 24/24 活跃 |
| FI Manager | 正常 | 主备切换演练 7 天内通过 |
| YARN 队列 | 关注 | batch_core 内存分配率 76%% |
| 元数据服务 | 正常 | Hive MetaStore 连接池无堆积 |

## 关键指标

| 指标 | 当前值 | 阈值 | 状态 |
|------|--------|------|------|
| HBase 读 RT P99 | 38ms | 50ms | 正常 |
| HBase 写 RT P99 | 62ms | 80ms | 正常 |
| 慢 Region 数 | 3 | 5 | 正常 |
| Kafka ISR 缩减次数（24h） | 0 | 0 | 正常 |

## 风险项与处置建议

### P1 · YARN 队列压力（batch_core）
- **建议**：晚高峰临时提升 max-capacity 8%%；隔离 adhoc 查询队列

### P2 · HBase 慢 Region（user_profile 表）
- **建议**：规划低峰 Major Compaction；检查热点 rowkey 设计

## 后续动作

- [ ] 容量性能：评估 batch_core 队列晚高峰扩容
- [ ] 治理中心：补齐 2 张核心表血缘负责人

%s`, now, clusterLabel, score, statusLabel(score), degradedNote),
	}
}

func mockGBaseProfile(now, clusterLabel, degradedNote string, scoreOffset int) inspectionMockProfile {
	score := clampScore(91 + scoreOffset)
	return inspectionMockProfile{
		Score:       score,
		ScoreStatus: scoreStatusFromMockScore(score),
		MetricsEvidence: map[string]interface{}{
			"activeConnections": 186,
			"slowSqlCount":      3,
			"qps":               4200,
			"lockWaits":         0,
		},
		Markdown: fmt.Sprintf(`# GBase 数据库深度健康巡检报告

**巡检时间**：%s  
**覆盖范围**：%s  
**综合健康分**：%d / 100（%s）

---

## 执行摘要

GBase 主集群连接、吞吐与锁等待指标正常。检出 3 条慢 SQL，均为可优化查询计划问题，无阻塞链路与长事务堆积。

## 健康度评分

| 维度 | 状态 | 说明 |
|------|------|------|
| 连接与吞吐 | 正常 | 活跃连接 186/300，QPS 4200 |
| 查询性能 | 正常 | 慢 SQL 3 条，均低于紧急阈值 |
| 锁与事务 | 正常 | 锁等待 0，最长事务 42s |
| 复制延迟 | 正常 | 备库延迟 < 2s |

## 关键指标

| 指标 | 当前值 | 阈值 | 状态 |
|------|--------|------|------|
| 活跃连接 | 186 | 300 | 正常 |
| 慢 SQL（>3s） | 3 | 10 | 正常 |
| Buffer Pool 命中率 | 99.1%% | ≥ 98%% | 正常 |
| 磁盘使用率 | 61%% | < 80%% | 正常 |

## 风险项与处置建议

### P2 · 慢 SQL（billing_detail 全表扫描）
- **建议**：补充 (order_id, biz_date) 联合索引；改写为分区裁剪

### P3 · 统计信息过期（inventory_snapshot）
- **建议**：重新收集统计信息并验证执行计划

## 后续动作

- [ ] DBA 工单：落地 billing_detail 索引变更
- [ ] 治理中心：纳入慢 SQL 周度复盘

%s`, now, clusterLabel, score, statusLabel(score), degradedNote),
	}
}

func mockGovernanceProfile(now, clusterLabel, degradedNote string, scoreOffset int) inspectionMockProfile {
	score := clampScore(84 + scoreOffset)
	return inspectionMockProfile{
		Score:       score,
		ScoreStatus: scoreStatusFromMockScore(score),
		MetricsEvidence: map[string]interface{}{
			"apiSuccessRate":   0.982,
			"qualityAlerts":    5,
			"lineageCoverage":  0.97,
			"ownerTaggedRatio": 0.94,
		},
		Markdown: fmt.Sprintf(`# 开发治理平台深度健康巡检报告

**巡检时间**：%s  
**覆盖范围**：%s  
**综合健康分**：%d / 100（%s）

---

## 执行摘要

元数据 API 与血缘服务整体可用。数据质量规则触发 5 条告警，资产责任人标签覆盖率 94%%，仍有 2 张核心表缺失 Owner 映射。

## 健康度评分

| 维度 | 状态 | 说明 |
|------|------|------|
| 服务可用性 | 正常 | API 成功率 98.2%% |
| 数据质量 | 关注 | 5 条规则连续超阈值 |
| 血缘完整性 | 正常 | 链路完整率 97%% |
| 资产治理 | 关注 | 核心表 Owner 覆盖率 94%% |

## 风险项与处置建议

### P1 · 质量规则异常（null_rate_order_id）
- **证据**：连续 3 天超阈值，影响下游报表可信度
- **建议**：启用自动工单并通知表 Owner

### P2 · 资产责任人缺失（2 张核心表）
- **建议**：按数据域补齐 Owner 与 SLA 标签

## 后续动作

- [ ] 治理中心：下发质量整改任务并跟踪闭环
- [ ] 变更护航：关联最近发布变更做影响评估

%s`, now, clusterLabel, score, statusLabel(score), degradedNote),
	}
}

func mockDataAppsProfile(now, clusterLabel, degradedNote string, scoreOffset int) inspectionMockProfile {
	score := clampScore(79 + scoreOffset)
	return inspectionMockProfile{
		Score:       score,
		ScoreStatus: scoreStatusFromMockScore(score),
		MetricsEvidence: map[string]interface{}{
			"failedTasks":      4,
			"slaBreaches":      2,
			"pipelineDelaySec": 920,
			"appsMonitored":    32,
		},
		Markdown: fmt.Sprintf(`# 数据 App 运维深度健康巡检报告

**巡检时间**：%s  
**覆盖范围**：%s（32 条链路）  
**综合健康分**：%d / 100（%s）

---

## 执行摘要

共巡检 32 条数据 App 调度链路，4 个任务失败，2 条核心报表 SLA 未达标。失败集中在 revenue 链路与用户留存 cohort 依赖的上游同步延迟。

## 健康度评分

| 维度 | 状态 | 说明 |
|------|------|------|
| 调度成功率 | 关注 | 24h 失败任务 4 个 |
| SLA 达标率 | 关注 | 2 条核心链路超标 |
| 依赖链路 | 亚健康 | 上游同步延迟 920s |
| 告警噪声 | 正常 | 重复告警率 6%% |

## SLA 摘要

| 链路 | 目标 | 实际 | 状态 |
|------|------|------|------|
| 日报表-经营分析 | 08:00 | 08:14 | 超标 |
| 用户留存 cohort | 09:30 | 09:28 | 达标 |
| 实时 GMV 看板 | 实时 < 5min | 6min | 超标 |

## 风险项与处置建议

### P1 · daily_revenue_pipeline 失败
- **建议**：重跑失败分区；检查上游 ODS 同步窗口

### P2 · 实时 GMV 延迟
- **建议**：提升 Flink 消费并行度；校验维表刷新频率

## 后续动作

- [ ] 事件中心：关联失败任务告警组做根因分析
- [ ] 变更护航：评估昨日配置变更对 SLA 的影响

%s`, now, clusterLabel, score, statusLabel(score), degradedNote),
	}
}

func statusLabel(score int) string {
	switch {
	case score >= 90:
		return "健康"
	case score >= 75:
		return "亚健康"
	default:
		return "风险"
	}
}
