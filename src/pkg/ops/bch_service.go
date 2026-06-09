package ops

// BchClusterHealth represents the health status of a BCH cluster.
type BchClusterHealth struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Region         string                 `json:"region"`
	Status         string                 `json:"status"` // healthy, warning, critical
	Score          int                    `json:"score"`
	ScoreStatus    string                 `json:"scoreStatus"`
	Coverage       float64                `json:"coverage"`
	Freshness      string                 `json:"freshness"`
	PresentSources []string               `json:"presentSources"`
	MissingSources []string               `json:"missingSources"`
	NodeCount      int                    `json:"nodeCount"`
	ActiveAlerts   int                    `json:"activeAlerts"`
	CpuUsedPercent float64                `json:"cpuUsedPercent"`
	MemUsedPercent float64                `json:"memUsedPercent"`
	DfsUsedPercent float64                `json:"dfsUsedPercent"`
	Metrics        map[string]interface{} `json:"metrics"`
}


// FlinkJobMetric represents metrics for a Flink Job
type FlinkJobMetric struct {
	LagTrend        int   `json:"lagTrend"`
	MaxLag          int64 `json:"maxLag"`
	AvgLag          int64 `json:"avgLag"`
	IsBackpressured bool  `json:"isBP"`
	CpuMax          int   `json:"cpuMax"`
	CpuAvg          int   `json:"cpuAvg"`
	HeapMax         int   `json:"heapMax"`
	FullGcCount     int   `json:"fullGcCount"`
	Restarts        int   `json:"restarts"`
}

// FlinkJob Penalty Breakdown
type FlinkPenalty struct {
	Item      string `json:"item"`
	Deduction int    `json:"deduction"`
	Type      string `json:"type"` // fatal, warning, info
}

// FlinkJob CoT step data
type FlinkCotStep struct {
	Text  string `json:"text"`
	State string `json:"state"` // active, warning, critical
}

// FlinkJob CoT steps
type FlinkCotSteps struct {
	Step1 FlinkCotStep `json:"step1"`
	Step2 FlinkCotStep `json:"step2"`
	Step3 FlinkCotStep `json:"step3"`
}

// FlinkJob represents a Flink job in BCH domain
type FlinkJob struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Owner         string         `json:"owner"`
	Cluster       string         `json:"cluster"`
	Status        string         `json:"status"` // RUNNING, FAILED
	Score         int            `json:"score"`
	SScore        int            `json:"sScore"` // Stability score
	PScore        int            `json:"pScore"` // Performance score
	EScore        int            `json:"eScore"` // Efficiency score
	Metrics       FlinkJobMetric `json:"metrics"`
	Penalties     []FlinkPenalty `json:"penalties"`
	Diagnosis     string         `json:"diagnosis"`
	RootCause     string         `json:"rootCause"` // S0, S1, S2, etc.
	RootCauseText string         `json:"rootCauseText"`
	Actions       []string       `json:"actions"`
	CotSteps      FlinkCotSteps  `json:"cotSteps"`
}

// SparkJobMetric represents metrics for a Spark Job
type SparkJobMetric struct {
	ExecutorMemoryOverheadMB int     `json:"executorMemoryOverheadMB"`
	MaxTaskDurationSec       int     `json:"maxTaskDurationSec"`
	AvgTaskDurationSec       int     `json:"avgTaskDurationSec"`
	TotalTasks               int     `json:"totalTasks"`
	FailedTasks              int     `json:"failedTasks"`
	CpuSkewRatio             float64 `json:"cpuSkewRatio"`
	MemorySkewRatio          float64 `json:"memorySkewRatio"`
	InputBytes               int64   `json:"inputBytes"`
	ShuffleReadBytes         int64   `json:"shuffleReadBytes"`
	ShuffleWriteBytes        int64   `json:"shuffleWriteBytes"`
}

// SparkJob represents a Spark job in BCH domain
type SparkJob struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Owner        string         `json:"owner"`
	Cluster      string         `json:"cluster"`
	Status       string         `json:"status"` // SUCCEEDED, RUNNING, FAILED
	Labels       []string       `json:"labels"` // 数据倾斜, 资源倾斜, etc.
	DurationSec  int            `json:"durationSec"`
	Metrics      SparkJobMetric `json:"metrics"`
	TuningAdvice string         `json:"tuningAdvice"`
}

