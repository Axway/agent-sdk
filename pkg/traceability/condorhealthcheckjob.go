package traceability

import (
	"fmt"
	"net"
	"net/http"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
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
	return err == nil
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
	// Proxy url parameter is nil since the dialer will use proxy url from traceCfg
	dialer, err := ingestionSingleEntryDialer(nil, &net.Dialer{Timeout: j.agentHealthChecker.timeout})
	if err != nil {
		return err
	}
	conn, err := dialer.Dial(j.agentHealthChecker.protocol, j.agentHealthChecker.host)
	if err != nil {
		return fmt.Errorf("%s connection failed. %s", host, err.Error())
	}
	conn.Close()

	return nil
}

func (j *condorHealthCheckJob) checkHTTPConnection(host string) error {
	request := api.Request{
		Method: http.MethodGet,
		URL:    j.agentHealthChecker.protocol + "://" + j.agentHealthChecker.host,
	}

	client := api.NewClient(j.getTLSConfig(), j.agentHealthChecker.proxyURL)
	response, err := client.Send(request)
	if err != nil {
		return fmt.Errorf("%s connection failed. %s", host, err.Error())
	}
	if response.Code == http.StatusRequestTimeout {
		return fmt.Errorf("%s connection failed. HTTP response: %v", host, response.Code)
	}

	return nil
}

func (j *condorHealthCheckJob) getTLSConfig() config.TLSConfig {
	tls, _ := tlscommon.LoadTLSConfig(j.agentHealthChecker.tlsCfg)
	tlsCfg := tls.ToConfig()
	tlsConfig := config.NewTLSConfig().(*config.TLSConfiguration)
	tlsConfig.InsecureSkipVerify = tlsCfg.InsecureSkipVerify
	tlsConfig.MaxVersion = config.TLSVersion(tlsCfg.MaxVersion)
	tlsConfig.MinVersion = config.TLSVersion(tlsCfg.MinVersion)

	tlsConfig.CipherSuites = make([]config.TLSCipherSuite, 0)
	for _, cipher := range tlsCfg.CipherSuites {
		tlsConfig.CipherSuites = append(tlsConfig.CipherSuites, config.TLSCipherSuite(cipher))
	}
	return tlsConfig
}
