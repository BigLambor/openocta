package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/openocta/openocta/pkg/db"
	"github.com/openocta/openocta/pkg/gateway/protocol"
	"github.com/openocta/openocta/pkg/rbac"
)

func TestHTTPAuthHardening_RejectsUnauthorized(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("OPENOCTA_STATE_DIR", tempDir)
	t.Setenv("OPENOCTA_RUN_MODE", "service")
	t.Setenv("OPENOCTA_SKIP_CHANNELS", "1")
	t.Setenv("OPENOCTA_SKIP_CRON", "1")
	t.Setenv("OPENOCTA_GATEWAY_TOKEN", "test-token-12345")

	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if err := rbac.InitDB(tempDir); err != nil {
		t.Fatalf("rbac InitDB: %v", err)
	}

	srv := NewServer(":0", "test-1.0.0")
	handler := srv.Handler()

	// Request /api/config without Token
	req := httptest.NewRequest("GET", "/api/config", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", w.Code)
	}

	// Request with invalid token
	req = httptest.NewRequest("GET", "/api/config", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", w.Code)
	}

	// Request with valid token
	req = httptest.NewRequest("GET", "/api/config", nil)
	req.Header.Set("Authorization", "Bearer test-token-12345")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Since we authenticated, status should not be 401 (might be 200 or 403/Forbidden depending on RBAC menu:config, but not 401)
	if w.Code == http.StatusUnauthorized {
		t.Errorf("Expected authorized token to bypass 401, got %d", w.Code)
	}
}

func TestPprofAuthHardening(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("OPENOCTA_STATE_DIR", tempDir)
	t.Setenv("OPENOCTA_SKIP_CHANNELS", "1")
	t.Setenv("OPENOCTA_SKIP_CRON", "1")

	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if err := rbac.InitDB(tempDir); err != nil {
		t.Fatalf("rbac InitDB: %v", err)
	}

	// 1. By default, OPENOCTA_ENABLE_PPROF is not set, so pprof routes are not registered -> 404
	srv1 := NewServer(":0", "test-1.0.0")
	req1 := httptest.NewRequest("GET", "/debug/pprof/", nil)
	req1.RemoteAddr = "127.0.0.1:1234"
	w1 := httptest.NewRecorder()
	srv1.Handler().ServeHTTP(w1, req1)
	if w1.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for disabled pprof, got %d", w1.Code)
	}

	// 2. Enable pprof via env var
	t.Setenv("OPENOCTA_ENABLE_PPROF", "1")
	srv2 := NewServer(":0", "test-1.0.0")
	handler2 := srv2.Handler()

	// Loopback request should succeed (since it falls through to pprof handler, which returns 200/html/etc)
	req2 := httptest.NewRequest("GET", "/debug/pprof/", nil)
	req2.RemoteAddr = "127.0.0.1:1234"
	w2 := httptest.NewRecorder()
	handler2.ServeHTTP(w2, req2)
	// Status could be 200 or 301 (redirects), but shouldn't be 403 or 404
	if w2.Code == http.StatusForbidden {
		t.Errorf("Loopback request to enabled pprof was forbidden: %d", w2.Code)
	}

	// Non-loopback request without admin session should be 403 Forbidden
	req3 := httptest.NewRequest("GET", "/debug/pprof/", nil)
	req3.RemoteAddr = "192.168.1.100:1234"
	w3 := httptest.NewRecorder()
	handler2.ServeHTTP(w3, req3)
	if w3.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden for non-loopback pprof request, got %d", w3.Code)
	}
}

func TestWebSocketOriginChecks(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("OPENOCTA_STATE_DIR", tempDir)
	t.Setenv("OPENOCTA_SKIP_CHANNELS", "1")
	t.Setenv("OPENOCTA_SKIP_CRON", "1")

	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if err := rbac.InitDB(tempDir); err != nil {
		t.Fatalf("rbac InitDB: %v", err)
	}

	// Test case 1: Desktop mode WebSocket origin checks
	t.Setenv("OPENOCTA_RUN_MODE", "desktop")
	srv := NewServer(":0", "test-1.0.0")

	// Allow localhost (should pass origin check, but fail upgrading with 500 because ResponseRecorder doesn't implement http.Hijacker)
	req1 := httptest.NewRequest("GET", "/ws", nil)
	req1.Header.Set("Upgrade", "websocket")
	req1.Header.Set("Connection", "upgrade")
	req1.Header.Set("Sec-WebSocket-Key", "x3JJHMbDL1EzLkh9GBhXDw==")
	req1.Header.Set("Sec-WebSocket-Version", "13")
	req1.Header.Set("Origin", "http://localhost:5173")
	w1 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w1, req1)
	if w1.Code == http.StatusForbidden {
		t.Errorf("Expected localhost origin to be allowed, but got 403 Forbidden")
	}

	// Reject external domain
	req2 := httptest.NewRequest("GET", "/ws", nil)
	req2.Header.Set("Upgrade", "websocket")
	req2.Header.Set("Connection", "upgrade")
	req2.Header.Set("Sec-WebSocket-Key", "x3JJHMbDL1EzLkh9GBhXDw==")
	req2.Header.Set("Sec-WebSocket-Version", "13")
	req2.Header.Set("Origin", "http://malicious-attacker.com")
	w2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Errorf("Expected malicious origin to be rejected with 403 Forbidden, got %d", w2.Code)
	}

	// Allow wails origin
	req3 := httptest.NewRequest("GET", "/ws", nil)
	req3.Header.Set("Upgrade", "websocket")
	req3.Header.Set("Connection", "upgrade")
	req3.Header.Set("Sec-WebSocket-Key", "x3JJHMbDL1EzLkh9GBhXDw==")
	req3.Header.Set("Sec-WebSocket-Version", "13")
	req3.Header.Set("Origin", "wails://localhost")
	w3 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w3, req3)
	if w3.Code == http.StatusForbidden {
		t.Errorf("Expected wails origin to be allowed, but got 403 Forbidden")
	}
}

