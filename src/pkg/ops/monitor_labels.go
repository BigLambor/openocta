package ops

import (
	"fmt"
	"regexp"
	"strings"
)

// MonitorLabelPair is one Prometheus/VictoriaMetrics label selector pair.
type MonitorLabelPair struct {
	Key   string
	Value string
}

// DomainMonitorGuide documents how a business domain links assets to VM series.
type DomainMonitorGuide struct {
	DomainLabel   string
	LabelKeys     []string
	Example       string
	BaseQueryHint string
	VerifyQuery   string
	CheckSteps    []string
}

var labelKeyPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

var domainMonitorGuides = map[string]DomainMonitorGuide{
	DomainHadoop: {
		DomainLabel:   "BCH 生态 (hadoop)",
		LabelKeys:     []string{"job", "cluster", "instance"},
		Example:       `job="hadoop-prod",cluster="bj-bch-prod"`,
		BaseQueryHint: `avg(up{job=~".*(hadoop|yarn|hdfs).*"} or up{instance=~".*hadoop.*"})`,
		VerifyQuery:   `count(up{job=~".*(hadoop|yarn|hdfs).*"})`,
		CheckSteps: []string{
			"在 VM 执行 label_values(up, job) 或 label_values(up, cluster)，确认目标集群标签值",
			"登记 monitorLabels 使用与 VM 一致的 job/cluster，而非资产 id (cluster-uuid)",
			"保存后驾驶舱域健康分应能按该集群聚合；Agent vm_query 会注入相同标签",
		},
	},
	DomainFI: {
		DomainLabel:   "FI 商业生态 (fi)",
		LabelKeys:     []string{"job", "cluster", "fusion_cluster", "instance"},
		Example:       `job="fi-prod",cluster="huhe-fi-prod"`,
		BaseQueryHint: `avg(up{job=~".*(fusion|fi).*"} or up{instance=~".*fi.*"})`,
		VerifyQuery:   `count(up{job=~".*(fusion|fi).*"})`,
		CheckSteps: []string{
			"确认 FI Manager / node_exporter 上报的 job 或 cluster 标签",
			"monitorLabels 至少包含 job 或 cluster 之一，且值与 VM 时序完全一致",
			"多 FI 集群同域时，必须用 cluster/env 等标签区分，避免域级粗查询串数据",
		},
	},
	DomainGBase: {
		DomainLabel:   "GBase 数据库 (gbase)",
		LabelKeys:     []string{"job", "cluster", "instance"},
		Example:       `job="gbase-prod",instance="gbase-primary"`,
		BaseQueryHint: `avg(up{job=~".*gbase.*"} or up{instance=~".*gbase.*"})`,
		VerifyQuery:   `count(up{job=~".*gbase.*"})`,
		CheckSteps: []string{
			"核对 GBase exporter 的 job/instance 标签",
			"主备或多套库用 cluster 或 instance 区分",
			"gbaseDsnRef 用于 SQL 巡检，monitorLabels 仅负责指标关联",
		},
	},
	DomainGovernance: {
		DomainLabel:   "开发治理平台 (governance)",
		LabelKeys:     []string{"job", "cluster", "service"},
		Example:       `job="gov-platform",service="metadata-registry"`,
		BaseQueryHint: `avg(up{job=~".*(governance|metadata).*"} or up{instance=~".*governance.*"})`,
		VerifyQuery:   `count(up{job=~".*(governance|metadata).*"})`,
		CheckSteps: []string{
			"治理平台组件常以 service/job 区分，确认 VM 中实际 label 名",
			"monitorLabels 与平台 Prometheus 抓取配置保持一致",
		},
	},
	DomainDataApps: {
		DomainLabel:   "数据 App 运维 (dataapps)",
		LabelKeys:     []string{"job", "cluster", "app"},
		Example:       `job="dataapp-scheduler",app="core-scheduler"`,
		BaseQueryHint: `avg(up{job=~".*(dataapp|scheduler|pipeline).*"} or up{instance=~".*dataapp.*"})`,
		VerifyQuery:   `count(up{job=~".*(dataapp|scheduler|pipeline).*"})`,
		CheckSteps: []string{
			"调度类 App 常用 job/app 标签，先在 VM 查 label_values(up, app)",
			"monitorLabels 用于驾驶舱健康分与 Agent PromQL 注入，与资产 id 无关",
		},
	},
}

// MonitorGuideForDomain returns label alignment guidance for a normalized domain.
func MonitorGuideForDomain(domain string) (DomainMonitorGuide, bool) {
	g, ok := domainMonitorGuides[strings.TrimSpace(strings.ToLower(domain))]
	return g, ok
}

