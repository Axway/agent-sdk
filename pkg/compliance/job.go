package compliance

import (
	"fmt"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
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
		crr := &management.ComplianceRuntimeResult{}
		ri, err := cacheManager.GetComplianceRuntimeResultByName(complianceName)
		if err != nil {
			j.logger.WithError(err).WithField("complianceRuntimeResultName", complianceName).Debug("compliance runtime result not existing")
			crr = management.NewComplianceRuntimeResult(complianceName, result.ComplianceScopedEnv)
		} else {
			crr.FromInstance(ri)
		}

		updateSpec(crr, result)
		logger := j.logger.
			WithField("crrName", crr.GetName()).
			WithField("riskScore", result.RiskScore)

		logger.Debug("creating/updating compliance runtime result")
		_, err = agent.GetCentralClient().CreateOrUpdateResource(crr)
		if err != nil {
			logger.WithError(err).Error("failed to create/update runtime compliance result")
			continue
		}
		crrName := fmt.Sprintf("%s/%s", result.ComplianceScopedEnv, crr.Name)
		linkComplianceSubresource(logger, result, crrName)
	}
}

func updateSpec(crr *management.ComplianceRuntimeResult, result RuntimeResult) {
	defer func() {
		hashInt, _ := util.ComputeHash(crr.Spec)
		if crr.Spec.ComplianceAgent == "" {
			crr.Spec.ComplianceAgent = result.ComplianceAgentName
		}
		if time.Now().Add(-6 * time.Hour).After(time.Time(crr.Spec.Timestamp)) {
			crr.Spec.Timestamp = v1.Time(time.Now())
		}
		if crr.Spec.Type == "" {
			crr.Spec.Type = result.ComplianceAgentType
		}
		util.SetAgentDetailsKey(crr, definitions.AttrSpecHash, fmt.Sprintf("%v", hashInt))
	}()

	if len(crr.Spec.Results) == 0 {
		crr.Spec.Results = []interface{}{
			map[string]interface{}{
				"runtime": map[string]interface{}{
					"riskScore": result.RiskScore,
				},
				"type": result.ComplianceAgentType,
			},
		}
		return
	}

	res, ok := crr.Spec.Results[0].(map[string]interface{})
	if !ok {
		return
	}
	defer func() { res["type"] = result.ComplianceAgentType }()

	runtime, ok := res["runtime"].(map[string]interface{})
	if !ok {
		res["runtime"] = map[string]interface{}{
			"riskScore": result.RiskScore,
		}
		return
	}
	runtime["riskScore"] = result.RiskScore
}

func linkComplianceSubresource(logger log.FieldLogger, result RuntimeResult, linkedComplianceName string) {
	if result.ApiServiceInstance == nil {
		return
	}

	subRes := map[string]interface{}{
		management.ApiServiceInstanceComplianceruntimeresultSubResourceName: linkedComplianceName,
	}
	if err := agent.GetCentralClient().CreateSubResource(result.ApiServiceInstance.ResourceMeta, subRes); err != nil {
		logger.WithError(err).Error("updating compliance runtime result subResource reference for api service instance")
	}
}
