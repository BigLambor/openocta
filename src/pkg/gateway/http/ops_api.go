package http

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/openocta/openocta/pkg/gateway/handlers"
	"github.com/openocta/openocta/pkg/jobrun"
	"github.com/openocta/openocta/pkg/ops"
	"github.com/openocta/openocta/pkg/rbac"
)

func opsWriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func opsWriteError(w http.ResponseWriter, status int, msg string) {
	opsWriteJSON(w, status, map[string]string{"error": msg})
}

func userHasDomainPermission(session *rbac.UserSession, domain string) bool {
	if session == nil || session.RoleName == "admin" {
		return true
	}
	for _, p := range session.Permissions {
		if p == "menu:"+domain {
			return true
		}
	}
	return false
}

func (s *Server) handleOpsListClusters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	domain := strings.TrimSpace(r.URL.Query().Get("domain"))
	if domain == "all" {
		domain = ""
	}

	session := GetUserSession(r)
	if session != nil && session.RoleName != "admin" {
		if domain != "" {
			if !userHasDomainPermission(session, domain) {
				opsWriteError(w, http.StatusForbidden, "没有该技术域的访问权限")
				return
			}
		}
	}

	list, err := ops.ListClusters(domain)
	if err != nil {
		opsWriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if session != nil && session.RoleName != "admin" {
		filtered := make([]ops.Cluster, 0)
		for _, c := range list {
			if userHasDomainPermission(session, c.Domain) {
				filtered = append(filtered, c)
			}
		}
		list = filtered
	}

	opsWriteJSON(w, http.StatusOK, map[string]interface{}{
		"clusters": list,
		"total":    len(list),
	})
}

func (s *Server) handleOpsGetCluster(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		opsWriteError(w, http.StatusBadRequest, "缺少集群 ID")
		return
	}
	c, err := ops.GetCluster(id)
	if err != nil {
		opsWriteError(w, http.StatusNotFound, err.Error())
		return
	}

	session := GetUserSession(r)
	if session != nil && session.RoleName != "admin" {
		if !userHasDomainPermission(session, c.Domain) {
			opsWriteError(w, http.StatusForbidden, "没有该技术域的访问权限")
			return
		}
	}

	opsWriteJSON(w, http.StatusOK, c)
}

func (s *Server) handleOpsCreateCluster(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body ops.ClusterCreate
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		opsWriteError(w, http.StatusBadRequest, "无效的 JSON 格式")
		return
	}
	c, err := ops.CreateCluster(body)
	if err != nil {
		opsWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	opsWriteJSON(w, http.StatusCreated, c)
}

func (s *Server) handleOpsPatchCluster(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		opsWriteError(w, http.StatusBadRequest, "缺少集群 ID")
		return
	}
	var patch ops.ClusterPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		opsWriteError(w, http.StatusBadRequest, "无效的 JSON 格式")
		return
	}
	c, err := ops.PatchCluster(id, patch)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "不存在") {
			status = http.StatusNotFound
		}
		opsWriteError(w, status, err.Error())
		return
	}
	opsWriteJSON(w, http.StatusOK, c)
}

func (s *Server) handleOpsSyncCMDB(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var rawClusters []map[string]interface{}
	var strategy string
	var mapping *ops.CMDBMapping

	if r.Body != nil {
		var body struct {
			Clusters []map[string]interface{} `json:"clusters"`
			Strategy string                   `json:"strategy"`
			Mapping  *ops.CMDBMapping         `json:"mapping"`
		}
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil && err != io.EOF {
			opsWriteError(w, http.StatusBadRequest, "无效的 JSON 格式")
			return
		}
		rawClusters = body.Clusters
		strategy = body.Strategy
		mapping = body.Mapping
	}
	result, err := ops.SyncClustersFromCMDB(r.Context(), rawClusters, strategy, mapping)
	if err != nil {
		opsWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	opsWriteJSON(w, http.StatusOK, result)
}

func (s *Server) handleOpsDeleteCluster(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		opsWriteError(w, http.StatusBadRequest, "缺少集群 ID")
		return
	}
	if err := ops.DeleteCluster(id); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "不存在") {
			status = http.StatusNotFound
		}
		opsWriteError(w, status, err.Error())
		return
	}
	opsWriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleOpsDashboardSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	summary := ops.BuildDashboardSummaryWithContext(r.Context())

	session := GetUserSession(r)
	if session != nil && session.RoleName != "admin" {
		filteredDomains := make([]ops.DomainHealthSummary, 0)
		var totalClusters, healthyClusters, warningClusters, criticalClusters int

		for _, d := range summary.Domains {
			if userHasDomainPermission(session, d.Domain) {
				filteredDomains = append(filteredDomains, d)
				totalClusters += d.ClusterCount
				healthyClusters += d.HealthyCount
				warningClusters += d.WarningCount
				criticalClusters += d.CriticalCount
			}
		}

		summary.Domains = filteredDomains
		summary.TotalClusters = totalClusters
		summary.HealthyClusters = healthyClusters
		summary.WarningClusters = warningClusters
		summary.CriticalClusters = criticalClusters

		// Recalculate pending alerts based on user permissions
		alertsList := ops.ListAlertGroups("", "")
		var pendingAlerts int
		for _, g := range alertsList.Groups {
			if userHasDomainPermission(session, g.Domain) && (g.Status == ops.AlertStatusActive || g.Status == ops.AlertStatusAnalyzing) {
				pendingAlerts++
			}
		}
		summary.PendingAlerts = pendingAlerts
	}

	opsWriteJSON(w, http.StatusOK, summary)
}

