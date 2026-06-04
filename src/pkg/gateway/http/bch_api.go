package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/openocta/openocta/pkg/gateway/handlers"
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

func (s *Server) handleBchChatFlinkJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		opsWriteError(w, http.StatusBadRequest, "缺少 Flink 作业 ID")
		return
	}

	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		opsWriteError(w, http.StatusBadRequest, "无效的请求参数")
		return
	}
	if strings.TrimSpace(body.Message) == "" {
		opsWriteError(w, http.StatusBadRequest, "消息内容不能为空")
		return
	}

	svc := getBchService()
	job, err := svc.DiagnoseFlinkJob(id)
	if err != nil {
		opsWriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 构造模型提示词，注入作业运行指标上下文
	systemPrompt := fmt.Sprintf(`你是一个专业的 BCH 流计算作业诊断数字员工。
当前正在针对 Flink 作业「%s」（Owner: %s）进行即时问询。

该作业的健康状态上下文如下：
- 综合健康评分：%d 分
- 诊断结论：%s
- 根因分类：%s (%s)
- 建议动作：%s
- 运行时指标：
  * 重启次数：%d 次/小时
  * Full GC 次数：%d 次/小时
  * 消费 Lag 趋势：斜率 %d (最大积压 %d, 平均积压 %d)
  * CPU 使用率：最大 %d%%, 平均 %d%%
  * 堆内存使用率：%d%%

用户向你提问："%s"
请结合以上客观监控数据与 Flink 专业运维知识，给出专业、具体、针对性强的调优解答。你的回答应该专业、精炼，条理分明，不要包含废话。`,
		job.Name, job.Owner, job.Score, job.Diagnosis, job.RootCauseText, job.RootCause, strings.Join(job.Actions, "; "),
		job.Metrics.Restarts, job.Metrics.FullGcCount, job.Metrics.LagTrend, job.Metrics.MaxLag, job.Metrics.AvgLag,
		job.Metrics.CpuMax, job.Metrics.CpuAvg, job.Metrics.HeapMax,
		body.Message,
	)

	// 调用平台自带的 RunCronAgentOnce 执行真实大模型调用，获取答案
	reply, err := handlers.RunCronAgentOnce(s.ctx, "default", "bch-flink-chat-"+id, systemPrompt)
	if err != nil {
		opsWriteError(w, http.StatusInternalServerError, "AI 模型调用失败: "+err.Error())
		return
	}

	opsWriteJSON(w, http.StatusOK, map[string]string{
		"reply": reply,
	})
}
