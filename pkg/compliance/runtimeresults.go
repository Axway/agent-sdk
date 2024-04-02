package compliance

import (
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type RuntimeResult struct {
	APIServiceInstance string `json:"apiServiceInstance"`
	HighCount          int64  `json:"highCount"`
	MediumCount        int64  `json:"mediumCount"`
	LowCount           int64  `json:"lowCount"`
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
	r.items[result.APIServiceInstance] = result
}
