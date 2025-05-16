package compliance

import (
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type RuntimeResult struct {
	ComplianceRuntimeResult string  `json:"complianceRuntimeResult"`
	RiskScore               float64 `json:"riskScore"`
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
