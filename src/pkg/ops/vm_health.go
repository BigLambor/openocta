package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/openocta/openocta/pkg/agent/tools"
)

const vmQueryTimeout = 4 * time.Second

// domainVMQueries are representative instant PromQL probes per business domain.
var domainVMQueries = map[string]string{
	DomainHadoop:     `avg(up{job=~".*(hadoop|yarn|hdfs).*"} or up{instance=~".*hadoop.*"})`,
	DomainFI:         `avg(up{job=~".*(fusion|fi).*"} or up{instance=~".*fi.*"})`,
	DomainGBase:      `avg(up{job=~".*gbase.*"} or up{instance=~".*gbase.*"})`,
	DomainGovernance: `avg(up{job=~".*(governance|metadata).*"} or up{instance=~".*governance.*"})`,
	DomainDataApps:   `avg(up{job=~".*(dataapp|scheduler|pipeline).*"} or up{instance=~".*dataapp.*"})`,
}

type promInstantResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
	Error string `json:"error"`
}

// vmClient performs instant PromQL queries against VictoriaMetrics / Prometheus.
type vmClient struct {
	baseURL string
	http    *http.Client
}

func resolveVMBaseURL() string {
	if u := strings.TrimSpace(os.Getenv("VICTORIAMETRICS_URL")); u != "" {
		return normalizeMetricsBaseURL(u)
	}
	if u := strings.TrimSpace(os.Getenv("PROMETHEUS_URL")); u != "" {
		return normalizeMetricsBaseURL(u)
	}
	return ""
}

func normalizeMetricsBaseURL(u string) string {
	u = strings.TrimSpace(u)
	u = strings.TrimSuffix(u, "/")
	for {
		switch {
		case strings.HasSuffix(u, "/api/v1/query"):
			u = strings.TrimSuffix(u, "/api/v1/query")
		case strings.HasSuffix(u, "/api/v1"):
			u = strings.TrimSuffix(u, "/api/v1")
		default:
			return strings.TrimSuffix(u, "/")
		}
		u = strings.TrimSuffix(u, "/")
	}
}

func resolveClusterMetricsBaseURL(c Cluster) string {
	if c.MetricsBaseUrl != "" {
		return normalizeMetricsBaseURL(c.MetricsBaseUrl)
	}
	ref := strings.TrimSpace(c.VMUrlRef)
	if ref == "" {
		return ""
	}
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return normalizeMetricsBaseURL(ref)
	}
	return normalizeMetricsBaseURL(os.Getenv(ref))
}

func newVMClient() *vmClient {
	base := resolveVMBaseURL()
	if base == "" {
		return nil
	}
	return &vmClient{
		baseURL: base,
		http:    &http.Client{Timeout: vmQueryTimeout},
	}
}

func (c *vmClient) configured() bool {
	return c != nil && c.baseURL != ""
}

func (c *vmClient) queryInstant(ctx context.Context, query string) ([]float64, error) {
	apiURL, err := url.Parse(c.baseURL + "/api/v1/query")
	if err != nil {
		return nil, err
	}
	q := apiURL.Query()
	q.Set("query", query)
	apiURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("VM API %d: %s", resp.StatusCode, truncate(string(body), 120))
	}

	var parsed promInstantResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if parsed.Status != "success" {
		if parsed.Error != "" {
			return nil, fmt.Errorf("%s", parsed.Error)
		}
		return nil, fmt.Errorf("query failed: %s", parsed.Status)
	}

	values := make([]float64, 0, len(parsed.Data.Result))
	for _, r := range parsed.Data.Result {
		if len(r.Value) < 2 {
			continue
		}
		v, ok := scalarToFloat(r.Value[1])
		if !ok {
			continue
		}
		values = append(values, v)
	}
	return values, nil
}

