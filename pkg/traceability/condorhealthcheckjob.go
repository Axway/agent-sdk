package traceability

import (
	"fmt"
	"net"
	"net/http"
	"net/url"

	"golang.org/x/net/proxy"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/jobs"
)

const healthcheckCondor = "Traceability connectivity"

// condorHealthCheckJob -
type condorHealthCheckJob struct {
	jobs.Job
	agentHealthChecker *traceabilityAgentHealthChecker
}

// Ready -
func (j *condorHealthCheckJob) Ready() bool {
	err := j.checkConnections(healthcheckCondor)
	if err != nil {
		return false
	}
	return true
}

// Status -
func (j *condorHealthCheckJob) Status() error {
	err := j.checkConnections(healthcheckCondor)
	if err != nil {
		return err
	}
	return nil
}

// Execute -
func (j *condorHealthCheckJob) Execute() error {
	return nil
}

func (j *condorHealthCheckJob) checkConnections(name string) error {
	if j.agentHealthChecker.protocol == "tcp" {
		return j.checkTCPConnection(name)
	}
	return j.checkHTTPConnection(name)
}

func (j *condorHealthCheckJob) checkTCPConnection(host string) error {
	var err error
	defaultDialer := &net.Dialer{Timeout: j.agentHealthChecker.timeout}
	d := proxy.FromEnvironmentUsing(defaultDialer)

	// Setup the proxy if needed
	if j.agentHealthChecker.proxyURL != "" {
		uri, err := url.Parse(j.agentHealthChecker.proxyURL)
		if err != nil {
			return fmt.Errorf("%s proxy could not be parsed. %s", host, err.Error())
		}
		d, err = proxy.FromURL(uri, defaultDialer)
		if err != nil {
			return fmt.Errorf("%s could not setup proxy. %s", host, err.Error())
		}
	}

	_, err = d.Dial(j.agentHealthChecker.protocol, j.agentHealthChecker.host)
	if err != nil {
		return fmt.Errorf("%s connection failed. %s", host, err.Error())
	}

	return nil
}

func (j *condorHealthCheckJob) checkHTTPConnection(host string) error {
	request := api.Request{
		Method: http.MethodConnect,
		URL:    j.agentHealthChecker.protocol + "://" + j.agentHealthChecker.host,
	}

	client := api.NewClient(nil, j.agentHealthChecker.proxyURL)
	response, err := client.Send(request)
	if err != nil {
		return fmt.Errorf("%s connection failed. %s", host, err.Error())
	}
	if response.Code == http.StatusRequestTimeout {
		return fmt.Errorf("%s connection failed. HTTP response: %v", host, response.Code)
	}

	return nil
}