func (s *Server) handleOpsHealthSignals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	domain := strings.TrimSpace(r.URL.Query().Get("domain"))
	objectType := strings.TrimSpace(r.URL.Query().Get("objectType"))
	objectID := strings.TrimSpace(r.URL.Query().Get("objectId"))
	if domain == "all" {
		domain = ""
	}

	session := GetUserSession(r)
	if session != nil && session.RoleName != "admin" && domain != "" && !userHasDomainPermission(session, domain) {
		opsWriteError(w, http.StatusForbidden, "没有该技术域的访问权限")
		return
	}

	signals, err := ops.ListHealthSignals()
	if err != nil {
		opsWriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	filtered := make([]ops.HealthSignal, 0, len(signals))
	for _, sig := range signals {
		if domain != "" && sig.Domain != domain {
			continue
		}
		if objectType != "" && sig.ObjectType != objectType {
			continue
		}
		if objectID != "" && sig.ObjectID != objectID {
			continue
		}
		if session != nil && session.RoleName != "admin" && !userHasDomainPermission(session, sig.Domain) {
			continue
		}
		filtered = append(filtered, sig)
	}
	opsWriteJSON(w, http.StatusOK, map[string]interface{}{
		"signals": filtered,
		"total":   len(filtered),
	})
}

func (s *Server) handleOpsHealthSnapshots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	domain := strings.TrimSpace(r.URL.Query().Get("domain"))
	objectType := strings.TrimSpace(r.URL.Query().Get("objectType"))
	objectID := strings.TrimSpace(r.URL.Query().Get("objectId"))
	if domain == "all" {
		domain = ""
	}

	session := GetUserSession(r)
	if session != nil && session.RoleName != "admin" && domain != "" && !userHasDomainPermission(session, domain) {
		opsWriteError(w, http.StatusForbidden, "没有该技术域的访问权限")
		return
	}

	snapshots, err := ops.ListHealthSnapshots()
	if err != nil {
		opsWriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	filtered := make([]ops.HealthSnapshot, 0, len(snapshots))
	for _, snap := range snapshots {
		if domain != "" && snap.Domain != domain {
			continue
		}
		if objectType != "" && snap.ObjectType != objectType {
			continue
		}
		if objectID != "" && snap.ObjectID != objectID {
			continue
		}
		if session != nil && session.RoleName != "admin" && !userHasDomainPermission(session, snap.Domain) {
			continue
		}
		filtered = append(filtered, snap)
	}
	opsWriteJSON(w, http.StatusOK, map[string]interface{}{
		"snapshots": filtered,
		"total":     len(filtered),
	})
}

func (s *Server) handleOpsScenarios(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	domain := strings.TrimSpace(r.URL.Query().Get("domain"))
	if domain == "all" {
		domain = ""
	}
	session := GetUserSession(r)
	if session != nil && session.RoleName != "admin" && domain != "" && !userHasDomainPermission(session, domain) {
		opsWriteError(w, http.StatusForbidden, "没有该技术域的访问权限")
		return
	}
	all := ops.ListOpsScenarios()
	scenarios := make([]ops.OpsScenario, 0, len(all))
	for _, scenario := range all {
		if domain != "" && scenario.DomainKey != domain {
			continue
		}
		if session != nil && session.RoleName != "admin" && !userHasDomainPermission(session, scenario.DomainKey) {
			continue
		}
		scenarios = append(scenarios, scenario)
	}
	opsWriteJSON(w, http.StatusOK, map[string]interface{}{
		"scenarios": scenarios,
		"total":     len(scenarios),
	})
}

