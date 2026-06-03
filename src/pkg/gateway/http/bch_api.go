package http

import (
	"net/http"
	"strings"

	"github.com/openocta/openocta/pkg/ops"
)

func getBchService() ops.BchService {
	// Future extension: if os.Getenv("BCH_DATA_PROVIDER") == "real" { return ops.NewRealBchService() }
	return ops.NewMockBchService()
}

func (s *Server) handleBchClustersHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	svc := getBchService()
	data, err := svc.GetClustersHealth()
	if err != nil {
		opsWriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	opsWriteJSON(w, http.StatusOK, data)
}

func (s *Server) handleBchListFlinkJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	svc := getBchService()
	data, err := svc.ListFlinkJobs()
	if err != nil {
		opsWriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	opsWriteJSON(w, http.StatusOK, data)
}

func (s *Server) handleBchGetFlinkJobConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		opsWriteError(w, http.StatusBadRequest, "缺少 Flink 作业 ID")
		return
	}
	svc := getBchService()
	configStr, err := svc.GetFlinkJobConfig(id)
	if err != nil {
		opsWriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(configStr))
}

func (s *Server) handleBchDiagnoseFlinkJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		opsWriteError(w, http.StatusBadRequest, "缺少 Flink 作业 ID")
		return
	}
	svc := getBchService()
	data, err := svc.DiagnoseFlinkJob(id)
	if err != nil {
		opsWriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	opsWriteJSON(w, http.StatusOK, data)
}

func (s *Server) handleBchListSparkJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	svc := getBchService()
	data, err := svc.ListSparkJobs()
	if err != nil {
		opsWriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	opsWriteJSON(w, http.StatusOK, data)
}

func (s *Server) handleBchTuneSparkJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		opsWriteError(w, http.StatusBadRequest, "缺少 Spark 作业 ID")
		return
	}
	svc := getBchService()
	data, err := svc.TuneSparkJob(id)
	if err != nil {
		opsWriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	opsWriteJSON(w, http.StatusOK, data)
}

func (s *Server) handleBchGetHdfsFsImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	namespace := strings.TrimSpace(r.URL.Query().Get("namespace"))
	if namespace == "" {
		namespace = "NS1"
	}
	svc := getBchService()
	data, err := svc.GetHdfsFsImage(namespace)
	if err != nil {
		opsWriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	opsWriteJSON(w, http.StatusOK, data)
}

func (s *Server) handleBchListEmployees(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	svc := getBchService()
	data, err := svc.ListEmployees()
	if err != nil {
		opsWriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	opsWriteJSON(w, http.StatusOK, data)
}
