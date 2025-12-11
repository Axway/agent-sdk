package sampling

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/jobs"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type apiAppErrorSamplingResetJob struct {
	jobs.Job
	logger log.FieldLogger
}

func newAPIAppErrorSamplingResetJob() *apiAppErrorSamplingResetJob {
	logger := log.NewFieldLogger().
		WithPackage("sdk.traceability").
		WithComponent("apiAppErrorSamplingResetJob")
	return &apiAppErrorSamplingResetJob{logger: logger}
}

func (j *apiAppErrorSamplingResetJob) Ready() bool {
	status, _ := hc.GetGlobalStatus()
	return status == string(hc.OK)
}

func (j *apiAppErrorSamplingResetJob) Status() error {
	if status, _ := hc.GetGlobalStatus(); status != string(hc.OK) {
		err := fmt.Errorf("agent is marked as not running")
		j.logger.WithError(err).Trace("status failed")
		return err
	}
	return nil
}

func (j *apiAppErrorSamplingResetJob) Execute() error {
	agentSamples.samplingLock.Lock()
	defer agentSamples.samplingLock.Unlock()
	j.logger.Trace("removing every api-app key pair")
	agentSamples.apiAppErrorSampling = make(map[string]struct{})
	return nil
}
