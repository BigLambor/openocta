package rbac

import (
	"database/sql"
	"fmt"
)

type roleRepository struct {
	db *sql.DB
}

func newRoleRepository(db *sql.DB) RoleRepository {
	if db == nil {
		return nil
	}
	return &roleRepository{db: db}
}

func (r *roleRepository) List() ([]RoleResponse, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("role repository 未初始化")
	}
	rows, err := r.db.Query(`SELECT id, name, description FROM roles`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := []RoleResponse{}
	for rows.Next() {
		var role RoleResponse
		if err := rows.Scan(&role.ID, &role.Name, &role.Description); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (r *roleRepository) Permissions(roleID int) ([]string, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("role repository 未初始化")
	}
	rows, err := r.db.Query(`
		SELECT permission_code
		FROM role_permissions
		WHERE role_id = ?
	`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	permissions := []string{}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		permissions = append(permissions, code)
	}
	return permissions, rows.Err()
}

func (r *roleRepository) EnsureRole(id int, name, description string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("role repository 未初始化")
	}
	_, err := r.db.Exec(`INSERT OR IGNORE INTO roles (id, name, description) VALUES (?, ?, ?)`, id, name, description)
	return err
}

func (r *roleRepository) EnsurePermission(code, name, ptype string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("role repository 未初始化")
	}
	_, err := r.db.Exec(`INSERT OR IGNORE INTO permissions (code, name, type) VALUES (?, ?, ?)`, code, name, ptype)
	return err
}

func (r *roleRepository) EnsureRolePermission(roleID int, permissionCode string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("role repository 未初始化")
	}
	_, err := r.db.Exec(`INSERT OR IGNORE INTO role_permissions (role_id, permission_code) VALUES (?, ?)`, roleID, permissionCode)
	return err
}

func (r *roleRepository) SeedDefaults() error {
	if r == nil || r.db == nil {
		return fmt.Errorf("role repository 未初始化")
	}

	roles := []struct {
		id   int
		name string
		desc string
	}{
		{1, "admin", "超级管理员 - 拥有全量管理与各运维域执行权限"},
		{2, "hadoop_operator", "Hadoop生态运维员 - 具备Hadoop域的巡检与交互权限"},
		{3, "fi_operator", "FI商业版运维员 - 具备FI域的巡检与交互权限"},
		{4, "gbase_operator", "GBase数据库运维员 - 具备GBase域的巡检与交互权限"},
		{5, "viewer", "只读访客 - 仅有大盘和系统巡检查看权限，无法交互与手动巡检"},
	}
	for _, role := range roles {
		if err := r.EnsureRole(role.id, role.name, role.desc); err != nil {
			return err
		}
	}

	permissions := []struct {
		code  string
		name  string
		ptype string
	}{
		{"menu:overview", "主导航: 运维大屏", "menu"},
		{"menu:hadoop", "主导航: Hadoop生态", "menu"},
		{"menu:fi", "主导航: FI商业生态", "menu"},
		{"menu:gbase", "主导航: GBase数据库", "menu"},
		{"menu:governance", "主导航: 开发治理平台", "menu"},
		{"menu:dataapps", "主导航: 数据App运维", "menu"},
		{"menu:config", "主导航: 系统设置", "menu"},
		{"ops:inspect", "操作: 执行深度巡检", "ops"},
		{"ops:diagnose", "操作: 发起智能诊断", "ops"},
		{"ops:ack", "操作: 确认处理告警组", "ops"},
		{"ops:wework_conf", "操作: 企业微信通道配置", "ops"},
		{PermToolExecute, "Tool: 执行 Agent 工具", "tool"},
		{PermToolExecuteShell, "Tool: 执行 Shell/命令", "tool"},
		{PermToolExecuteWrite, "Tool: 写入/编辑文件", "tool"},
		{PermToolExecuteMCP, "Tool: 执行 MCP/扩展工具", "tool"},
	}
	for _, perm := range permissions {
		if err := r.EnsurePermission(perm.code, perm.name, perm.ptype); err != nil {
			return err
		}
	}
	for _, perm := range permissions {
		if err := r.EnsureRolePermission(1, perm.code); err != nil {
			return err
		}
	}
	for _, code := range []string{"ops:ack"} {
		if err := r.EnsureRolePermission(1, code); err != nil {
			return err
		}
	}
	for _, code := range []string{"menu:overview", "menu:gbase", "ops:diagnose", "ops:inspect", "ops:ack", PermToolExecute, PermToolExecuteShell, PermToolExecuteWrite, PermToolExecuteMCP} {
		if err := r.EnsureRolePermission(4, code); err != nil {
			return err
		}
	}
	for _, code := range []string{PermToolExecute, PermToolExecuteShell, PermToolExecuteWrite, PermToolExecuteMCP} {
		for _, roleID := range []int{2, 3} {
			if err := r.EnsureRolePermission(roleID, code); err != nil {
				return err
			}
		}
	}
	for _, code := range []string{"menu:overview", "menu:hadoop", "menu:fi", "menu:gbase", "menu:governance", "menu:dataapps"} {
		if err := r.EnsureRolePermission(5, code); err != nil {
			return err
		}
	}
	return nil
}
