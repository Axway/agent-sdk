package compliance

import (
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type RuntimeResult struct {
	ApiServiceInstance      *management.APIServiceInstance
	Environment             *management.Environment
	ApiService              *management.APIService
	ComplianceScopedEnv     string
	ComplianceRuntimeResult string
	ComplianceAgentName     string
	ComplianceAgentType     string
	RiskScore               float64
}

type RuntimeResults interface {
	AddRuntimeResult(RuntimeResult)
}

type runtimeResults struct {
	logger log.FieldLogger
	items  map[string]RuntimeResult
}

func (r *runtimeResults) AddRuntimeResult(result RuntimeResult) {
	if r.items == nil {
		r.items = make(map[string]RuntimeResult)
	}
	r.items[result.ComplianceRuntimeResult] = result
}
