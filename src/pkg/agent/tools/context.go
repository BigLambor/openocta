package tools

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/openocta/openocta/pkg/session"
)

// ClusterConfig represents the minimal configuration needed by tools.
type ClusterConfig struct {
	ID             string
	Name           string
	Region         string
	MonitorLabels  string
	VMUrlRef       string
	MetricsBaseUrl string
	JMXUrl         string
	FIManagerUrl   string
	GBaseDsnRef    string
	CredentialsRef string
}

// GetClusterConfig is a hook registered by the gateway to provide cluster configurations.
var GetClusterConfig func(clusterID string) (ClusterConfig, error)

// ListClustersConfig is a hook registered by the gateway to list clusters.
var ListClustersConfig func(domain string) ([]ClusterConfig, error)

// OpsContext represents the parsed ops context from the prompt/transcript.
type OpsContext struct {
	Domain    string
	ClusterID string
	Component string
}

// ParseOpsContext extracts domain, cluster ID, and component from the context or session transcript.
func ParseOpsContext(ctx context.Context) *OpsContext {
	var prompt string
	if p, ok := ctx.Value(session.ContextKeyPrompt).(string); ok && p != "" {
		prompt = p
	} else if path, ok := ctx.Value(session.ContextKeyTranscriptPath).(string); ok && path != "" {
		if msgs, err := session.ReadTranscriptMessages(path, 0); err == nil {
			for _, m := range msgs {
				if strings.ToLower(m.Role) == "user" {
					for _, c := range m.Content {
						if strings.EqualFold(c.Type, "text") && strings.Contains(c.Text, "[运维上下文]") {
							prompt = c.Text
							break
						}
					}
				}
				if prompt != "" {
					break
				}
			}
		}
	}

	if prompt == "" {
		return nil
	}

	lines := strings.Split(prompt, "\n")
	var opsLine string
	for _, l := range lines {
		if strings.Contains(l, "[运维上下文]") {
			opsLine = l
			break
		}
	}
	if opsLine == "" {
		return nil
	}

	parts := strings.Split(opsLine, "|")
	var domain, clusterID, component string
	var title, subtitle string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "[运维上下文]") {
			part = strings.TrimPrefix(part, "[运维上下文]")
			part = strings.TrimSpace(part)
		}
		if strings.HasPrefix(part, "业务域:") {
			displayName := strings.TrimSpace(strings.TrimPrefix(part, "业务域:"))
			domain = displayNameToDomain(displayName)
		} else if strings.HasPrefix(part, "目标:") {
			title = strings.TrimSpace(strings.TrimPrefix(part, "目标:"))
		} else if strings.HasPrefix(part, "cluster=") {
			clusterID = strings.TrimSpace(strings.TrimPrefix(part, "cluster="))
		} else if strings.HasPrefix(part, "clusters=") {
			clusterID = "all"
		} else if strings.HasPrefix(part, "component=") {
			component = strings.TrimSpace(strings.TrimPrefix(part, "component="))
			if decoded, err := url.QueryUnescape(component); err == nil {
				component = decoded
			}
		} else {
			subtitle = part
		}
	}

	if component == "" && clusterID != "" && clusterID != "all" {
		if title != "" {
			if GetClusterConfig != nil {
				c, err := GetClusterConfig(clusterID)
				if err == nil {
					if title != c.Name {
						component = title
					}
				} else {
					if title != subtitle {
						component = title
					}
				}
			} else {
				if title != subtitle {
					component = title
				}
			}
		}
	}

	return &OpsContext{
		Domain:    domain,
		ClusterID: clusterID,
		Component: component,
	}
}

func displayNameToDomain(name string) string {
	switch name {
	case "BCH生态":
		return "hadoop"
	case "FI商业生态":
		return "fi"
	case "GBase数据库":
		return "gbase"
	case "开发治理平台":
		return "governance"
	case "数据App运维":
		return "dataapps"
	default:
		return strings.ToLower(name)
	}
}

func domainDisplayName(domain string) string {
	switch strings.ToLower(domain) {
	case "hadoop":
		return "BCH生态"
	case "fi":
		return "FI商业生态"
	case "gbase":
		return "GBase数据库"
	case "governance":
		return "开发治理平台"
	case "dataapps":
		return "数据App运维"
	default:
		return domain
	}
}

// BuildOpsContextLine constructs the [运维上下文] line to be prepended to the prompt.
func BuildOpsContextLine(domain, clusterId, component string) string {
	if domain == "" {
		return ""
	}
	domainName := domainDisplayName(domain)

	// If clusterId is empty or "all"
	if clusterId == "" || clusterId == "all" {
		var clusterCount int
		if ListClustersConfig != nil {
			clusters, _ := ListClustersConfig(domain)
			clusterCount = len(clusters)
		}
		title := "业务域全域"
		subtitle := fmt.Sprintf("%d 个集群", clusterCount)
		clusterPart := fmt.Sprintf("clusters=%d", clusterCount)
		return fmt.Sprintf("[运维上下文] 业务域: %s | 目标: %s | %s | %s", domainName, title, subtitle, clusterPart)
	}

	var name, region string
	var err error
	var hasCluster bool
	if GetClusterConfig != nil {
		var c ClusterConfig
		c, err = GetClusterConfig(clusterId)
		if err == nil {
			name = c.Name
			region = c.Region
			hasCluster = true
		}
	}

	if component != "" {
		title := component
		subtitle := name
		if subtitle == "" {
			subtitle = "自定义上下文"
		}
		clusterPart := "cluster=" + clusterId
		componentPart := "component=" + url.QueryEscape(component)
		return fmt.Sprintf("[运维上下文] 业务域: %s | 目标: %s | %s | %s | %s", domainName, title, subtitle, clusterPart, componentPart)
	}

	if !hasCluster {
		title := clusterId
		subtitle := "自定义上下文"
		clusterPart := "cluster=" + clusterId
		return fmt.Sprintf("[运维上下文] 业务域: %s | 目标: %s | %s | %s", domainName, title, subtitle, clusterPart)
	}

	title := name
	var subtitle string
	if region != "" {
		subtitle = region + " · 集群全域"
	} else {
		subtitle = "集群全域视角"
	}
	clusterPart := "cluster=" + clusterId
	return fmt.Sprintf("[运维上下文] 业务域: %s | 目标: %s | %s | %s", domainName, title, subtitle, clusterPart)
}

