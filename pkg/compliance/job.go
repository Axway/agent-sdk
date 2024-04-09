package compliance

import (
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	SourceCompliancePath = "/source/compliance"
)

type Processor interface {
	CollectRuntimeResult(RuntimeResults) error
}

type runtimeComplianceJob struct {
	logger    log.FieldLogger
	id        string
	processor Processor
}

func (j *runtimeComplianceJob) Status() error {
	return nil
}

func (j *runtimeComplianceJob) Ready() bool {
	return true
}

func (j *runtimeComplianceJob) Execute() error {
	if j.processor != nil {
		results := &runtimeResults{
			logger: j.logger,
		}
		j.logger.Info("starting runtime compliance processing")
		j.processor.CollectRuntimeResult(results)
		j.publishResources(results)
		j.logger.Info("completed runtime compliance processing")
	}
	return nil
}

func (j *runtimeComplianceJob) publishResources(results *runtimeResults) {
	cacheManager := agent.GetCacheManager()
	for instanceName, result := range results.items {
		ri, err := cacheManager.GetAPIServiceInstanceByName(instanceName)
		if err != nil {
			j.logger.WithError(err).WithField("instanceName", instanceName).Warn("skipping instance")
			continue
		}

		instance := &management.APIServiceInstance{}
		instance.FromInstance(ri)
		if instance.Source != nil {
			compliance := management.ApiServiceInstanceSourceCompliance{
				Runtime: management.ApiServiceInstanceSourceRuntimeStatus{
					Result: management.ApiServiceInstanceSourceRuntimeStatusResult{
						Timestamp: v1.Time(time.Now()),
						RiskScore: result.RiskScore,
					},
				},
			}

			patches := make([]map[string]interface{}, 0)
			patches = append(patches, map[string]interface{}{
				apic.PatchOperation: apic.PatchOpAdd,
				apic.PatchPath:      SourceCompliancePath,
				apic.PatchValue:     compliance,
			})

			logger := j.logger.
				WithField("instanceId", ri.Metadata.ID).
				WithField("instanceName", instanceName).
				WithField("riskScore", result.RiskScore)

			logger.Debug("updating runtime compliance result")
			_, err := agent.GetCentralClient().PatchSubResource(instance, management.ApiServiceInstanceSourceSubResourceName, patches)
			if err != nil {
				logger.WithError(err).Error("failed to updated runtime compliance result")
			}
		}
	}
}
