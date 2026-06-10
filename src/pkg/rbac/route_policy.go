package rbac

import (
	"fmt"
	"strings"
)

// Permission constants for REST and WebSocket authorization (C3-6/C3-7/C3-8).
const (
	PermSessionRead  = "session:read"
	PermSessionWrite = "session:write"
)

// HTTPRouteSpec declares RBAC for a REST route pattern.
type HTTPRouteSpec struct {
	Method               string
	PathPrefix           string
	Permission           string
	AllowGatewayToken    bool
	AllowUnauthenticated bool
}

// HTTPRoutes is the canonical REST route permission registry.
var HTTPRoutes = []HTTPRouteSpec{
	{Method: "GET", PathPrefix: "/api/auth/setup-status", AllowUnauthenticated: true},
	{Method: "POST", PathPrefix: "/api/auth/setup", AllowUnauthenticated: true},
	{Method: "POST", PathPrefix: "/api/auth/login", AllowUnauthenticated: true},
	{Method: "POST", PathPrefix: "/api/auth/logout", AllowUnauthenticated: true},

	{Method: "GET", PathPrefix: "/api/auth/me", Permission: ""},
	{Method: "GET", PathPrefix: "/api/auth/sessions", Permission: ""},
	{Method: "POST", PathPrefix: "/api/auth/logout-all", Permission: ""},

	{Method: "GET", PathPrefix: "/api/rbac/users", Permission: "menu:config"},
	{Method: "POST", PathPrefix: "/api/rbac/users", Permission: "menu:config"},
	{Method: "DELETE", PathPrefix: "/api/rbac/users/", Permission: "menu:config"},
	{Method: "GET", PathPrefix: "/api/rbac/roles", Permission: "menu:config"},

	{Method: "GET", PathPrefix: "/api/ops/clusters", Permission: "", AllowGatewayToken: true},
	{Method: "POST", PathPrefix: "/api/ops/clusters", Permission: "menu:config", AllowGatewayToken: true},
	{Method: "POST", PathPrefix: "/api/ops/clusters/sync-cmdb", Permission: "menu:config", AllowGatewayToken: true},
	{Method: "GET", PathPrefix: "/api/ops/clusters/", Permission: "", AllowGatewayToken: true},
	{Method: "PATCH", PathPrefix: "/api/ops/clusters/", Permission: "menu:config", AllowGatewayToken: true},
	{Method: "DELETE", PathPrefix: "/api/ops/clusters/", Permission: "menu:config", AllowGatewayToken: true},
	{Method: "GET", PathPrefix: "/api/ops/dashboard/summary", Permission: "", AllowGatewayToken: true},
	{Method: "GET", PathPrefix: "/api/ops/scenarios", Permission: "", AllowGatewayToken: true},
	{Method: "GET", PathPrefix: "/api/ops/health/signals", Permission: "", AllowGatewayToken: true},
	{Method: "GET", PathPrefix: "/api/ops/health/snapshots", Permission: "", AllowGatewayToken: true},
	{Method: "GET", PathPrefix: "/api/ops/alerts/groups", Permission: "", AllowGatewayToken: true},
	{Method: "GET", PathPrefix: "/api/ops/alerts/groups/", Permission: "", AllowGatewayToken: true},
	{Method: "PATCH", PathPrefix: "/api/ops/alerts/groups/", Permission: "ops:ack", AllowGatewayToken: true},
	{Method: "GET", PathPrefix: "/api/ops/inspection/im-status", Permission: "", AllowGatewayToken: true},
	{Method: "GET", PathPrefix: "/api/ops/job-runs", Permission: "", AllowGatewayToken: true},
	{Method: "GET", PathPrefix: "/api/ops/job-runs/", Permission: "", AllowGatewayToken: true},

	{Method: "GET", PathPrefix: "/api/ops/bch/", Permission: "menu:hadoop", AllowGatewayToken: true},
	{Method: "POST", PathPrefix: "/api/ops/bch/", Permission: "menu:hadoop", AllowGatewayToken: true},

	{Method: "GET", PathPrefix: "/health", Permission: "", AllowGatewayToken: true},
	{Method: "GET", PathPrefix: "/api/health", Permission: "", AllowGatewayToken: true},

	{Method: "POST", PathPrefix: "/api/skills/upload", Permission: "menu:config", AllowGatewayToken: true},
	{Method: "POST", PathPrefix: "/api/employee-skills/upload", Permission: "menu:config", AllowGatewayToken: true},
	{Method: "DELETE", PathPrefix: "/api/employee-skills/delete", Permission: "menu:config", AllowGatewayToken: true},

	{Method: "GET", PathPrefix: "/api/config", Permission: "menu:config", AllowGatewayToken: true},
	{Method: "GET", PathPrefix: "/api/config/env", Permission: "menu:config", AllowGatewayToken: true},
	{Method: "POST", PathPrefix: "/api/config/patch", Permission: "menu:config", AllowGatewayToken: true},
	{Method: "PATCH", PathPrefix: "/api/config/patch", Permission: "menu:config", AllowGatewayToken: true},

	{Method: "POST", PathPrefix: "/api/desktop/", Permission: "menu:config", AllowGatewayToken: true},

	{Method: "GET", PathPrefix: "/api/v1/", Permission: "", AllowGatewayToken: true},
	{Method: "POST", PathPrefix: "/api/v1/install", Permission: "menu:config", AllowGatewayToken: true},
}

