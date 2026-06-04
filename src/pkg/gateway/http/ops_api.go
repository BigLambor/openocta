package http

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/openocta/openocta/pkg/gateway/handlers"
	"github.com/openocta/openocta/pkg/ops"
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

func (s *Server) handleOpsListClusters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	domain := strings.TrimSpace(r.URL.Query().Get("domain"))
	list, err := ops.ListClusters(domain)
	if err != nil {
		opsWriteError(w, http.StatusInternalServerError, err.Error())
		return
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
	opsWriteJSON(w, http.StatusOK, ops.BuildDashboardSummaryWithContext(r.Context()))
}

func (s *Server) handleOpsListAlertGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	domain := strings.TrimSpace(r.URL.Query().Get("domain"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	opsWriteJSON(w, http.StatusOK, ops.ListAlertGroups(domain, status))
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
	var patch ops.AlertGroupPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		opsWriteError(w, http.StatusBadRequest, "无效的 JSON 格式")
		return
	}
	operator := "system"
	if session := GetUserSession(r); session != nil {
		operator = session.Username
	}
	g, err := ops.PatchAlertGroup(id, patch, operator)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "不存在") {
			status = http.StatusNotFound
		}
		opsWriteError(w, status, err.Error())
		return
	}
	opsWriteJSON(w, http.StatusOK, g)
}

func (s *Server) handleOpsInspectionIMStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	enabled := listEnabledIMChannels(s.ctx)
	opsWriteJSON(w, http.StatusOK, ops.InspectionIMStatusFromChannels(enabled))
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