// queryInstantByLabel returns one scalar per time series keyed by a label (e.g. job_id).
func (c *vmClient) queryInstantByLabel(ctx context.Context, query, labelKey string) (map[string]float64, error) {
	if !c.configured() {
		return nil, fmt.Errorf("vm client not configured")
	}
	apiURL, err := url.Parse(c.baseURL + "/api/v1/query")
	if err != nil {
		return nil, err
	}
	q := apiURL.Query()
	q.Set("query", query)
	apiURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("VM API %d: %s", resp.StatusCode, truncate(string(body), 120))
	}

	var parsed promInstantResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if parsed.Status != "success" {
		if parsed.Error != "" {
			return nil, fmt.Errorf("%s", parsed.Error)
		}
		return nil, fmt.Errorf("query failed: %s", parsed.Status)
	}

	out := map[string]float64{}
	for _, r := range parsed.Data.Result {
		if len(r.Value) < 2 {
			continue
		}
		key := strings.TrimSpace(r.Metric[labelKey])
		if key == "" {
			key = strings.TrimSpace(r.Metric["job_name"])
		}
		if key == "" {
			continue
		}
		v, ok := scalarToFloat(r.Value[1])
		if !ok {
			continue
		}
		out[key] = v
	}
	return out, nil
}

func scalarToFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case string:
		var f float64
		if _, err := fmt.Sscanf(n, "%f", &f); err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

// scoreFromSamples maps PromQL scalar(s) to a 0–100 health score.
func scoreFromSamples(samples []float64) (int, bool) {
	if len(samples) == 0 {
		return 0, false
	}
	var sum float64
	for _, v := range samples {
		sum += normalizeSample(v)
	}
	score := int(sum/float64(len(samples)) + 0.5)
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score, true
}

func normalizeSample(v float64) float64 {
	switch {
	case v <= 0:
		return 0
	case v <= 1:
		return v * 100
	case v <= 100:
		return v
	default:
		// Large counters / gauges without known scale — treat presence as partial health.
		return 75
	}
}

func domainHealthScore(ctx context.Context, client *vmClient, domain string) (*int, string) {
	if client == nil || !client.configured() {
		return nil, "未配置 VICTORIAMETRICS_URL"
	}

	clustersList, err := ListClusters(domain)
	if err != nil || len(clustersList) == 0 {
		return nil, "该业务域下无集群"
	}

	var totalScore int
	var count int
	var errors []string

	for _, c := range clustersList {
		baseQuery := domainVMQueries[domain]
		if baseQuery == "" {
			baseQuery = "avg(up)"
		}

		clusterClient := client
		if u := resolveClusterMetricsBaseURL(c); u != "" {
			clusterClient = &vmClient{
				baseURL: u,
				http:    client.http,
			}
		}

		query := baseQuery
		if c.MonitorLabels != "" {
			query = tools.InjectLabelsIntoPromQL(baseQuery, c.MonitorLabels)
		}

		values, err := clusterClient.queryInstant(ctx, query)
		if err != nil || len(values) == 0 {
			fallbackQuery := "avg(up)"
			if c.MonitorLabels != "" {
				fallbackQuery = tools.InjectLabelsIntoPromQL("avg(up)", c.MonitorLabels)
			}
			values, err = clusterClient.queryInstant(ctx, fallbackQuery)
		}

		if err != nil {
			errors = append(errors, fmt.Sprintf("集群 %s 查询失败: %v", c.Name, err))
			continue
		}

		score, ok := scoreFromSamples(values)
		if ok {
			totalScore += score
			count++
		}
	}

	if count == 0 {
		if len(errors) > 0 {
			return nil, strings.Join(errors, "; ")
		}
		return nil, "监控指标暂无数据"
	}

	avgScore := totalScore / count
	return &avgScore, ""
}

func enrichDashboardVMHealth(ctx context.Context, summary *DashboardSummary) {
	client := newVMClient()
	summary.VMConfigured = client != nil && client.configured()
	if !summary.VMConfigured {
		for i := range summary.Domains {
			if summary.Domains[i].ClusterCount > 0 {
				summary.Domains[i].HealthScoreNote = "未配置 VICTORIAMETRICS_URL"
			}
		}
		return
	}

	for i := range summary.Domains {
		d := &summary.Domains[i]
		if d.ClusterCount == 0 {
			continue
		}
		if d.HealthScoreSource == "composite" {
			continue
		}
		score, note := domainHealthScore(ctx, client, d.Domain)
		if score != nil {
			d.HealthScore = score
			d.HealthScoreSource = "victoriametrics"
			if note != "" {
				d.HealthScoreNote = note
			}
		} else {
			d.HealthScoreNote = note
		}
	}
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
