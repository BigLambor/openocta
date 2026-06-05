package ops

import (
	"fmt"
	"sort"
	"time"
)

func refreshClusterHealthFacts(clustersList []Cluster) ([]HealthSnapshot, error) {
	if healthStore == nil {
		return nil, nil
	}

	existing, err := healthStore.ListSignals()
	if err != nil {
		return nil, err
	}

	collected := make([]HealthSignal, 0, len(clustersList)*2)
	for _, c := range clustersList {
		if defaultDomainHealthPolicy(c.Domain) == nil {
			continue
		}
		collected = append(collected, collectAssetStatusSignal(c))
		if sig, ok := collectAlertSignal(c); ok {
			collected = append(collected, sig)
		}
	}
	if len(collected) > 0 {
		if err := healthStore.UpsertSignals(collected); err != nil {
			return nil, err
		}
		existing, err = healthStore.ListSignals()
		if err != nil {
			return nil, err
		}
	}

	snapshots := make([]HealthSnapshot, 0, len(clustersList))
	for _, c := range clustersList {
		policy := defaultDomainHealthPolicy(c.Domain)
		if policy == nil {
			continue
		}
		signals := signalsForObject(existing, HealthObjectCluster, c.ID)
		snapshots = append(snapshots, AggregateHealthSnapshot(c, policy, signals))
	}
	if len(snapshots) > 0 {
		if err := healthStore.UpsertSnapshots(snapshots); err != nil {
			return nil, err
		}
	}
	return snapshots, nil
}

func collectAssetStatusSignal(c Cluster) HealthSignal {
	score := assetStatusScore(c.Status)
	status := HealthStatusUnknown
	if score != nil {
		status = statusFromAssetStatus(c.Status)
	}
	return newCollectorSignal(c, SignalTypeAssetStatus, status, score, map[string]interface{}{
		"clusterStatus": c.Status,
		"owner":         c.Owner,
		"nodeCount":     c.NodeCount,
		"components":    c.Components,
	}, "")
}

func assetStatusScore(status string) *int {
	var score int
	switch status {
	case "healthy":
		score = 100
	case "warning":
		score = 70
	case "critical":
		score = 35
	case "inactive", "unknown":
		return nil
	default:
		return nil
	}
	return &score
}

func statusFromAssetStatus(status string) string {
	switch status {
	case "healthy":
		return HealthStatusHealthy
	case "warning":
		return HealthStatusWarning
	case "critical":
		return HealthStatusCritical
	default:
		return HealthStatusUnknown
	}
}

func collectAlertSignal(c Cluster) (HealthSignal, bool) {
	groups := activeAlertGroupsForCluster(c.ID)
	if len(groups) == 0 {
		score := 100
		return newCollectorSignal(c, SignalTypeAlerts, HealthStatusHealthy, &score, map[string]interface{}{
			"activeCount":   0,
			"criticalCount": 0,
			"warningCount":  0,
			"infoCount":     0,
		}, ""), true
	}

	seen := map[string]AlertGroup{}
	for _, g := range groups {
		seen[alertFingerprintForGroup(g)] = g
	}

	var criticalCount, warningCount, infoCount int
	for _, g := range seen {
		switch normalizeSeverity(g.Severity) {
		case "critical":
			criticalCount++
		case "warning":
			warningCount++
		default:
			infoCount++
		}
	}

	score := 100 - criticalCount*15 - warningCount*8 - infoCount*2
	if criticalCount > 0 && score < 35 {
		score = 35
	}
	if score < 0 {
		score = 0
	}

	status := HealthStatusHealthy
	if criticalCount > 0 {
		status = HealthStatusCritical
	} else if warningCount > 0 {
		status = HealthStatusWarning
	}

	return newCollectorSignal(c, SignalTypeAlerts, status, &score, map[string]interface{}{
		"activeCount":   len(seen),
		"criticalCount": criticalCount,
		"warningCount":  warningCount,
		"infoCount":     infoCount,
		"fingerprints":  sortedAlertFingerprints(seen),
	}, ""), true
}

func activeAlertGroupsForCluster(clusterID string) []AlertGroup {
	alertsMu.RLock()
	defer alertsMu.RUnlock()

	out := make([]AlertGroup, 0)
	for _, g := range alertGroups {
		if g.ClusterID != clusterID {
			continue
		}
		if g.Status == AlertStatusActive || g.Status == AlertStatusAnalyzing {
			out = append(out, g)
		}
	}
	return out
}

func alertFingerprintForGroup(g AlertGroup) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s", g.Alertname, g.Service, g.Instance, g.ClusterID, g.Component)
}

func sortedAlertFingerprints(groups map[string]AlertGroup) []string {
	out := make([]string, 0, len(groups))
	for fp := range groups {
		out = append(out, fp)
	}
	sort.Strings(out)
	return out
}