func (s *Server) handleOpsListAlertGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	domain := strings.TrimSpace(r.URL.Query().Get("domain"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if domain == "all" {
		domain = ""
	}

	session := GetUserSession(r)
	if session != nil && session.RoleName != "admin" {
		if domain != "" {
			if !userHasDomainPermission(session, domain) {
				opsWriteError(w, http.StatusForbidden, "没有该技术域的访问权限")
				return
			}
		}
	}

	res := ops.ListAlertGroups(domain, status)

	if session != nil && session.RoleName != "admin" {
		filtered := make([]ops.AlertGroup, 0)
		var originalTotal int
		pendingActive := 0
		for _, g := range res.Groups {
			if userHasDomainPermission(session, g.Domain) {
				filtered = append(filtered, g)
				originalTotal += g.OriginalCount
				if g.Status == ops.AlertStatusActive || g.Status == ops.AlertStatusAnalyzing {
					pendingActive++
				}
			}
		}

		merged := len(filtered)
		var rate float64
		if originalTotal > 0 && merged > 0 {
			rate = (1 - float64(merged)/float64(originalTotal)) * 100
			if rate < 0 {
				rate = 0
			}
		}

		res.Groups = filtered
		res.Total = merged
		res.OriginalTotal = originalTotal
		res.MergedTotal = merged
		res.ReductionRate = rate
		res.PendingActive = pendingActive
	}

	opsWriteJSON(w, http.StatusOK, res)
}

func (s *Server) handleOpsGetAlertGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		opsWriteError(w, http.StatusBadRequest, "缺少告警组 ID")
		return
	}
	g, err := ops.GetAlertGroup(id)
	if err != nil {
		opsWriteError(w, http.StatusNotFound, err.Error())
		return
	}

	session := GetUserSession(r)
	if session != nil && session.RoleName != "admin" {
		if !userHasDomainPermission(session, g.Domain) {
			opsWriteError(w, http.StatusForbidden, "没有该技术域的访问权限")
			return
		}
	}

	opsWriteJSON(w, http.StatusOK, g)
}

func (s *Server) handleOpsPatchAlertGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		opsWriteError(w, http.StatusBadRequest, "缺少告警组 ID")
		return
	}

	g, err := ops.GetAlertGroup(id)
	if err != nil {
		opsWriteError(w, http.StatusNotFound, err.Error())
		return
	}

	session := GetUserSession(r)
	if session != nil && session.RoleName != "admin" {
		if !userHasDomainPermission(session, g.Domain) {
			opsWriteError(w, http.StatusForbidden, "没有该技术域的访问权限")
			return
		}
	}

	var patch ops.AlertGroupPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		opsWriteError(w, http.StatusBadRequest, "无效的 JSON 格式")
		return
	}
	operator := "system"
	if session != nil {
		operator = session.Username
	}
	updatedGroup, err := ops.PatchAlertGroup(id, patch, operator)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "不存在") {
			status = http.StatusNotFound
		}
		opsWriteError(w, status, err.Error())
		return
	}
	opsWriteJSON(w, http.StatusOK, updatedGroup)
}

func (s *Server) handleOpsInspectionIMStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	enabled := listEnabledIMChannels(s.ctx)
	opsWriteJSON(w, http.StatusOK, ops.InspectionIMStatusFromChannels(enabled))
}

func (s *Server) handleOpsListJobRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	svc := jobrun.Default()
	if svc == nil {
		opsWriteJSON(w, http.StatusOK, map[string]interface{}{
			"runs":  []jobrun.JobRun{},
			"total": 0,
		})
		return
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	filter := jobrun.ListFilter{
		JobID:       strings.TrimSpace(r.URL.Query().Get("jobId")),
		TriggerType: strings.TrimSpace(r.URL.Query().Get("triggerType")),
		TriggerRef:  strings.TrimSpace(r.URL.Query().Get("triggerRef")),
		Limit:       limit,
	}
	runs, err := svc.List(filter)
	if err != nil {
		opsWriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if runs == nil {
		runs = []jobrun.JobRun{}
	}
	opsWriteJSON(w, http.StatusOK, map[string]interface{}{
		"runs":  runs,
		"total": len(runs),
	})
}

func (s *Server) handleOpsGetJobRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		opsWriteError(w, http.StatusBadRequest, "缺少 JobRun ID")
		return
	}
	svc := jobrun.Default()
	if svc == nil {
		opsWriteError(w, http.StatusServiceUnavailable, "JobRun 服务未初始化")
		return
	}
	detail, err := svc.GetDetail(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			opsWriteError(w, http.StatusNotFound, "JobRun 不存在")
			return
		}
		opsWriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	opsWriteJSON(w, http.StatusOK, detail)
}

func listEnabledIMChannels(ctx *handlers.Context) []string {
	if ctx == nil || ctx.Config == nil || ctx.Config.Channels == nil {
		return nil
	}
	var out []string
	for _, id := range []string{"feishu", "dingtalk"} {
		if c := ctx.Config.Channels.GetChannelConfig(id); c != nil {
			if en, _ := c["enabled"].(bool); en {
				out = append(out, id)
			}
		}
	}
	return out
}
