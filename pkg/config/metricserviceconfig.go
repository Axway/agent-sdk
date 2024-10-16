package config

import (
	"net/url"

	"github.com/Axway/agent-sdk/pkg/util/exception"
)

const (
	pathMetricServiceEnable = "agentFeatures.metricServices.enable"
	pathMetricServiceURL    = "agentFeatures.metricServices.url"
	pathRejectOnFail        = "agentFeatures.metricServices.rejectOnFail"
)

type MetricServiceConfig interface {
	MetricServiceEnabled() bool
	GetMetricServiceURL() string
	RejectOnFailEnabled() bool
	validate()
}

type MetricServiceConfiguration struct {
	MetricServiceConfig
	Enable       bool   `config:"enable"`       // set to true to have the sdk initiate the connection to the custom metric service
	URL          string `config:"url"`          // set the url that the agent will connect to the metric service on
	RejectOnFail bool   `config:"rejectOnFail"` // set to true to reject the access request if the quota enforcement call fails
}

func (a *MetricServiceConfiguration) validate() {
	if a.GetMetricServiceURL() == "" {
		exception.Throw(ErrBadConfig.FormatError(pathMetricServiceEnable))
	} else if _, err := url.ParseRequestURI(a.GetMetricServiceURL()); err != nil {
		exception.Throw(ErrBadConfig.FormatError(pathMetricServiceEnable))
	}
}

func (c *MetricServiceConfiguration) MetricServiceEnabled() bool {
	return c.Enable
}

// ProcessSystemSignalsEnabled - True if the agent SDK listens for system signals and manages shutdown
func (c *MetricServiceConfiguration) GetMetricServiceURL() string {
	return c.URL
}

// VersionCheckerEnabled - True if the agent SDK should check for newer versions of the agent.
func (c *MetricServiceConfiguration) RejectOnFailEnabled() bool {
	return c.RejectOnFail
}
