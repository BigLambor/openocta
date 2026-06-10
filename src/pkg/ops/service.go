package ops

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/openocta/openocta/pkg/db"
)

var (
	serviceMu   sync.RWMutex
	storePath   string
	clusters    []Cluster
	clusterRepo *clusterRepository
)

const envSeedDemoData = "OPENOCTA_SEED_DEMO_DATA"

func seedDemoDataEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envSeedDemoData))) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

// InitStore loads cluster assets into openocta.db and keeps an in-memory read cache.
func InitStore(stateDir string) error {
	serviceMu.Lock()
	defer serviceMu.Unlock()

	storePath = filepath.Join(stateDir, "ops", "clusters.json")
	clusterRepo = newClusterRepository(db.GetDB())
	if clusterRepo == nil {
		clusters = []Cluster{}
		return fmt.Errorf("openocta.db 未初始化")
	}

	if _, err := clusterRepo.ImportJSON(storePath); err != nil {
		return err
	}
	loaded, err := clusterRepo.List("")
	if err != nil {
		return err
	}
	if len(loaded) == 0 && seedDemoDataEnabled() {
		for _, c := range demoClusters() {
			if err := clusterRepo.Upsert(c); err != nil {
				return err
			}
		}
		loaded, err = clusterRepo.List("")
		if err != nil {
			return err
		}
	}
	clusters = loaded
	return nil
}

// ListClusters returns clusters, optionally filtered by domain.
func ListClusters(domain string) ([]Cluster, error) {
	serviceMu.RLock()
	defer serviceMu.RUnlock()

	domain = strings.TrimSpace(strings.ToLower(domain))
	if clusterRepo == nil {
		return []Cluster{}, nil
	}
	return clusterRepo.List(domain)
}

// GetCluster returns one cluster by ID.
func GetCluster(id string) (Cluster, error) {
	serviceMu.RLock()
	defer serviceMu.RUnlock()

	if clusterRepo == nil {
		return Cluster{}, fmt.Errorf("ops store 未初始化")
	}
	return clusterRepo.Get(id)
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
	if err := ValidateMonitorLabelsForCluster(domain, status, in.MonitorLabels); err != nil {
		return Cluster{}, err
	}
	normalizedLabels, err := NormalizeMonitorLabels(strings.TrimSpace(in.MonitorLabels))
	if err != nil {
		return Cluster{}, err
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
		MonitorLabels:  normalizedLabels,
		VMUrlRef:       strings.TrimSpace(in.VMUrlRef),
		MetricsBaseUrl: normalizeMetricsBaseURL(in.MetricsBaseUrl),
		JMXUrl:         strings.TrimSpace(in.JMXUrl),
		FIManagerUrl:   strings.TrimSpace(in.FIManagerUrl),
		GBaseDsnRef:    strings.TrimSpace(in.GBaseDsnRef),
		CredentialsRef: strings.TrimSpace(in.CredentialsRef),
	}

	serviceMu.Lock()
	defer serviceMu.Unlock()

	if clusterRepo == nil {
		return Cluster{}, fmt.Errorf("ops store 未初始化")
	}
	if existing, ok, err := clusterRepo.FindByDomainName(domain, name); err != nil {
		return Cluster{}, err
	} else if ok && existing.ID != c.ID {
		return Cluster{}, fmt.Errorf("该业务域下已存在同名集群: %s", name)
	}
	if err := clusterRepo.Create(c); err != nil {
		return Cluster{}, err
	}
	if err := reloadClustersLocked(); err != nil {
		return Cluster{}, err
	}
	return c, nil
}

