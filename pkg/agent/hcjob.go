package agent

import (
	"fmt"

	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type healthChecker interface {
	Healthcheck(_ string) *hc.Status
}

type centralHealthCheckJob struct {
	logger        log.FieldLogger
	healthChecker healthChecker
}

func newCentralHealthCheckJob(checker healthChecker) *centralHealthCheckJob {
	logger := log.NewFieldLogger().WithPackage("agent.sdk").WithComponent("centralHealthCheckJob")
	return &centralHealthCheckJob{
		logger:        logger,
		healthChecker: checker,
	}
}

func (c *centralHealthCheckJob) Execute() error {
	return c.check()
}

func (c *centralHealthCheckJob) Status() error {
	return c.check()
}

func (c *centralHealthCheckJob) Ready() bool {
	return c.check() == nil
}

func (c *centralHealthCheckJob) check() error {
	status := c.healthChecker.Healthcheck("")
	if status == nil || status.Result != hc.OK {
		err := fmt.Errorf("central health check status is not OK")
		c.logger.Info(err)
		return err
	}
	return nil
}
