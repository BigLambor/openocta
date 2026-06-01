package ops

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	serviceMu sync.RWMutex
	storePath string
	clusters  []Cluster
)

// InitStore loads or creates the cluster store under stateDir/ops/clusters.json.
func InitStore(stateDir string) error {
	serviceMu.Lock()
	defer serviceMu.Unlock()

	storePath = filepath.Join(stateDir, "ops", "clusters.json")
	store, err := LoadStore(storePath)
	if err != nil {
		return err
	}
	clusters = store.Clusters
	return nil
}

// ListClusters returns clusters, optionally filtered by domain.
func ListClusters(domain string) ([]Cluster, error) {
	serviceMu.RLock()
	defer serviceMu.RUnlock()

	domain = strings.TrimSpace(strings.ToLower(domain))
	out := make([]Cluster, 0, len(clusters))
	for _, c := range clusters {
		if domain == "" || c.Domain == domain {
			out = append(out, c)
		}
	}
	return out, nil
}

// GetCluster returns one cluster by ID.
func GetCluster(id string) (Cluster, error) {
	serviceMu.RLock()
	defer serviceMu.RUnlock()

	id = strings.TrimSpace(id)
	for _, c := range clusters {
		if c.ID == id {
			return c, nil
		}
	}
	return Cluster{}, fmt.Errorf("集群不存在: %s", id)
}

// CreateCluster registers a new cluster.
func CreateCluster(in ClusterCreate) (Cluster, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Cluster{}, fmt.Errorf("集群名称不能为空")
	}
	domain, err := NormalizeDomain(in.Domain)
	if err != nil {
		return Cluster{}, err
	}
	status, err := NormalizeStatus(in.Status)
	if err != nil {
		return Cluster{}, err
	}
	if in.NodeCount < 0 {
		return Cluster{}, fmt.Errorf("节点数不能为负数")
	}

	components := normalizeComponents(in.Components)
	now := nowMs()
	c := Cluster{
		ID:             "cluster-" + uuid.New().String(),
		Name:           name,
		Domain:         domain,
		Region:         strings.TrimSpace(in.Region),
		NodeCount:      in.NodeCount,
		Components:     components,
		Owner:          strings.TrimSpace(in.Owner),
		Status:         status,
		Description:    strings.TrimSpace(in.Description),
		CreatedAtMs:    now,
		UpdatedAtMs:    now,
		MonitorLabels:  strings.TrimSpace(in.MonitorLabels),
		VMUrlRef:       strings.TrimSpace(in.VMUrlRef),
		MetricsBaseUrl: strings.TrimSpace(in.MetricsBaseUrl),
		JMXUrl:         strings.TrimSpace(in.JMXUrl),
		FIManagerUrl:   strings.TrimSpace(in.FIManagerUrl),
		GBaseDsnRef:    strings.TrimSpace(in.GBaseDsnRef),
		CredentialsRef: strings.TrimSpace(in.CredentialsRef),
	}

	serviceMu.Lock()
	defer serviceMu.Unlock()

	for _, existing := range clusters {
		if existing.Domain == domain && strings.EqualFold(existing.Name, name) {
			return Cluster{}, fmt.Errorf("该业务域下已存在同名集群: %s", name)
		}
	}
	clusters = append(clusters, c)
	if err := persistLocked(); err != nil {
		clusters = clusters[:len(clusters)-1]
		return Cluster{}, err
	}
	return c, nil
}