func signalsForObject(signals []HealthSignal, objectType, objectID string) []HealthSignal {
	out := make([]HealthSignal, 0)
	for _, s := range signals {
		if s.ObjectType == objectType && s.ObjectID == objectID {
			out = append(out, s)
		}
	}
	return out
}

// AggregateHealthSnapshot applies DomainHealthPolicy to the latest fresh signal per type.
func AggregateHealthSnapshot(c Cluster, policy *DomainHealthPolicy, signals []HealthSignal) HealthSnapshot {
	now := time.Now().UTC()
	latest := latestFreshSignalsByType(signals, now)
	expected := expectedSourcesForCluster(c, policy, latest)
	present := presentSources(policy, latest)
	missing := missingSources(expected, latest)
	coverage := 0.0
	if len(expected) > 0 {
		coverage = float64(len(expected)-len(missing)) / float64(len(expected))
	}

	requiredOK := hasRequiredAnyOf(policy.RequiredAnyOf, latest)
	var score *int
	scoreStatus := ScoreStatusUnknown
	if !requiredOK {
		scoreStatus = ScoreStatusDegraded
	} else if coverage < policy.MinCoverageForScore {
		scoreStatus = ScoreStatusPartial
	} else if aggregateScore, ok := weightedScore(policy, latest); ok {
		score = &aggregateScore
		scoreStatus = scoreStatusFromScore(aggregateScore)
	} else {
		scoreStatus = ScoreStatusPartial
	}

	return HealthSnapshot{
		SchemaVersion:            "1",
		AggregationPolicyVersion: c.Domain + ":" + policy.PolicyVersion,
		ObjectType:               HealthObjectCluster,
		ObjectID:                 c.ID,
		ClusterID:                c.ID,
		Domain:                   c.Domain,
		Score:                    score,
		ScoreStatus:              scoreStatus,
		Source:                   "composite",
		Coverage:                 coverage,
		MissingSources:           missing,
		PresentSources:           present,
		Signals:                  mapSignals(latest),
		ObservedAt:               now.Format(time.RFC3339),
	}
}

func latestFreshSignalsByType(signals []HealthSignal, now time.Time) map[string]HealthSignal {
	latest := map[string]HealthSignal{}
	for _, s := range signals {
		if signalExpired(s, now) {
			s.Freshness = FreshnessExpired
			continue
		}
		existing, ok := latest[s.Type]
		if !ok || signalObservedTime(s).After(signalObservedTime(existing)) {
			latest[s.Type] = s
		}
	}
	return latest
}

func expectedSourcesForCluster(c Cluster, policy *DomainHealthPolicy, latest map[string]HealthSignal) []string {
	seen := map[string]struct{}{}
	for _, src := range policy.RequiredAnyOf {
		if _, ok := policy.Weights[src]; ok {
			seen[src] = struct{}{}
		}
	}
	for _, src := range policy.OptionalSources {
		if _, ok := policy.Weights[src]; !ok {
			continue
		}
		if optionalSourceConfigured(c, src, latest) {
			seen[src] = struct{}{}
		}
	}
	return sortedKeys(seen)
}

func optionalSourceConfigured(c Cluster, src string, latest map[string]HealthSignal) bool {
	if _, ok := latest[src]; ok {
		return true
	}
	switch src {
	case SignalTypeAssetStatus:
		return true
	case SignalTypeMetrics:
		return resolveClusterMetricsBaseURL(c) != "" || resolveVMBaseURL() != ""
	default:
		return false
	}
}

func presentSources(policy *DomainHealthPolicy, latest map[string]HealthSignal) []string {
	seen := map[string]struct{}{}
	for src := range latest {
		if _, ok := policy.Weights[src]; ok {
			seen[src] = struct{}{}
		}
	}
	return sortedKeys(seen)
}

func missingSources(expected []string, latest map[string]HealthSignal) []string {
	missing := make([]string, 0)
	for _, src := range expected {
		if _, ok := latest[src]; !ok {
			missing = append(missing, src)
		}
	}
	return missing
}

func hasRequiredAnyOf(required []string, latest map[string]HealthSignal) bool {
	if len(required) == 0 {
		return true
	}
	for _, src := range required {
		if s, ok := latest[src]; ok && s.Error == "" {
			return true
		}
	}
	return false
}

func weightedScore(policy *DomainHealthPolicy, latest map[string]HealthSignal) (int, bool) {
	var weighted float64
	var totalWeight float64
	for typ, signal := range latest {
		if signal.Score == nil || signal.Error != "" {
			continue
		}
		weight, ok := policy.Weights[typ]
		if !ok || weight <= 0 {
			continue
		}
		weighted += float64(*signal.Score) * weight
		totalWeight += weight
	}
	if totalWeight <= 0 {
		return 0, false
	}
	score := int(weighted/totalWeight + 0.5)
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score, true
}

func mapSignals(latest map[string]HealthSignal) []HealthSignal {
	out := make([]HealthSignal, 0, len(latest))
	for _, s := range latest {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Type < out[j].Type
	})
	return out
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