// MethodPermissions is the canonical WebSocket method permission registry.
var MethodPermissions = map[string]string{
	"config.get":    "menu:config",
	"config.env":    "menu:config",
	"config.set":    "menu:config",
	"config.apply":  "menu:config",
	"config.patch":  "menu:config",
	"config.schema": "menu:config",
	"mcp.servers.delete": "menu:config",

	"cron.list":   "menu:config",
	"cron.add":    "menu:config",
	"cron.remove": "menu:config",
	"cron.update": "menu:config",
	"cron.run":    "menu:config",

	"skills.install":  "menu:config",
	"skills.update":   "menu:config",
	"skills.delete":   "menu:config",
	"skills.saveFile": "menu:config",

	"sessions.create":           "session:write",
	"sessions.patch":            "session:write",
	"sessions.reset":            "session:write",
	"sessions.delete":           "session:write",
	"sessions.compact":          "session:write",
	"sessions.list":             "session:read",
	"sessions.preview":          "session:read",
	"sessions.ensure":           "session:read",
	"sessions.usage":            "session:read",
	"sessions.usage.timeseries": "session:read",
	"sessions.usage.logs":       "session:read",

	"chat.send":   PermToolExecute,
	"chat.inject": "menu:config",
	"files.read":  PermToolExecute,

	"agents.create":    "menu:config",
	"agents.update":    "menu:config",
	"agents.delete":    "menu:config",
	"agents.files.set": "menu:config",

	"employees.create": "menu:config",
	"employees.delete": "menu:config",

	"employee.tasks.create": "menu:config",
	"employee.tasks.update": "menu:config",
	"employee.tasks.delete": "menu:config",

	"update.run": "menu:config",

	"approvals.approve":          "ops:ack",
	"approvals.deny":             "ops:ack",
	"approvals.whitelistSession": "ops:ack",
	"exec.approvals.set":         "ops:ack",
	"exec.approvals.node.set":    "ops:ack",
	"exec.approval.resolve":      "ops:ack",

	"channels.logout": "menu:config",
	"voicewake.set":   "menu:config",

	"node.pair.approve":   "menu:config",
	"node.pair.reject":    "menu:config",
	"device.pair.approve": "menu:config",
	"device.pair.reject":  "menu:config",
	"device.token.rotate": "menu:config",
	"device.token.revoke": "menu:config",

	"swarm.workspaces.create":   "menu:config",
	"swarm.workspaces.delete":   "menu:config",
	"swarm.workspaces.abortAll": "menu:config",
	"swarm.members.add":         "menu:config",
	"swarm.members.remove":      "menu:config",
	"swarm.message.send":        "menu:config",
}

// LookupHTTPRoute returns the matching route spec for method/path.
func LookupHTTPRoute(method, path string) (HTTPRouteSpec, bool) {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = normalizeHTTPPath(path)
	var best HTTPRouteSpec
	found := false
	bestLen := -1
	for _, spec := range HTTPRoutes {
		if spec.Method != method {
			continue
		}
		prefix := normalizeHTTPPath(spec.PathPrefix)
		if path == prefix || strings.HasPrefix(path, prefix) {
			if len(prefix) > bestLen {
				best = spec
				bestLen = len(prefix)
				found = true
			}
		}
	}
	return best, found
}

func normalizeHTTPPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if path != "/" && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}
	return path
}

// RequiredMethodPermission returns the permission code for a WebSocket method.
func RequiredMethodPermission(method string) string {
	return MethodPermissions[strings.TrimSpace(method)]
}

// AuthorizeMethod checks whether session may invoke a WebSocket method.
// Nil session with legacy gateway token is allowed (transitional).
func AuthorizeMethod(session *UserSession, method string, legacyGatewayToken bool) error {
	perm := RequiredMethodPermission(method)
	if perm == "" {
		if session == nil && !legacyGatewayToken {
			return fmt.Errorf("unauthorized")
		}
		return nil
	}
	if session == nil {
		if legacyGatewayToken {
			return nil
		}
		return fmt.Errorf("forbidden: requires permission %s", perm)
	}
	if HasPermission(session, perm) {
		return nil
	}
	return fmt.Errorf("forbidden: requires permission %s", perm)
}

// AuthorizeHTTP checks REST access for a route.
func AuthorizeHTTP(session *UserSession, method, path string, legacyGatewayToken bool) error {
	spec, ok := LookupHTTPRoute(method, path)
	if !ok {
		if session == nil && !legacyGatewayToken {
			return fmt.Errorf("unauthorized")
		}
		return nil
	}
	if spec.AllowUnauthenticated {
		return nil
	}
	if spec.Permission == "" {
		if session == nil && !(spec.AllowGatewayToken && legacyGatewayToken) {
			return fmt.Errorf("unauthorized")
		}
		return nil
	}
	if session == nil {
		if spec.AllowGatewayToken && legacyGatewayToken {
			return nil
		}
		return fmt.Errorf("unauthorized")
	}
	if HasPermission(session, spec.Permission) {
		return nil
	}
	return fmt.Errorf("forbidden: requires permission %s", spec.Permission)
}
