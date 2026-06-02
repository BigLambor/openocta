package handlers

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/openocta/openocta/pkg/config"
	"github.com/openocta/openocta/pkg/employees"
	"github.com/openocta/openocta/pkg/gateway/protocol"
	"github.com/openocta/openocta/pkg/installmetadata"
)

func normalizeEmployeeID(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	// 兼容目录/市场层 local:<id>
	if strings.HasPrefix(strings.ToLower(s), "local:") {
		s = s[len("local:"):]
	}
	s = strings.TrimSpace(s)
	// employee 会话 key 以 ":" 分隔，id 中不应包含冒号；这里做最小兼容。
	s = strings.ReplaceAll(s, ":", "-")
	return s
}

// EmployeesListHandler 处理 "employees.list"：返回所有数字员工模板（内置 + 用户自建）。
func EmployeesListHandler(opts HandlerOpts) error {
	env := func(k string) string { return os.Getenv(k) }
	list, err := employees.ListSummaries(env)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: "employees.list: " + err.Error(),
		}, nil)
		return nil
	}
	opts.Respond(true, map[string]interface{}{
		"employees": list,
	}, nil, nil)
	return nil
}

// EmployeesGetHandler 处理 "employees.get"：根据 id 返回单个数字员工 manifest。
func EmployeesGetHandler(opts HandlerOpts) error {
	rawID, _ := opts.Params["id"].(string)
	id := normalizeEmployeeID(rawID)
	env := func(k string) string { return os.Getenv(k) }
	m, err := employees.LoadManifest(id, env)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeNotFound,
			Message: "employees.get: employee not found",
		}, nil)
		return nil
	}
	opts.Respond(true, m, nil, nil)
	return nil
}