// PatchCluster updates fields on an existing cluster.
func PatchCluster(id string, patch ClusterPatch) (Cluster, error) {
	serviceMu.Lock()
	defer serviceMu.Unlock()

	if clusterRepo == nil {
		return Cluster{}, fmt.Errorf("ops store 未初始化")
	}

	c, err := clusterRepo.Get(id)
	if err != nil {
		return Cluster{}, err
	}
	if patch.Name != nil {
		name := strings.TrimSpace(*patch.Name)
		if name == "" {
			return Cluster{}, fmt.Errorf("集群名称不能为空")
		}
		if existing, ok, err := clusterRepo.FindByDomainName(c.Domain, name); err != nil {
			return Cluster{}, err
		} else if ok && existing.ID != c.ID {
			return Cluster{}, fmt.Errorf("该业务域下已存在同名集群: %s", name)
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
		c.MetricsBaseUrl = normalizeMetricsBaseURL(*patch.MetricsBaseUrl)
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
	if err := ValidateMonitorLabelsForCluster(c.Domain, c.Status, c.MonitorLabels); err != nil {
		return Cluster{}, err
	}
	if c.MonitorLabels != "" {
		normalized, err := NormalizeMonitorLabels(c.MonitorLabels)
		if err != nil {
			return Cluster{}, err
		}
		c.MonitorLabels = normalized
	}
	c.UpdatedAtMs = nowMs()
	if err := clusterRepo.Update(c); err != nil {
		return Cluster{}, err
	}
	if err := reloadClustersLocked(); err != nil {
		return Cluster{}, err
	}
	return c, nil
}

// DeleteCluster removes a cluster by ID.
func DeleteCluster(id string) error {
	serviceMu.Lock()
	defer serviceMu.Unlock()

	id = strings.TrimSpace(id)
	if clusterRepo == nil {
		return fmt.Errorf("ops store 未初始化")
	}
	if err := clusterRepo.Delete(id, nowMs()); err != nil {
		return err
	}
	return reloadClustersLocked()
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

	if snapshots, err := refreshClusterHealthFacts(clusters); err == nil {
		applySnapshotHealthToSummary(&summary, snapshots)
	}
	summary.PendingAlerts = CountPendingAlerts()

	vmCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	enrichDashboardVMHealth(vmCtx, &summary)
	return summary
}

func applySnapshotHealthToSummary(summary *DashboardSummary, snapshots []HealthSnapshot) {
	if len(snapshots) == 0 {
		return
	}
	byDomain := map[string][]HealthSnapshot{}
	for _, s := range snapshots {
		byDomain[s.Domain] = append(byDomain[s.Domain], s)
	}
	for i := range summary.Domains {
		d := &summary.Domains[i]
		items := byDomain[d.Domain]
		if len(items) == 0 {
			continue
		}

		var totalScore, scoreCount int
		var totalCoverage float64
		var degradedCount, partialCount int
		missing := map[string]struct{}{}
		present := map[string]struct{}{}

		for _, s := range items {
			totalCoverage += s.Coverage
			if s.Score != nil {
				totalScore += *s.Score
				scoreCount++
			}
			switch s.ScoreStatus {
			case ScoreStatusDegraded:
				degradedCount++
			case ScoreStatusPartial:
				partialCount++
			}
			for _, src := range s.MissingSources {
				missing[src] = struct{}{}
			}
			for _, src := range s.PresentSources {
				present[src] = struct{}{}
			}
		}

		coverage := totalCoverage / float64(len(items))
		d.Coverage = &coverage
		d.HealthScoreSource = "composite"
		d.MissingSources = sortedKeys(missing)
		d.PresentSources = sortedKeys(present)
		if scoreCount > 0 {
			score := totalScore / scoreCount
			d.HealthScore = &score
			d.ScoreStatus = scoreStatusFromScore(score)
			d.HealthScoreNote = fmt.Sprintf("L3 Facts 综合分，覆盖度 %.0f%%", coverage*100)
			continue
		}
		if degradedCount > 0 {
			d.ScoreStatus = ScoreStatusDegraded
			d.HealthScoreNote = fmt.Sprintf("%d 个对象必需源缺失或失败，覆盖度 %.0f%%", degradedCount, coverage*100)
		} else if partialCount > 0 {
			d.ScoreStatus = ScoreStatusPartial
			d.HealthScoreNote = fmt.Sprintf("%d 个对象覆盖不足，覆盖度 %.0f%%", partialCount, coverage*100)
		} else {
			d.ScoreStatus = ScoreStatusUnknown
			d.HealthScoreNote = "L3 Facts 暂无可用综合分"
		}
	}
}

func persistLocked() error {
	if clusterRepo == nil {
		return fmt.Errorf("ops store 未初始化")
	}
	return reloadClustersLocked()
}

func reloadClustersLocked() error {
	loaded, err := clusterRepo.List("")
	if err != nil {
		return err
	}
	clusters = loaded
	return nil
}

func demoClusters() []Cluster {
	now := nowMs()
	return []Cluster{
		{
			ID:             "cluster-bch-prod-a",
			Name:           "哈池 BCH 生产集群 A",
			Domain:         DomainHadoop,
			Region:         "哈池",
			NodeCount:      3457,
			Components:     []string{"HDFS", "YARN", "HIVE", "SPARK"},
			Owner:          "彭晓东",
			Status:         "healthy",
			CreatedAtMs:    now,
			UpdatedAtMs:    now,
			MonitorLabels:  "job=hadoop-prod",
			MetricsBaseUrl: "http://127.0.0.1:18900",
		},
		{
			ID:             "cluster-gbase-prod",
			Name:           "哈池 GBase 生产数据库",
			Domain:         DomainGBase,
			Region:         "哈池",
			NodeCount:      12,
			Components:     []string{"GBase 8a", "GBase 8t", "ConnectionPool"},
			Owner:          "赵铁柱",
			Status:         "healthy",
			CreatedAtMs:    now,
			UpdatedAtMs:    now,
			MonitorLabels:  "job=gbase-prod",
			MetricsBaseUrl: "http://127.0.0.1:18900",
		},
		{
			ID:             "cluster-fi-prod",
			Name:           "呼池 FusionInsight 生产集群",
			Domain:         DomainFI,
			Region:         "呼和浩特",
			NodeCount:      85,
			Components:     []string{"FI-YARN", "FI-HBase", "FI-Kafka"},
			Owner:          "王小明",
			Status:         "warning",
			CreatedAtMs:    now,
			UpdatedAtMs:    now,
			MonitorLabels:  "job=fi-prod",
			MetricsBaseUrl: "http://127.0.0.1:18900",
		},
		{
			ID:             "cluster-gov-platform",
			Name:           "数据开发治理中心平台",
			Domain:         DomainGovernance,
			Region:         "北京",
			NodeCount:      5,
			Components:     []string{"MetadataRegistry", "LineageEngine", "DataQuality"},
			Owner:          "李华",
			Status:         "healthy",
			CreatedAtMs:    now,
			UpdatedAtMs:    now,
			MonitorLabels:  "job=gov-platform",
			MetricsBaseUrl: "http://127.0.0.1:18900",
		},
		{
			ID:             "cluster-dataapp-scheduler",
			Name:           "核心数据 App 调度平台",
			Domain:         DomainDataApps,
			Region:         "北京",
			NodeCount:      8,
			Components:     []string{"Airflow", "DolphinScheduler", "SLA-Monitor"},
			Owner:          "陈刚",
			Status:         "healthy",
			CreatedAtMs:    now,
			UpdatedAtMs:    now,
			MonitorLabels:  "job=dataapp-scheduler",
			MetricsBaseUrl: "http://127.0.0.1:18900",
		},
	}
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