// ParseMonitorLabels parses PromQL label selector syntax: key=value or key="value".
func ParseMonitorLabels(raw string) ([]MonitorLabelPair, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if looksLikeJSONLabels(raw) {
		return nil, fmt.Errorf("monitorLabels 请使用 PromQL 标签格式（如 job=\"prod\",cluster=\"a\"），不要使用 JSON")
	}

	raw = strings.TrimPrefix(raw, "{")
	raw = strings.TrimSuffix(raw, "}")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("monitorLabels 不能为空片段")
	}

	var pairs []MonitorLabelPair
	for _, part := range splitLabelPairs(raw) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		eq := strings.Index(part, "=")
		if eq <= 0 {
			return nil, fmt.Errorf("monitorLabels 片段无效: %q（应为 key=value）", part)
		}
		key := strings.TrimSpace(part[:eq])
		value := strings.TrimSpace(part[eq+1:])
		if !labelKeyPattern.MatchString(key) {
			return nil, fmt.Errorf("monitorLabels 标签名无效: %q", key)
		}
		unquoted, err := unquoteLabelValue(value)
		if err != nil {
			return nil, err
		}
		if unquoted == "" {
			return nil, fmt.Errorf("monitorLabels 标签 %q 的值不能为空", key)
		}
		pairs = append(pairs, MonitorLabelPair{Key: key, Value: unquoted})
	}
	if len(pairs) == 0 {
		return nil, fmt.Errorf("monitorLabels 至少包含一组 key=value")
	}
	return pairs, nil
}

// FormatMonitorLabels canonicalizes parsed pairs for storage.
func FormatMonitorLabels(pairs []MonitorLabelPair) string {
	if len(pairs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(pairs))
	for _, p := range pairs {
		parts = append(parts, fmt.Sprintf("%s=%q", p.Key, p.Value))
	}
	return strings.Join(parts, ",")
}

// NormalizeMonitorLabels parses and re-formats monitorLabels.
func NormalizeMonitorLabels(raw string) (string, error) {
	pairs, err := ParseMonitorLabels(raw)
	if err != nil {
		return "", err
	}
	return FormatMonitorLabels(pairs), nil
}

// ValidateMonitorLabelsForCluster enforces the asset → monitorLabels → VM query chain.
func ValidateMonitorLabelsForCluster(domain, status, raw string) error {
	status = strings.TrimSpace(strings.ToLower(status))
	raw = strings.TrimSpace(raw)

	if status == "inactive" {
		if raw == "" {
			return nil
		}
		_, err := NormalizeMonitorLabels(raw)
		return err
	}

	if raw == "" {
		return fmt.Errorf("非下线集群必须配置 monitorLabels，否则无法关联 VictoriaMetrics 指标（资产 id 不会自动写入 PromQL）")
	}

	pairs, err := ParseMonitorLabels(raw)
	if err != nil {
		return err
	}

	guide, ok := MonitorGuideForDomain(domain)
	if !ok {
		return nil
	}

	keySet := make(map[string]struct{}, len(pairs))
	for _, p := range pairs {
		keySet[p.Key] = struct{}{}
	}
	for _, key := range guide.LabelKeys {
		if _, ok := keySet[key]; ok {
			return nil
		}
	}
	return fmt.Errorf(
		"%s 的 monitorLabels 须包含以下标签之一: %s（示例: %s）",
		guide.DomainLabel,
		strings.Join(guide.LabelKeys, ", "),
		guide.Example,
	)
}

func looksLikeJSONLabels(raw string) bool {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "{") {
		return false
	}
	return strings.Contains(raw, `":`) || strings.Contains(raw, `": `)
}

func splitLabelPairs(raw string) []string {
	var parts []string
	var current strings.Builder
	inQuotes := false
	escaped := false

	for _, r := range raw {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' && inQuotes {
			current.WriteRune(r)
			escaped = true
			continue
		}
		if r == '"' {
			inQuotes = !inQuotes
			current.WriteRune(r)
			continue
		}
		if r == ',' && !inQuotes {
			parts = append(parts, current.String())
			current.Reset()
			continue
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func unquoteLabelValue(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		inner := value[1 : len(value)-1]
		inner = strings.ReplaceAll(inner, `\"`, `"`)
		return inner, nil
	}
	if strings.ContainsAny(value, ",{}") {
		return "", fmt.Errorf("monitorLabels 值 %q 含特殊字符时请使用双引号", value)
	}
	return value, nil
}
