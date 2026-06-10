package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openocta/openocta/pkg/db"
	"github.com/openocta/openocta/pkg/rbac"
)

func TestLoginLockoutReturns429(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("OPENOCTA_STATE_DIR", tempDir)
	t.Setenv("OPENOCTA_RUN_MODE", "service")
	t.Setenv("OPENOCTA_SKIP_CHANNELS", "1")
	t.Setenv("OPENOCTA_SKIP_CRON", "1")
	t.Setenv("OPENOCTA_LOGIN_MAX_ATTEMPTS", "2")
	t.Setenv("OPENOCTA_LOGIN_LOCKOUT_MINUTES", "15")

	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if err := rbac.InitDB(tempDir); err != nil {
		t.Fatalf("rbac InitDB: %v", err)
	}
	if _, err := rbac.SetupInitialAdmin("admin", "SecurePass1!"); err != nil {
		t.Fatalf("SetupInitialAdmin: %v", err)
	}

	srv := NewServer(":0", "test-1.0.0")
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	client := &http.Client{}
	login := func(password string) *http.Response {
		body, _ := json.Marshal(map[string]string{"username": "admin", "password": password})
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "203.0.113.50:1234"
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("login request: %v", err)
		}
		return resp
	}

	resp := login("wrong-pass")
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("first failure expected 401, got %d", resp.StatusCode)
	}

	resp = login("wrong-pass")
	resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("second failure expected 429 lockout, got %d", resp.StatusCode)
	}

	resp = login("SecurePass1!")
	resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("valid password should still be blocked while locked, got %d", resp.StatusCode)
	}
}

func TestLogoutAllRevokesOtherSessions(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("OPENOCTA_STATE_DIR", tempDir)
	t.Setenv("OPENOCTA_RUN_MODE", "service")
	t.Setenv("OPENOCTA_SKIP_CHANNELS", "1")
	t.Setenv("OPENOCTA_SKIP_CRON", "1")

	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if err := rbac.InitDB(tempDir); err != nil {
		t.Fatalf("rbac InitDB: %v", err)
	}
	if _, err := rbac.SetupInitialAdmin("admin", "SecurePass1!"); err != nil {
		t.Fatalf("SetupInitialAdmin: %v", err)
	}

	tokenA, err := rbac.AuthenticateUser("admin", "SecurePass1!")
	if err != nil {
		t.Fatalf("AuthenticateUser A: %v", err)
	}
	tokenB, err := rbac.AuthenticateUser("admin", "SecurePass1!")
	if err != nil {
		t.Fatalf("AuthenticateUser B: %v", err)
	}

	srv := NewServer(":0", "test-1.0.0")
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()
	client := &http.Client{}

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/auth/logout-all", nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("logout-all: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("logout-all expected 200, got %d", resp.StatusCode)
	}

	if _, err := rbac.ValidateToken(tokenA); err == nil {
		t.Fatal("expected tokenA revoked")
	}
	if _, err := rbac.ValidateToken(tokenB); err != nil {
		t.Fatalf("expected tokenB still valid: %v", err)
	}

	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/auth/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list sessions expected 200, got %d", resp.StatusCode)
	}
}