// PatchCluster updates fields on an existing cluster.
func PatchCluster(id string, patch ClusterPatch) (Cluster, error) {
	serviceMu.Lock()
	defer serviceMu.Unlock()

	idx := -1
	for i, c := range clusters {
		if c.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return Cluster{}, fmt.Errorf("集群不存在: %s", id)
	}

	c := clusters[idx]
	if patch.Name != nil {
		name := strings.TrimSpace(*patch.Name)
		if name == "" {
			return Cluster{}, fmt.Errorf("集群名称不能为空")
		}
		for _, existing := range clusters {
			if existing.ID != c.ID && existing.Domain == c.Domain && strings.EqualFold(existing.Name, name) {
				return Cluster{}, fmt.Errorf("该业务域下已存在同名集群: %s", name)
			}
		}
		c.Name = name
	}
	if patch.Domain != nil {
		domain, err := NormalizeDomain(*patch.Domain)
		if err != nil {
			return Cluster{}, err
		}
		c.Domain = domain
	}
	if patch.Region != nil {
		c.Region = strings.TrimSpace(*patch.Region)
	}
	if patch.NodeCount != nil {
		if *patch.NodeCount < 0 {
			return Cluster{}, fmt.Errorf("节点数不能为负数")
		}
		c.NodeCount = *patch.NodeCount
	}
	if patch.Components != nil {
		c.Components = normalizeComponents(*patch.Components)
	}
	if patch.Owner != nil {
		c.Owner = strings.TrimSpace(*patch.Owner)
	}
	if patch.Status != nil {
		status, err := NormalizeStatus(*patch.Status)
		if err != nil {
			return Cluster{}, err
		}
		c.Status = status
	}
	if patch.Description != nil {
		c.Description = strings.TrimSpace(*patch.Description)
	}
	if patch.MonitorLabels != nil {
		c.MonitorLabels = strings.TrimSpace(*patch.MonitorLabels)
	}
	if patch.VMUrlRef != nil {
		c.VMUrlRef = strings.TrimSpace(*patch.VMUrlRef)
	}
	if patch.MetricsBaseUrl != nil {
		c.MetricsBaseUrl = strings.TrimSpace(*patch.MetricsBaseUrl)
	}
	if patch.JMXUrl != nil {
		c.JMXUrl = strings.TrimSpace(*patch.JMXUrl)
	}
	if patch.FIManagerUrl != nil {
		c.FIManagerUrl = strings.TrimSpace(*patch.FIManagerUrl)
	}
	if patch.GBaseDsnRef != nil {
		c.GBaseDsnRef = strings.TrimSpace(*patch.GBaseDsnRef)
	}
	if patch.CredentialsRef != nil {
		c.CredentialsRef = strings.TrimSpace(*patch.CredentialsRef)
	}
	c.UpdatedAtMs = nowMs()
	clusters[idx] = c
	if err := persistLocked(); err != nil {
		return Cluster{}, err
	}
	return c, nil
}

// DeleteCluster removes a cluster by ID.
func DeleteCluster(id string) error {
	serviceMu.Lock()
	defer serviceMu.Unlock()

	id = strings.TrimSpace(id)
	for i, c := range clusters {
		if c.ID == id {
			clusters = append(clusters[:i], clusters[i+1:]...)
			return persistLocked()
		}
	}
	return fmt.Errorf("集群不存在: %s", id)
}

// BuildDashboardSummary builds overview metrics from registered clusters.
func BuildDashboardSummary() DashboardSummary {
	return buildDashboardSummary(context.Background())
}

// BuildDashboardSummaryWithContext aggregates clusters and queries VictoriaMetrics health scores.
func BuildDashboardSummaryWithContext(ctx context.Context) DashboardSummary {
	if ctx == nil {
		ctx = context.Background()
	}
	return buildDashboardSummary(ctx)
}

func buildDashboardSummary(ctx context.Context) DashboardSummary {
	serviceMu.RLock()
	defer serviceMu.RUnlock()

	summary := DashboardSummary{
		Domains: make([]DomainHealthSummary, 0, len(validDomains)),
	}
	byDomain := map[string]*DomainHealthSummary{}
	for d := range validDomains {
		byDomain[d] = &DomainHealthSummary{Domain: d}
	}

	for _, c := range clusters {
		summary.TotalClusters++
		switch c.Status {
		case "healthy":
			summary.HealthyClusters++
		case "warning":
			summary.WarningClusters++
		case "critical":
			summary.CriticalClusters++
		}
		dh := byDomain[c.Domain]
		if dh == nil {
			dh = &DomainHealthSummary{Domain: c.Domain}
			byDomain[c.Domain] = dh
		}
		dh.ClusterCount++
		switch c.Status {
		case "healthy":
			dh.HealthyCount++
		case "warning":
			dh.WarningCount++
		case "critical":
			dh.CriticalCount++
		}
	}

	order := []string{DomainHadoop, DomainFI, DomainGBase, DomainGovernance, DomainDataApps}
	for _, d := range order {
		dh := byDomain[d]
		if dh == nil {
			continue
		}
		if dh.ClusterCount == 0 {
			dh.Note = "尚未纳管集群"
		} else if dh.CriticalCount > 0 {
			dh.Note = fmt.Sprintf("%d 个集群异常", dh.CriticalCount)
		} else if dh.WarningCount > 0 {
			dh.Note = fmt.Sprintf("%d 个集群亚健康", dh.WarningCount)
		} else {
			dh.Note = "运行平稳"
		}
		summary.Domains = append(summary.Domains, *dh)
	}
	summary.PendingAlerts = CountPendingAlerts()

	vmCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	enrichDashboardVMHealth(vmCtx, &summary)
	return summary
}

func persistLocked() error {
	if storePath == "" {
		return fmt.Errorf("ops store 未初始化")
	}
	return SaveStore(storePath, &storeFile{Version: 1, Clusters: clusters})
}

func strPtr(s string) *string {
	return &s
}

func normalizeComponents(parts []string) []string {
	if len(parts) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		key := strings.ToLower(p)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	return out
}
