package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/openocta/openocta/embed"
	"github.com/openocta/openocta/pkg/paths"
)

// EnvGetter returns an environment variable value by name.
type EnvGetter func(string) string

// DefaultEnv uses os.Getenv.
func DefaultEnv(key string) string {
	return os.Getenv(key)
}

// Load reads and parses the config file.
// Loads .env from the current working directory first so declared env vars are visible.
// If the config file does not exist, initializes from embedded openocta.json.example
// (writes it to the config path so the user can edit it).
// Returns default config if file is empty.
func Load(env EnvGetter) (*OpenOctaConfig, error) {
	_ = LoadEnvFromCurrentDir() // best-effort: .env from cwd
	if env == nil {
		env = DefaultEnv
	}
	stateDir := paths.ResolveStateDir(env)
	configPath := paths.ResolveConfigPath(env, stateDir)
	canonicalPath := paths.ResolveCanonicalConfigPath(env, stateDir)
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize from embedded openocta.json.example
			example, eErr := embed.ConfigExampleJSON()
			if eErr == nil && len(example) > 0 {
				if mkErr := os.MkdirAll(stateDir, 0700); mkErr == nil {
					var ex OpenOctaConfig
					if json.Unmarshal(example, &ex) == nil {
						if normalizeDesktopGatewayAuth(&ex, env) {
							if b, mErr := json.MarshalIndent(&ex, "", "  "); mErr == nil {
								example = b
							}
						}
					}
					_ = os.WriteFile(canonicalPath, example, 0600)
				}
				data = example
			} else {
				return &OpenOctaConfig{}, nil
			}
		} else {
			return nil, err
		}
	}
	var cfg OpenOctaConfig
	if len(data) == 0 {
		return &cfg, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if normalizeDesktopGatewayAuth(&cfg, env) {
		if out, mErr := json.MarshalIndent(&cfg, "", "  "); mErr == nil {
			_ = os.WriteFile(configPath, out, 0600)
		}
	}
	return &cfg, nil
}

// DefaultGatewayToken is the historical default gateway.auth.token.
const DefaultGatewayToken = "edc146993b5ae0b1544c3137cc888f94436cf11e1952cff6"

// generateRandomToken generates a secure 24-byte random hex token.
func generateRandomToken() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "fallback" + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// normalizeDesktopGatewayAuth normalizes gateway token in desktop run mode.
// If it is empty or is the legacy DefaultGatewayToken, generates a new random token.
func normalizeDesktopGatewayAuth(cfg *OpenOctaConfig, env EnvGetter) bool {
	if env == nil {
		env = DefaultEnv
	}
	var modePtr *string
	if cfg != nil && cfg.Gateway != nil {
		modePtr = cfg.Gateway.Mode
	}
	if paths.ResolveRunMode(env, modePtr) != "desktop" {
		return false
	}
	if cfg == nil {
		return false
	}
	if cfg.Gateway == nil {
		cfg.Gateway = &GatewayConfig{}
	}
	if cfg.Gateway.Auth == nil {
		cfg.Gateway.Auth = &GatewayAuthConfig{}
	}
	if cfg.Gateway.Auth.Mode != nil {
		if strings.EqualFold(strings.TrimSpace(*cfg.Gateway.Auth.Mode), "password") {
			return false
		}
	}
	changed := false
	curToken := strings.TrimSpace(cfg.Gateway.Auth.Token)
	if curToken == "" || curToken == DefaultGatewayToken {
		cfg.Gateway.Auth.Token = generateRandomToken()
		changed = true
	}
	if cfg.Gateway.Auth.Mode == nil || strings.TrimSpace(*cfg.Gateway.Auth.Mode) == "" {
		m := "token"
		cfg.Gateway.Auth.Mode = &m
		changed = true
	}
	return changed
}

// EnsureDefaultConfig ensures ~/.openocta/openocta.json exists; if not, creates the dir and writes minimal default config with a random Gateway Token.
func EnsureDefaultConfig(env EnvGetter) error {
	if env == nil {
		env = DefaultEnv
	}
	stateDir := paths.ResolveStateDir(env)
	configPath := paths.ResolveCanonicalConfigPath(env, stateDir)
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return err
	}
	modeToken := "token"
	modeLocal := "local"
	bindLoopback := "loopback"
	modeOff := "off"
	port := 18900
	resetOnExit := false
	cfg := &OpenOctaConfig{
		Meta: &ConfigMeta{
			LastTouchedVersion: "2026.2.9",
			LastTouchedAt:      time.Now().UTC().Format(time.RFC3339Nano),
		},
		Gateway: &GatewayConfig{
			Port: &port,
			Mode: &modeLocal,
			Bind: &bindLoopback,
			Auth: &GatewayAuthConfig{
				Mode:  &modeToken,
				Token: generateRandomToken(),
			},
			Tailscale: &GatewayTailscaleConfig{
				Mode:        &modeOff,
				ResetOnExit: &resetOnExit,
			},
		},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0600)
}
