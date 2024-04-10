package compliance

import (
	"time"

	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type Manager interface {
	RegisterRuntimeComplianceJob(interval time.Duration, processor Processor)
	Trigger()
}

type complianceManager struct {
	logger log.FieldLogger
	job    *runtimeComplianceJob
}

var manager Manager

func GetManager() Manager {
	if manager == nil {
		cm := &complianceManager{
			logger: log.NewFieldLogger().WithComponent("compliance"),
		}
		manager = cm
	}
	return manager
}

func (m *complianceManager) Trigger() {
	if m.job != nil {
		m.job.Execute()
	}
}

func (m *complianceManager) RegisterRuntimeComplianceJob(interval time.Duration, processor Processor) {
	if m.job != nil {
		jobs.UnregisterJob(m.job.id)
	}

	job := &runtimeComplianceJob{
		logger:    m.logger,
		processor: processor,
	}
	id, err := jobs.RegisterIntervalJobWithName(job, interval, "Runtime compliance")
	if err != nil {
		m.logger.WithError(err).Error("failed to register runtime compliance job")
	}
	job.id = id
	m.job = job
}
