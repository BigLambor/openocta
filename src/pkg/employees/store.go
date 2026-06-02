package employees

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/openocta/openocta/pkg/paths"
)

// ListSummaries 返回所有数字员工的概要信息。
// env 通常为 os.Getenv。
func ListSummaries(env func(string) string) ([]Summary, error) {
	var out []Summary

	root := ResolveEmployeesDir(env)
	if fi, err := os.Stat(root); err == nil && fi.IsDir() {
		entries, err := os.ReadDir(root)
		if err == nil {
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				id := e.Name()
				manifestPath := filepath.Join(root, id, "manifest.json")
				data, err := os.ReadFile(manifestPath)
				if err != nil {
					continue
				}
				var m Manifest
				if err := json.Unmarshal(data, &m); err != nil {
					continue
				}
				if m.ID == "" {
					m.ID = id
				}
				skillNames := listEmployeeSkillNames(env, m.ID, m.SkillIDs)
				mcpKeys := make([]string, 0, len(m.McpServers))
				for k := range m.McpServers {
					if k != "" {
						mcpKeys = append(mcpKeys, k)
					}
				}
				sort.Strings(mcpKeys)
				typeVal := strings.TrimSpace(m.Type)
				if typeVal == "" {
					typeVal = "其它"
				}
				s := Summary{
					ID:               m.ID,
					Name:             coalesce(m.Name, m.ID),
					Description:      m.Description,
					Prompt:           m.Prompt,
					Enabled:          m.Enabled,
					CreatedAt:        m.CreatedAt,
					Builtin:          m.Builtin,
					SkillIDs:         append([]string(nil), m.SkillIDs...),
					SkillNames:       skillNames,
					McpServerKeys:    mcpKeys,
					Type:             typeVal,
					From:             m.From,
					DomainKeys:       append([]string(nil), m.DomainKeys...),
					CapabilityKeys:   append([]string(nil), m.CapabilityKeys...),
					RoleType:         m.RoleType,
					Responsibilities: append([]string(nil), m.Responsibilities...),
					InputSources:     append([]string(nil), m.InputSources...),
					OutputTypes:      append([]string(nil), m.OutputTypes...),
					ActionScopes:     append([]string(nil), m.ActionScopes...),
					PermissionKeys:   append([]string(nil), m.PermissionKeys...),
					RunbookRefs:      append([]string(nil), m.RunbookRefs...),
					KnowledgeRefs:    append([]string(nil), m.KnowledgeRefs...),
				}
				out = append(out, s)
			}
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// LoadManifest 从 ~/.openocta/employees/<id>/manifest.json 加载指定 ID 的数字员工 manifest。
func LoadManifest(id string, env func(string) string) (*Manifest, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, os.ErrNotExist
	}
	root := ResolveEmployeesDir(env)
	manifestPath := filepath.Join(root, id, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, os.ErrNotExist
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, os.ErrNotExist
	}
	if m.ID == "" {
		m.ID = id
	}
	return &m, nil
}

// SaveManifest 在 ~/.openocta/employees/<id>/ 下写入 manifest.json。
func SaveManifest(m *Manifest, env func(string) string) error {
	if m == nil {
		return nil
	}
	id := strings.TrimSpace(m.ID)
	if id == "" {
		id = strings.TrimSpace(m.Name)
	}
	if id == "" {
		return os.ErrInvalid
	}

	root := ResolveEmployeesDir(env)
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// 若已存在 manifest，则保留 CreatedAt 等历史字段（除非调用方显式覆盖）。
	// 注意：Enabled 由调用方显式传入时不再被覆盖，以支持禁用操作。
	existingPath := filepath.Join(dir, "manifest.json")
	if data, err := os.ReadFile(existingPath); err == nil {
		var old Manifest
		if jsonErr := json.Unmarshal(data, &old); jsonErr == nil {
			if m.CreatedAt == 0 && old.CreatedAt != 0 {
				m.CreatedAt = old.CreatedAt
			}
		}
	}
	if m.CreatedAt == 0 {
		m.CreatedAt = time.Now().UnixMilli()
	}

	m.ID = id
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0o644)
}

// DeleteEmployee 删除指定 ID 的用户数字员工（不会影响内置）。
func DeleteEmployee(id string, env func(string) string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return os.ErrInvalid
	}
	root := ResolveEmployeesDir(env)
	dir := filepath.Join(root, id)
	// 内置员工没有磁盘目录，这里仅删除用户目录。
	if _, err := os.Stat(dir); err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

func coalesce(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// listEmployeeSkillNames 返回该员工的技能名称列表：manifest.skillIds + employee_skills/<id> 下的目录名。
func listEmployeeSkillNames(env func(string) string, employeeID string, manifestSkillIDs []string) []string {
	seen := make(map[string]struct{})
	for _, id := range manifestSkillIDs {
		if id = strings.TrimSpace(id); id != "" {
			seen[id] = struct{}{}
		}
	}
	stateDir := paths.ResolveStateDir(env)
	skillsDir := filepath.Join(stateDir, "employee_skills", employeeID)
	if entries, err := os.ReadDir(skillsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				name := strings.TrimSpace(e.Name())
				if name != "" {
					seen[name] = struct{}{}
				}
			}
		}
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