// EmployeesCreateHandler 处理 "employees.create"：创建或更新一个用户数字员工模板。
// 目前支持名称/描述/prompt/enabled 以及从现有 skills 中选择的 skillIds，skills 文件上传复用独立的 /api/skills/upload。
// 当仅传入 id 和 enabled 时（如禁用/启用切换），会合并已有 manifest，仅更新 enabled。
func EmployeesCreateHandler(opts HandlerOpts) error {
	name, _ := opts.Params["name"].(string)
	desc, _ := opts.Params["description"].(string)
	rawID, _ := opts.Params["id"].(string)
	prompt, _ := opts.Params["prompt"].(string)
	typeVal, _ := opts.Params["type"].(string)
	fromVal, _ := opts.Params["from"].(string)
	roleType, _ := opts.Params["roleType"].(string)
	enabledVal, hasEnabled := opts.Params["enabled"].(bool)
	enabled := true
	if hasEnabled {
		enabled = enabledVal
	}
	var skillIDs []string
	if raw, ok := opts.Params["skillIds"].([]interface{}); ok {
		for _, v := range raw {
			if s, ok := v.(string); ok && s != "" {
				skillIDs = append(skillIDs, s)
			}
		}
	}
	domainKeys, hasDomainKeys := stringSliceParam(opts.Params, "domainKeys")
	capabilityKeys, hasCapabilityKeys := stringSliceParam(opts.Params, "capabilityKeys")
	responsibilities, hasResponsibilities := stringSliceParam(opts.Params, "responsibilities")
	inputSources, hasInputSources := stringSliceParam(opts.Params, "inputSources")
	outputTypes, hasOutputTypes := stringSliceParam(opts.Params, "outputTypes")
	actionScopes, hasActionScopes := stringSliceParam(opts.Params, "actionScopes")
	permissionKeys, hasPermissionKeys := stringSliceParam(opts.Params, "permissionKeys")
	runbookRefs, hasRunbookRefs := stringSliceParam(opts.Params, "runbookRefs")
	knowledgeRefs, hasKnowledgeRefs := stringSliceParam(opts.Params, "knowledgeRefs")
	var mcpServers map[string]config.McpServerEntry
	if rawMcp, ok := opts.Params["mcpServers"]; ok && rawMcp != nil {
		if data, err := json.Marshal(rawMcp); err == nil {
			_ = json.Unmarshal(data, &mcpServers)
		}
	}

	env := func(k string) string { return os.Getenv(k) }
	id := normalizeEmployeeID(rawID)
	if id == "" {
		id = deriveEmployeeIDFromName(name)
	}
	// 新建时校验名称唯一性（rawID 为空表示创建新员工）
	if rawID == "" && name != "" {
		existingList, _ := employees.ListSummaries(env)
		for _, e := range existingList {
			if strings.EqualFold(strings.TrimSpace(e.ID), id) {
				opts.Respond(false, nil, &protocol.ErrorShape{
					Code:    protocol.ErrCodeInvalidRequest,
					Message: "名称已存在，请使用其他名称（名称唯一）",
				}, nil)
				return nil
			}
		}
	}

	// 仅更新 enabled 时（如禁用/启用），合并已有 manifest
	profilePatchPresent := roleType != "" ||
		hasDomainKeys ||
		hasCapabilityKeys ||
		hasResponsibilities ||
		hasInputSources ||
		hasOutputTypes ||
		hasActionScopes ||
		hasPermissionKeys ||
		hasRunbookRefs ||
		hasKnowledgeRefs
	updateEnabledOnly := hasEnabled && name == "" && desc == "" && prompt == "" && len(skillIDs) == 0 && mcpServers == nil && !profilePatchPresent
	var m *employees.Manifest
	if updateEnabledOnly {
		existing, err := employees.LoadManifest(id, env)
		if err == nil && existing != nil && !existing.Builtin {
			m = existing
			m.Enabled = enabled
		}
	} else if rawID != "" {
		// 编辑模式：加载已有 manifest 并合并传入字段（名称不可改）
		existing, err := employees.LoadManifest(id, env)
		if err == nil && existing != nil && !existing.Builtin {
			m = existing
			m.Description = desc
			m.Prompt = prompt
			m.Enabled = enabled
			if len(skillIDs) > 0 {
				m.SkillIDs = skillIDs
			}
			if mcpServers != nil {
				m.McpServers = mcpServers
			}
			if typeVal != "" {
				m.Type = strings.TrimSpace(typeVal)
			}
			if fromVal != "" {
				m.From = strings.TrimSpace(fromVal)
			}
			applyEmployeeProfilePatch(
				m,
				roleType,
				domainKeys, hasDomainKeys,
				capabilityKeys, hasCapabilityKeys,
				responsibilities, hasResponsibilities,
				inputSources, hasInputSources,
				outputTypes, hasOutputTypes,
				actionScopes, hasActionScopes,
				permissionKeys, hasPermissionKeys,
				runbookRefs, hasRunbookRefs,
				knowledgeRefs, hasKnowledgeRefs,
			)
		}
	}
	if m == nil {
		typeTrimmed := strings.TrimSpace(typeVal)
		fromTrimmed := strings.TrimSpace(fromVal)
		m = &employees.Manifest{
			ID:               id,
			Name:             name,
			Description:      desc,
			Prompt:           prompt,
			Enabled:          enabled,
			Builtin:          false,
			SkillIDs:         skillIDs,
			McpServers:       mcpServers,
			Type:             typeTrimmed, // 空表示「其它」
			From:             fromTrimmed,
			DomainKeys:       domainKeys,
			CapabilityKeys:   capabilityKeys,
			RoleType:         strings.TrimSpace(roleType),
			Responsibilities: responsibilities,
			InputSources:     inputSources,
			OutputTypes:      outputTypes,
			ActionScopes:     actionScopes,
			PermissionKeys:   permissionKeys,
			RunbookRefs:      runbookRefs,
			KnowledgeRefs:    knowledgeRefs,
		}
	}
	if err := employees.SaveManifest(m, env); err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: "employees.create: " + err.Error(),
		}, nil)
		return nil
	}
	opts.Respond(true, map[string]interface{}{"id": m.ID}, nil, nil)
	return nil
}