// InjectLabelsIntoPromQL parses the labels string and injects them into all metric selectors in the PromQL query.
func InjectLabelsIntoPromQL(query string, labelsStr string) string {
	labelsStr = strings.TrimSpace(labelsStr)
	labelsStr = strings.TrimPrefix(labelsStr, "{")
	labelsStr = strings.TrimSuffix(labelsStr, "}")
	if labelsStr == "" {
		return query
	}

	var labelPairs []string
	for _, p := range strings.Split(labelsStr, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			labelPairs = append(labelPairs, p)
		}
	}
	if len(labelPairs) == 0 {
		return query
	}

	var result strings.Builder
	runes := []rune(query)
	n := len(runes)

	isAlpha := func(r rune) bool {
		return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || r == ':'
	}
	isAlnum := func(r rune) bool {
		return isAlpha(r) || (r >= '0' && r <= '9')
	}

	keywords := map[string]bool{
		"and": true, "or": true, "unless": true, "by": true, "without": true,
		"on": true, "ignoring": true, "group_left": true, "group_right": true,
		"bool": true, "offset": true, "sum": true, "avg": true, "min": true,
		"max": true, "stddev": true, "stdvar": true, "count": true,
		"count_values": true, "bottomk": true, "topk": true, "quantile": true,
		"rate": true, "irate": true, "increase": true, "sum_over_time": true,
		"avg_over_time": true, "min_over_time": true, "max_over_time": true,
		"count_over_time": true, "abs": true, "absent": true, "absent_over_time": true,
		"ceil": true, "changes": true, "clamp": true, "clamp_max": true, "clamp_min": true,
		"day_of_month": true, "day_of_week": true, "days_in_month": true, "delta": true,
		"deriv": true, "exp": true, "floor": true, "histogram_quantile": true,
		"holt_winters": true, "hour": true, "idelta": true, "ln": true, "log2": true,
		"log10": true, "minute": true, "month": true, "predict_linear": true,
		"resets": true, "round": true, "scalar": true, "sort": true, "sort_desc": true,
		"sqrt": true, "time": true, "timestamp": true, "vector": true, "year": true,
	}

	inBrackets := false
	inByClause := false
	byParenDepth := 0

	for i := 0; i < n; {
		r := runes[i]

		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			result.WriteRune(r)
			i++
			continue
		}

		if r == '[' {
			inBrackets = true
			result.WriteRune(r)
			i++
			continue
		}
		if r == ']' {
			inBrackets = false
			result.WriteRune(r)
			i++
			continue
		}
		if r == '(' {
			if inByClause {
				byParenDepth++
			}
			result.WriteRune(r)
			i++
			continue
		}
		if r == ')' {
			if inByClause {
				byParenDepth--
				if byParenDepth <= 0 {
					inByClause = false
				}
			}
			result.WriteRune(r)
			i++
			continue
		}

		if isAlpha(r) {
			start := i
			for i < n && isAlnum(runes[i]) {
				i++
			}
			ident := string(runes[start:i])

			lowerIdent := strings.ToLower(ident)
			if lowerIdent == "by" || lowerIdent == "without" {
				inByClause = true
				byParenDepth = 0
			}

			nextNonSpaceChar := rune(0)
			nextIdx := i
			for nextIdx < n {
				nr := runes[nextIdx]
				if nr != ' ' && nr != '\t' && nr != '\n' && nr != '\r' {
					nextNonSpaceChar = nr
					break
				}
				nextIdx++
			}

			isFunctionCall := nextNonSpaceChar == '('
			hasSelectorBraces := nextNonSpaceChar == '{'

			isMetric := !keywords[lowerIdent] && !isFunctionCall && !inBrackets && !inByClause

			if isMetric {
				if hasSelectorBraces {
					result.WriteString(ident)
					for i < nextIdx {
						result.WriteRune(runes[i])
						i++
					}
					result.WriteRune('{')
					i++

					braceStart := i
					braceCount := 1
					for i < n && braceCount > 0 {
						if runes[i] == '{' {
							braceCount++
						} else if runes[i] == '}' {
							braceCount--
						}
						if braceCount > 0 {
							i++
						}
					}
					braceContent := strings.TrimSpace(string(runes[braceStart:i]))
					if braceContent != "" {
						result.WriteString(braceContent)
						result.WriteString(",")
					}
					result.WriteString(strings.Join(labelPairs, ","))
					result.WriteRune('}')
					i++
				} else {
					result.WriteString(ident)
					result.WriteRune('{')
					result.WriteString(strings.Join(labelPairs, ","))
					result.WriteRune('}')
				}
			} else {
				result.WriteString(ident)
			}
			continue
		}

		result.WriteRune(r)
		i++
	}

	return result.String()
}
