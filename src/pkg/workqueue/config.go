package workqueue

import "github.com/openocta/openocta/pkg/config"

const (
	defaultMaxConcurrentL2Runs = 8
	defaultMaxL2PerParentRun   = 50
	defaultParentRunTimeoutMs  = 30 * 60 * 1000
	defaultTaskLeaseMs         = 2 * 60 * 1000
	defaultPollIntervalMs      = 5 * 1000
	defaultL2RunTimeoutMs      = 10 * 60 * 1000
	defaultL2CooldownMs        = 60 * 60 * 1000
)

// RuntimeConfig holds effective queue settings.
type RuntimeConfig struct {
	MaxConcurrentL2Runs  int
	MaxL2PerParentRun    int
	DefaultL2CooldownMs  int64
	DomainReduceEnabled  bool
	DomainReduceUseLLM   bool
	ParentRunTimeoutMs   int64
	TaskLeaseMs          int64
	PollIntervalMs       int64
	L2RunTimeoutMs       int64
}

// ConfigFromOpenOcta resolves queue settings from app config.
func ConfigFromOpenOcta(cfg *config.OpenOctaConfig) RuntimeConfig {
	out := RuntimeConfig{
		MaxConcurrentL2Runs: defaultMaxConcurrentL2Runs,
		MaxL2PerParentRun:   defaultMaxL2PerParentRun,
		DefaultL2CooldownMs: defaultL2CooldownMs,
		ParentRunTimeoutMs:  defaultParentRunTimeoutMs,
		TaskLeaseMs:         defaultTaskLeaseMs,
		PollIntervalMs:      defaultPollIntervalMs,
		L2RunTimeoutMs:      defaultL2RunTimeoutMs,
	}
	if cfg == nil || cfg.Cron == nil {
		return out
	}
	c := cfg.Cron
	if v := intPtr(c.MaxConcurrentL2Runs); v > 0 {
		out.MaxConcurrentL2Runs = v
	} else if v := intPtr(c.MaxConcurrentRuns); v > 0 {
		out.MaxConcurrentL2Runs = v
	}
	if v := intPtr(c.MaxL2PerParentRun); v > 0 {
		out.MaxL2PerParentRun = v
	}
	if v := intPtr(c.ParentRunTimeoutMs); v > 0 {
		out.ParentRunTimeoutMs = int64(v)
	}
	if v := intPtr(c.DefaultL2CooldownMs); v > 0 {
		out.DefaultL2CooldownMs = int64(v)
	}
	if c.DomainReduceEnabled != nil {
		out.DomainReduceEnabled = *c.DomainReduceEnabled
	}
	if c.DomainReduceUseLLM != nil {
		out.DomainReduceUseLLM = *c.DomainReduceUseLLM
	}
	return out
}

func intPtr(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}