func stringSliceParam(params map[string]interface{}, key string) ([]string, bool) {
	raw, ok := params[key]
	if !ok || raw == nil {
		return nil, false
	}
	out := []string{}
	switch v := raw.(type) {
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				if s = strings.TrimSpace(s); s != "" {
					out = append(out, s)
				}
			}
		}
	case []string:
		for _, s := range v {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
	case string:
		for _, s := range strings.Split(v, ",") {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
	}
	return out, true
}

func applyEmployeeProfilePatch(
	m *employees.Manifest,
	roleType string,
	domainKeys []string, hasDomainKeys bool,
	capabilityKeys []string, hasCapabilityKeys bool,
	responsibilities []string, hasResponsibilities bool,
	inputSources []string, hasInputSources bool,
	outputTypes []string, hasOutputTypes bool,
	actionScopes []string, hasActionScopes bool,
	permissionKeys []string, hasPermissionKeys bool,
	runbookRefs []string, hasRunbookRefs bool,
	knowledgeRefs []string, hasKnowledgeRefs bool,
) {
	if m == nil {
		return
	}
	if strings.TrimSpace(roleType) != "" {
		m.RoleType = strings.TrimSpace(roleType)
	}
	if hasDomainKeys {
		m.DomainKeys = domainKeys
	}
	if hasCapabilityKeys {
		m.CapabilityKeys = capabilityKeys
	}
	if hasResponsibilities {
		m.Responsibilities = responsibilities
	}
	if hasInputSources {
		m.InputSources = inputSources
	}
	if hasOutputTypes {
		m.OutputTypes = outputTypes
	}
	if hasActionScopes {
		m.ActionScopes = actionScopes
	}
	if hasPermissionKeys {
		m.PermissionKeys = permissionKeys
	}
	if hasRunbookRefs {
		m.RunbookRefs = runbookRefs
	}
	if hasKnowledgeRefs {
		m.KnowledgeRefs = knowledgeRefs
	}
}

// EmployeesDeleteHandler 处理 "employees.delete"：删除用户自建数字员工及其关联会话。
func EmployeesDeleteHandler(opts HandlerOpts) error {
	rawID, _ := opts.Params["id"].(string)
	id := normalizeEmployeeID(rawID)
	if id == "" {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "employees.delete: id required",
		}, nil)
		return nil
	}
	env := func(k string) string { return os.Getenv(k) }

	// 删除该数字员工关联的会话（sessions.json 中 key 含 employee:id 的条目）
	if err := DeleteSessionsForEmployeeID(id, opts.Context); err != nil {
		// 记录日志但不阻断删除
		_ = err
	}

	if err := employees.DeleteEmployee(id, env); err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: "employees.delete: " + err.Error(),
		}, nil)
		return nil
	}
	_ = installmetadata.RemoveByLocalID(env, "employee", id)
	opts.Respond(true, map[string]interface{}{"ok": true}, nil, nil)
	return nil
}

// deriveEmployeeIDFromName 从名称推导一个稳定的 ID（小写、[a-z0-9_-]、长度 <=64）。
func deriveEmployeeIDFromName(name string) string {
	return name
	//s := strings.TrimSpace(strings.ToLower(name))
	//if s == "" {
	//	return "employee"
	//}
	//var b strings.Builder
	//for _, r := range s {
	//	if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
	//		b.WriteRune(r)
	//		continue
	//	}
	//	if r == '-' || r == '_' || r == ' ' {
	//		// 统一折叠为空格为连字符
	//		b.WriteRune('-')
	//	}
	//}
	//id := strings.Trim(b.String(), "-")
	//if id == "" {
	//	id = "employee"
	//}
	//if len(id) > 64 {
	//	id = id[:64]
	//}
	//return id
}
