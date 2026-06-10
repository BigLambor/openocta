package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/openocta/openocta/pkg/audit"
	"github.com/openocta/openocta/pkg/rbac"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type SetupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	RoleID   int    `json:"roleId"`
}

func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	needs, err := rbac.NeedsSetup()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"检查初始化状态失败"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]bool{"needsSetup": needs})
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"无效的JSON格式"}`))
		return
	}

	token, err := rbac.SetupInitialAdmin(req.Username, req.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "已完成初始化") {
			status = http.StatusConflict
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"error":"` + err.Error() + `"}`))
		return
	}

	session, err := rbac.ValidateToken(token)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"建立登录会话失败"}`))
		return
	}
	writeSessionResponse(w, r, token, session)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	needs, err := rbac.NeedsSetup()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"检查初始化状态失败"}`))
		return
	}
	if needs {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"系统尚未初始化，请先创建管理员账号","code":"setup_required"}`))
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"无效的JSON格式"}`))
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)

	if req.Username == "" || req.Password == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"用户名或密码不能为空"}`))
		return
	}

	ip := clientIP(r)
	lockStatus, err := rbac.CheckLoginAllowed(ip, req.Username)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"检查登录限制失败"}`))
		return
	}
	if !lockStatus.Allowed {
		writeLoginLocked(w, lockStatus)
		return
	}

	token, err := rbac.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		if lockedStatus, lockErr := rbac.RecordLoginFailure(ip, req.Username); lockErr == nil && !lockedStatus.Allowed {
			writeLoginLocked(w, lockedStatus)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"` + err.Error() + `"}`))
		return
	}
	_ = rbac.RecordLoginSuccess(ip, req.Username)

	session, err := rbac.ValidateToken(token)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"建立登录会话失败"}`))
		return
	}

	writeSessionResponse(w, r, token, session)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := readRBACSessionToken(r)
	if token != "" {
		if session, err := rbac.ValidateToken(token); err == nil {
			_ = audit.Record(audit.Entry{
				ActorID:    session.Username,
				Action:     "auth.logout",
				ObjectType: "user",
				ObjectID:   session.Username,
				Summary:    "登出当前会话",
				Metadata: map[string]interface{}{
					"tokenHint": rbacTokenHint(token),
				},
			})
		}
		_ = rbac.InvalidateToken(token)
	}
	clearSessionCookie(w, r)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func (s *Server) handleLogoutAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session := GetUserSession(r)
	if session == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"未登录"}`))
		return
	}
	currentToken := readRBACSessionToken(r)
	includeCurrent := strings.EqualFold(r.URL.Query().Get("includeCurrent"), "true")
	exceptToken := currentToken
	if includeCurrent {
		exceptToken = ""
	}
	removed, err := rbac.InvalidateAllSessions(session.UserID, exceptToken)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"吊销会话失败"}`))
		return
	}
	if includeCurrent {
		clearSessionCookie(w, r)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":      true,
		"removed": removed,
	})
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session := GetUserSession(r)
	if session == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"未登录"}`))
		return
	}
	sessions, err := rbac.ListUserSessions(session.UserID, readRBACSessionToken(r))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"获取会话列表失败"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"sessions": sessions,
	})
}

func rbacTokenHint(token string) string {
	if len(token) <= 8 {
		return token
	}
	return token[len(token)-8:]
}

func (s *Server) handleGetMe(w http.ResponseWriter, r *http.Request) {
	session := GetUserSession(r)
	if session == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"未登录"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(session)
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := rbac.ListUsers()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"获取用户列表失败: ` + err.Error() + `"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(users)
}

func (s *Server) handleListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := rbac.ListRoles()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"获取角色列表失败: ` + err.Error() + `"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(roles)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"无效的JSON格式"}`))
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)

	if req.Username == "" || req.Password == "" || req.RoleID <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"参数不完整"}`))
		return
	}

	err := rbac.CreateUser(req.Username, req.Password, req.RoleID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"创建用户失败: ` + err.Error() + `"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"无效的用户ID"}`))
		return
	}

	err = rbac.DeleteUser(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"删除用户失败: ` + err.Error() + `"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}