// HdfsFsImageDepthStats represents HDFS directory depth stats
type HdfsFsImageDepthStats struct {
	Depth   string  `json:"depth"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

// HdfsFsImageSizeStats represents HDFS file size stats
type HdfsFsImageSizeStats struct {
	Size    string  `json:"size"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

// HdfsFsImageUserStats represents HDFS user usage stats
type HdfsFsImageUserStats struct {
	User    string  `json:"user"`
	Files   int64   `json:"files"`
	Percent float64 `json:"percent"`
	Size    string  `json:"size"`
}

// HdfsFsImageTimeStats represents HDFS modification/access time stats
type HdfsFsImageTimeStats struct {
	Period  string  `json:"period"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

// HdfsFsImageFileTypeStats represents HDFS file type stats
type HdfsFsImageFileTypeStats struct {
	Ext     string  `json:"ext"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

// HdfsFsImagePathPattern represents HDFS path patterns
type HdfsFsImagePathPattern struct {
	Path    string  `json:"path"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

// HdfsFsImageStats represents HDFS FSImage depth analysis result
type HdfsFsImageStats struct {
	Namespace       string                     `json:"namespace"`
	TotalRecords    string                     `json:"totalRecords"`
	TotalFiles      string                     `json:"totalFiles"`
	TotalDirs       string                     `json:"totalDirs"`
	TotalSize       string                     `json:"totalSize"`
	AvgFileSize     string                     `json:"avgFileSize"`
	MaxDepth        string                     `json:"maxDepth"`
	ProcessingTime  string                     `json:"processingTime"`
	ProcessingSpeed string                     `json:"processingSpeed"`
	DepthData       []HdfsFsImageDepthStats    `json:"depthData"`
	SizeData        []HdfsFsImageSizeStats     `json:"sizeData"`
	UserData        []HdfsFsImageUserStats     `json:"userData"`
	ModifyData      []HdfsFsImageTimeStats     `json:"modifyData"`
	AccessData      []HdfsFsImageTimeStats     `json:"accessData"`
	FileTypeData    []HdfsFsImageFileTypeStats `json:"fileTypeData"`
	PathData        []HdfsFsImagePathPattern   `json:"pathData"`
	ZeroByteFiles   int64                      `json:"zeroByteFiles"`
	TrashFiles      int64                      `json:"trashFiles"`
}

// BchEmployeeTask represents recent task of digital employee
type BchEmployeeTask struct {
	Time   string `json:"time"`
	Task   string `json:"task"`
	Result string `json:"result"`
}

// BchEmployee represents a BCH digital employee
type BchEmployee struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Status      string            `json:"status"` // idle, working
	StatusDesc  string            `json:"statusDesc"`
	Description string            `json:"description"`
	Skills      []string          `json:"skills"`
	Tools       []string          `json:"tools"`
	RecentTasks []BchEmployeeTask `json:"recentTasks"`
}

// YarnQueueMetric represents historical resource utilization metrics of a YARN queue.
type YarnQueueMetric struct {
	AvgCpuPercent float64 `json:"avgCpuPercent"`
	MaxCpuPercent float64 `json:"maxCpuPercent"`
	AvgMemPercent float64 `json:"avgMemPercent"`
	MaxMemPercent float64 `json:"maxMemPercent"`
	ActiveApps    int     `json:"activeApps"`
}

// YarnQueueEvaluation represents a YARN queue capacity evaluation.
type YarnQueueEvaluation struct {
	ID                 string          `json:"id"`
	Name               string          `json:"name"`
	Cluster            string          `json:"cluster"`
	Metrics            YarnQueueMetric `json:"metrics"`
	Status             string          `json:"status"` // idle, over_allocated, healthy, under_allocated
	RiskLevel          string          `json:"riskLevel"` // low, medium, high
	CurrentCapacity    float64         `json:"currentCapacity"`
	MaxCapacity        float64         `json:"maxCapacity"`
	UsedCapacity       float64         `json:"usedCapacity"`
	PendingContainers  int             `json:"pendingContainers"`
	WaitingApps        int             `json:"waitingApps"`
	PeakUsage30d       float64         `json:"peakUsage30d"`
	LastActiveTime     string          `json:"lastActiveTime"`
	Reasons            []string        `json:"reasons"`
	Advice             string          `json:"advice"`
	Action             string          `json:"action"` // reclaim, downsize, expand, none
	TargetCapacity     float64         `json:"targetCapacity"`
	TargetMaxCapacity  float64         `json:"targetMaxCapacity"`
	ConfigPatch        string          `json:"configPatch"`
	RollbackPlan       string          `json:"rollbackPlan"`
}

// BchService manages BCH ecosystem operations
type BchService interface {
	GetClustersHealth() ([]BchClusterHealth, error)
	ListFlinkJobs() ([]FlinkJob, error)
	GetFlinkJobConfig(id string) (string, error)
	DiagnoseFlinkJob(id string) (*FlinkJob, error)
	ListSparkJobs() ([]SparkJob, error)
	TuneSparkJob(id string) (*SparkJob, error)
	GetHdfsFsImage(namespace string) (*HdfsFsImageStats, error)
	ListEmployees() ([]BchEmployee, error)
	ListYarnQueues() ([]YarnQueueEvaluation, error)
	ExecuteYarnQueueAction(id string) (bool, error)
	RollbackYarnQueueAction(id string) (bool, error)
}
