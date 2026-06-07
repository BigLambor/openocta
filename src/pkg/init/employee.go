package init

import (
	"os"

	"github.com/openocta/openocta/pkg/config"
	"github.com/openocta/openocta/pkg/employees"
)

// InitEmployee 在项目启动时确保 ~/.openocta/employees 目录存在并装载默认员工。
func InitEmployee(_ *config.OpenOctaConfig) error {
	env := func(k string) string { return os.Getenv(k) }
	root := employees.ResolveEmployeesDir(env)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}

	defaultEmployees := []*employees.Manifest{
		{
			ID:          "emp_bch_duty",
			Name:        "BCH 值班运维数字员工",
			Description: "负责实时接入集群与组件告警，在 5 分钟滑动窗口内执行降噪合并、根因推导与自动升级。",
			Prompt:      "你是一个 BCH 值班运维数字员工。请调用 query_vm_metrics 或 ops_ack_alert 等工具，实时接入并分析集群与组件告警，推导根因，进行影响分析并给出处置建议。",
			Enabled:     true,
			Type:        "智能值班",
			From:        "local",
			DomainKeys:  []string{"hadoop"},
			CapabilityKeys: []string{"observability"},
			RoleType:    "oncall",
			Responsibilities: []string{"告警实时降噪与合并", "根因推导与故障定位", "故障通知与自动升级"},
			InputSources: []string{"alerts"},
			OutputTypes: []string{"diagnosis_report"},
			ActionScopes: []string{"read_only", "ops_ack_alert"},
			SkillIDs:    []string{"alert-triage", "root-cause-analysis", "runbook-recommendation"},
		},
		{
			ID:          "emp_bch_inspect",
			Name:        "BCH 深度巡检数字员工",
			Description: "负责对开源 Hadoop 集群、YARN 资源队列及 HDFS 进行全天候深度巡检与健康打分。",
			Prompt:      "你是一个 BCH 深度巡检数字员工。请定期巡检 Hadoop (HDFS, YARN) 的核心健康状态与性能指标，识别集群风险项，进行深度评估与打分，并生成结构化的中文巡检报告。",
			Enabled:     true,
			Type:        "自动巡检",
			From:        "local",
			DomainKeys:  []string{"hadoop"},
			CapabilityKeys: []string{"inspection"},
			RoleType:    "inspector",
			Responsibilities: []string{"Hadoop 核心指标定时巡检", "HDFS 存储与元数据深度扫描", "集群健康打分与隐患预警"},
			InputSources: []string{"metrics", "logs"},
			OutputTypes: []string{"inspection_report"},
			ActionScopes: []string{"read_only"},
			SkillIDs:    []string{"alert-triage", "root-cause-analysis"},
		},
		{
			ID:          "emp_bch_diagnose",
			Name:        "BCH 作业诊断数字员工",
			Description: "专注于大数据 Flink / Spark 作业稳定性诊断、资源配额调整以及代码热点性能优化。",
			Prompt:      "你是一个 BCH 作业诊断数字员工。请使用专业的调优规则与诊断模型，分析 Flink / Spark 异常作业或长尾任务，定位资源瓶颈与代码缺陷，并给出具体的资源配额调整或代码优化方案。",
			Enabled:     true,
			Type:        "性能调优",
			From:        "local",
			DomainKeys:  []string{"hadoop"},
			CapabilityKeys: []string{"diagnosis"},
			RoleType:    "diagnoser",
			Responsibilities: []string{"Flink 算子反压与 Lag 积压诊断", "Spark 数据倾斜与长尾 Task 调优", "三角验证假性空闲排查"},
			InputSources: []string{"metrics", "logs"},
			OutputTypes: []string{"tuning_report"},
			ActionScopes: []string{"read_only"},
			SkillIDs:    []string{"alert-triage", "root-cause-analysis"},
		},
		{
			ID:          "emp_gbase_diagnose",
			Name:        "GBase 慢 SQL 诊断助手",
			Description: "专职 GBase 数据库慢查询、死锁与连接池耗尽故障的诊断与调优。",
			Prompt:      "你是一个 GBase 数据库诊断数字员工。请使用专业的 SQL 优化与配置调整能力，分析 GBase 的性能异常、死锁 and 慢日志，定位性能瓶颈，给出优化建议。",
			Enabled:     true,
			Type:        "智能值班",
			From:        "local",
			DomainKeys:  []string{"gbase"},
			CapabilityKeys: []string{"diagnosis"},
			RoleType:    "diagnoser",
			Responsibilities: []string{"GBase 慢 SQL 诊断与改写", "死锁与锁冲突分析", "连接池水位评估"},
			InputSources: []string{"metrics", "logs"},
			OutputTypes: []string{"diagnosis_report"},
			ActionScopes: []string{"read_only"},
			SkillIDs:    []string{"alert-triage", "root-cause-analysis", "runbook-recommendation"},
		},
		{
			ID:          "emp_fi_inspect",
			Name:        "FusionInsight 巡检助手",
			Description: "负责 FI 商业集群（含 YARN 队列、HBase 水位及 Kafka 分区）的定时指标巡检与风险预测。",
			Prompt:      "你是一个 FusionInsight 巡检数字员工。请定期巡检 FusionInsight 各核心组件指标，识别 HBase Region 倾斜、YARN 资源耗尽等隐患，进行健康评估打分。",
			Enabled:     true,
			Type:        "自动巡检",
			From:        "local",
			DomainKeys:  []string{"fi"},
			CapabilityKeys: []string{"inspection"},
			RoleType:    "inspector",
			Responsibilities: []string{"FI 集群健康状态评估", "YARN 队列水位与 HBase 倾斜扫描", "容量隐患预警"},
			InputSources: []string{"metrics", "logs"},
			OutputTypes: []string{"inspection_report"},
			ActionScopes: []string{"read_only"},
			SkillIDs:    []string{"alert-triage", "root-cause-analysis", "runbook-recommendation"},
		},
		{
			ID:          "emp_governance_remediate",
			Name:        "数据血缘治理助手",
			Description: "专注于开发治理平台的数据资产完整性、链路血缘断裂及表注释缺失的自动检测与修正。",
			Prompt:      "你是一个数据血缘治理数字员工。请自动扫描开发治理平台的数据资产，检测血缘链路断裂、注释缺失及冗余表，并给出修复或补全建议。",
			Enabled:     true,
			Type:        "治理中心",
			From:        "local",
			DomainKeys:  []string{"governance"},
			CapabilityKeys: []string{"governance"},
			RoleType:    "governor",
			Responsibilities: []string{"数据血缘完整性校验", "元数据与表注释自动生成", "废弃与低频表下线治理"},
			InputSources: []string{"metrics", "logs"},
			OutputTypes: []string{"remediation_report"},
			ActionScopes: []string{"read_only"},
			SkillIDs:    []string{"governance-lineage", "root-cause-analysis"},
		},
		{
			ID:          "emp_dataapps_ops",
			Name:        "数据 App SLA 护航助手",
			Description: "监控数据 App（任务管道与调度任务）的运行状态，对超时、SLA 违规或任务失败进行首问响应与处理。",
			Prompt:      "你是一个数据 App 运维数字员工。请监控数据 App 任务流运行状态，对关键 SLA 延迟、超时、数据依赖异常进行拦截诊断，并给出恢复方案。",
			Enabled:     true,
			Type:        "智能值班",
			From:        "local",
			DomainKeys:  []string{"dataapps"},
			CapabilityKeys: []string{"observability"},
			RoleType:    "oncall",
			Responsibilities: []string{"SLA 超时与运行阻塞监控", "任务链延迟与断裂分析", "自动重试与拦截处置"},
			InputSources: []string{"alerts"},
			OutputTypes: []string{"diagnosis_report"},
			ActionScopes: []string{"read_only"},
			SkillIDs:    []string{"alert-triage", "root-cause-analysis", "sla-escort"},
		},
	}

	for _, emp := range defaultEmployees {
		existing, err := employees.LoadManifest(emp.ID, env)
		if err != nil || existing == nil || !equalStringSlices(existing.SkillIDs, emp.SkillIDs) {
			_ = employees.SaveManifest(emp, env)
		}
	}

	return nil
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
