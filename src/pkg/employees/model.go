package employees

import "github.com/openocta/openocta/pkg/config"

// Manifest 描述一个数字员工模板的元数据。
// 对于内置数字员工，manifest 可能是虚拟构造的（仅内存中存在）。
type Manifest struct {
	// ID 是员工的稳定标识符，同时也用作会话 key 中的类型字段（employee-<ID>-...）。
	ID string `json:"id"`
	// Name 是展示给用户的名称。
	Name string `json:"name"`
	// Description 是员工的功能描述。
	Description string `json:"description"`
	// Prompt 是该数字员工的系统提示/人设说明。
	Prompt string `json:"prompt,omitempty"`
	// Enabled 标记该模板当前是否可用。
	Enabled bool `json:"enabled,omitempty"`
	// CreatedAt 记录模板创建时间（Unix 毫秒）。
	CreatedAt int64 `json:"createdAt,omitempty"`
	// Builtin 标记是否为内置员工（历史兼容，后端不再 embed 默认员工）。
	Builtin bool `json:"builtin"`
	// SkillIDs 是与该员工关联的全局 skill 名称（可来自 ~/.openocta/skills 或 workspace skills）。
	SkillIDs []string `json:"skillIds,omitempty"`
	// ScenarioKeys 是与该员工关联的 Scenario 名称（可直接在运行时挂载这些场景能力）。
	ScenarioKeys []string `json:"scenarioKeys,omitempty"`
	// McpServers 为该员工专属的 MCP 服务器配置，与会话时与全局 mcp.servers 合并（同 key 时员工覆盖全局）。
	McpServers map[string]config.McpServerEntry `json:"mcpServers,omitempty"`
	// Type 表示员工所属类型/分类；空表示「其它」。自建员工创建时为空；从官网下载时写入其 category。
	Type string `json:"type,omitempty"`
	// From 表示来源：local（自建）、remote（官网安装）等。
	From string `json:"from,omitempty"`
	// DomainKeys 表示该员工服务的技术域，例如 hadoop、gbase、fi。
	DomainKeys []string `json:"domainKeys,omitempty"`
	// CapabilityKeys 表示该员工归属的运维能力域，例如 observability、inspection、diagnosis。
	CapabilityKeys []string `json:"capabilityKeys,omitempty"`
	// RoleType 表示员工岗位类型，例如 oncall、inspector、diagnoser、governor。
	RoleType string `json:"roleType,omitempty"`
	// Responsibilities 描述该员工的岗位职责。
	Responsibilities []string `json:"responsibilities,omitempty"`
	// InputSources 描述该员工可接收的输入来源，例如 alerts、metrics、logs、tickets。
	InputSources []string `json:"inputSources,omitempty"`
	// OutputTypes 描述该员工的输出成果，例如 diagnosis_report、inspection_report。
	OutputTypes []string `json:"outputTypes,omitempty"`
	// ActionScopes 描述该员工允许执行的动作范围，例如 read_only、create_ticket、run_script。
	ActionScopes []string `json:"actionScopes,omitempty"`
	// PermissionKeys 描述该员工需要或具备的权限点。
	PermissionKeys []string `json:"permissionKeys,omitempty"`
	// RunbookRefs 是该员工关联的 Runbook 引用。
	RunbookRefs []string `json:"runbookRefs,omitempty"`
	// KnowledgeRefs 是该员工关联的知识库引用。
	KnowledgeRefs []string `json:"knowledgeRefs,omitempty"`
}

// Summary 是前端列表展示用的精简结构。
type Summary struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Prompt           string   `json:"prompt,omitempty"`
	Enabled          bool     `json:"enabled,omitempty"`
	CreatedAt        int64    `json:"createdAt,omitempty"`
	Builtin          bool     `json:"builtin"`
	SkillIDs         []string `json:"skillIds,omitempty"`
	ScenarioKeys     []string `json:"scenarioKeys,omitempty"`
	SkillNames       []string `json:"skillNames,omitempty"`    // 展示用：manifest.skillIds + employee_skills 目录名
	McpServerKeys    []string `json:"mcpServerKeys,omitempty"` // 展示用：manifest.mcpServers 的 key 列表
	Type             string   `json:"type,omitempty"`          // 所属类型/分类，空表示「其它」
	From             string   `json:"from,omitempty"`          // 来源：local、remote 等
	DomainKeys       []string `json:"domainKeys,omitempty"`
	CapabilityKeys   []string `json:"capabilityKeys,omitempty"`
	RoleType         string   `json:"roleType,omitempty"`
	Responsibilities []string `json:"responsibilities,omitempty"`
	InputSources     []string `json:"inputSources,omitempty"`
	OutputTypes      []string `json:"outputTypes,omitempty"`
	ActionScopes     []string `json:"actionScopes,omitempty"`
	PermissionKeys   []string `json:"permissionKeys,omitempty"`
	RunbookRefs      []string `json:"runbookRefs,omitempty"`
	KnowledgeRefs    []string `json:"knowledgeRefs,omitempty"`
}
