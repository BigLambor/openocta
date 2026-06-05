package automation

import (
	"github.com/openocta/openocta/pkg/cron"
)

// AutomationScheduler wraps the cron service to manage scheduled tasks.
type AutomationScheduler struct {
	CronSvc *cron.Service
}

// NewAutomationScheduler creates a new AutomationScheduler wrapper.
func NewAutomationScheduler(cronSvc *cron.Service) *AutomationScheduler {
	return &AutomationScheduler{
		CronSvc: cronSvc,
	}
}

// InitializeDefaultJobs configures and links default inspection tasks.
func (s *AutomationScheduler) InitializeDefaultJobs() error {
	if s.CronSvc == nil {
		return nil
	}
	// Currently delegated to cronSvc.ensureDefaultJobs via s.CronSvc.Start()
	return nil
}
