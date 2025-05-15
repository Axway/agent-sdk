package compliance

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/agent"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/util"
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
	for complianceName, result := range results.items {
		ri, err := cacheManager.GetComplianceRuntimeResultByName(complianceName)
		if err != nil {
			j.logger.WithError(err).WithField("complianceRuntimeResultName", complianceName).Warn("skipping compliance runtime result")
			continue
		}

		crr := &management.ComplianceRuntimeResult{}
		crr.FromInstance(ri)
		updateSpecResultsAndHash(crr, result)

		logger := j.logger.
			WithField("crrId", ri.GetMetadata().ID).
			WithField("crrName", crr.GetName()).
			WithField("riskScore", result.RiskScore)

		logger.Debug("updating compliance runtime result")
		_, err = agent.GetCentralClient().CreateOrUpdateResource(crr)
		if err != nil {
			logger.WithError(err).Error("failed to updated runtime compliance result")
		}
	}
}

func updateSpecResultsAndHash(crr *management.ComplianceRuntimeResult, result RuntimeResult) {
	defer func() {
		hashInt, _ := util.ComputeHash(crr.Spec)
		util.SetAgentDetailsKey(crr, definitions.AttrSpecHash, fmt.Sprintf("%v", hashInt))
	}()

	if len(crr.Spec.Results) == 0 {
		crr.Spec.Results = []interface{}{
			map[string]interface{}{
				"runtime": map[string]interface{}{
					"riskScore": result.RiskScore,
				},
			},
		}
		return
	}

	res, ok := crr.Spec.Results[0].(map[string]interface{})
	if !ok {
		return
	}

	runtime, ok := res["runtime"].(map[string]interface{})
	if !ok {
		res["runtime"] = map[string]interface{}{
			"riskScore": result.RiskScore,
		}
		return
	}
	runtime["riskScore"] = result.RiskScore
}