func TestWebSocketMethodPermissions(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("OPENOCTA_STATE_DIR", tempDir)
	t.Setenv("OPENOCTA_RUN_MODE", "service")
	t.Setenv("OPENOCTA_SKIP_CHANNELS", "1")
	t.Setenv("OPENOCTA_SKIP_CRON", "1")
	t.Setenv("OPENOCTA_GATEWAY_TOKEN", "gateway-token-secret")

	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if err := rbac.InitDB(tempDir); err != nil {
		t.Fatalf("rbac InitDB: %v", err)
	}

	srv := NewServer(":0", "test-1.0.0")
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	if _, err := rbac.SetupInitialAdmin("admin", "admin888!"); err != nil {
		t.Fatalf("SetupInitialAdmin: %v", err)
	}

	// 1. Create a Viewer user (role ID = 5, has no menu:config or ops:ack permissions)
	err := rbac.CreateUser("test_viewer", "password123", 5)
	if err != nil {
		t.Fatalf("Failed to create test_viewer: %v", err)
	}

	viewerToken, err := rbac.AuthenticateUser("test_viewer", "password123")
	if err != nil {
		t.Fatalf("Failed to authenticate test_viewer: %v", err)
	}

	adminToken, err := rbac.AuthenticateUser("admin", "admin888!")
	if err != nil {
		t.Fatalf("Failed to authenticate admin: %v", err)
	}

	wsURL := "ws" + ts.URL[4:] + "/ws"

	// Helper function to connect, handshake, call method, and return response
	callWSMethod := func(token string, method string, params interface{}) (bool, *protocol.ErrorShape) {
		dialer := websocket.Dialer{}
		conn, _, err := dialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Dial error: %v", err)
		}
		defer conn.Close()

		// Send connect request
		connectReq := protocol.RequestFrame{
			Type:   "req",
			ID:     "conn-1",
			Method: "connect",
			Params: protocol.ConnectParams{
				MinProtocol: 3,
				MaxProtocol: 3,
				Client: protocol.ConnectClientInfo{
					ID:       "test-client-1",
					Version:  "1.0.0",
					Platform: "mac",
					Mode:     "desktop",
				},
				Auth: &protocol.ConnectAuth{
					Token: token,
				},
			},
		}
		if err := conn.WriteJSON(connectReq); err != nil {
			t.Fatalf("WriteJSON connect: %v", err)
		}

		var resp protocol.ResponseFrame
		if err := conn.ReadJSON(&resp); err != nil {
			t.Fatalf("ReadJSON connect: %v", err)
		}
		if !resp.OK {
			return false, resp.Error
		}

		// Send method request
		methodReq := protocol.RequestFrame{
			Type:   "req",
			ID:     "method-2",
			Method: method,
			Params: params,
		}
		if err := conn.WriteJSON(methodReq); err != nil {
			t.Fatalf("WriteJSON method: %v", err)
		}

		if err := conn.ReadJSON(&resp); err != nil {
			t.Fatalf("ReadJSON method: %v", err)
		}

		return resp.OK, resp.Error
	}

	// Test case 2.1: Viewer calling "config.get" (requires "menu:config") should be rejected with forbidden
	ok, errShape := callWSMethod(viewerToken, "config.get", nil)
	if ok {
		t.Errorf("Expected config.get to be rejected for viewer, but it succeeded")
	} else if errShape == nil || errShape.Code != "forbidden" {
		t.Errorf("Expected forbidden error code, got: %+v", errShape)
	}

	// Test case 2.2: Viewer calling "health" (no permission required) should succeed
	ok, errShape = callWSMethod(viewerToken, "health", nil)
	if !ok {
		t.Errorf("Expected health to succeed for viewer, got error: %+v", errShape)
	}

	// Test case 2.3: Admin calling "config.get" (requires "menu:config") should succeed
	ok, errShape = callWSMethod(adminToken, "config.get", nil)
	if !ok {
		t.Errorf("Expected config.get to succeed for admin, got error: %+v", errShape)
	}

	// Test case 2.4: Legacy gateway token client calling "config.get" should succeed (bypass permission check)
	ok, errShape = callWSMethod("gateway-token-secret", "config.get", nil)
	if !ok {
		t.Errorf("Expected config.get to succeed for legacy gateway token, got error: %+v", errShape)
	}

	// Test case 2.5: Viewer calling chat.send (requires tool:execute) should be rejected
	ok, errShape = callWSMethod(viewerToken, "chat.send", map[string]interface{}{
		"sessionKey": "main",
		"message":    "hello",
	})
	if ok {
		t.Errorf("Expected chat.send to be rejected for viewer, but it succeeded")
	} else if errShape == nil || errShape.Code != "forbidden" {
		t.Errorf("Expected forbidden for viewer chat.send, got: %+v", errShape)
	}

	// Test case 2.6: Admin calling chat.send should pass RBAC gate (may fail later without model config)
	ok, errShape = callWSMethod(adminToken, "chat.send", map[string]interface{}{
		"sessionKey": "main",
		"message":    "ping",
	})
	if !ok && errShape != nil && errShape.Code == "forbidden" {
		t.Errorf("Expected admin chat.send not forbidden, got: %+v", errShape)
	}
}
