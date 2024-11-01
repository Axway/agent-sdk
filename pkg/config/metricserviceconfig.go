package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/Axway/agent-sdk/pkg/util/exception"
)

const (
	pathMetricServices      = "agentFeatures.metricServices"
	pathMetricServiceEnable = "enable"
	pathMetricServiceURL    = "url"
	pathRejectOnFail        = "rejectOnFail"
)

var metricServiceProps = []string{
	pathMetricServiceEnable,
	pathMetricServiceURL,
	pathRejectOnFail,
}

func addMetricServicesProperties(props properties.Properties) {
	props.AddObjectSliceProperty(pathMetricServices, metricServiceProps)
}

type MetricServiceConfig interface {
	MetricServiceEnabled() bool
	GetMetricServiceURL() string
	RejectOnFailEnabled() bool
	validate()
}

type MetricServiceConfiguration struct {
	MetricServiceConfig
	Enable       bool   // set to true to have the sdk initiate the connection to the custom metric service
	URL          string // set the url that the agent will connect to the metric service on
	RejectOnFail bool   // set to true to reject the access request if the quota enforcement call fails
}

type metricsvcconfig struct {
	Enable       string `json:"enable"`
	URL          string `json:"url"`
	RejectOnFail string `json:"rejectOnFail"`
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

func parseMetricServicesConfig(props properties.Properties) ([]MetricServiceConfiguration, error) {
	svcCfgList := props.ObjectSlicePropertyValue(pathMetricServices)

	cfgs := []MetricServiceConfiguration{}

	for _, svcCfgProps := range svcCfgList {
		svcCfg := metricsvcconfig{}

		buf, _ := json.Marshal(svcCfgProps)
		err := json.Unmarshal(buf, &svcCfg)
		if err != nil {
			return nil, fmt.Errorf("error parsing metrics service configuration, %s", err)
		}
		cfgs = append(cfgs, MetricServiceConfiguration{
			Enable:       parseBool(svcCfg.Enable),
			URL:          svcCfg.URL,
			RejectOnFail: parseBool(svcCfg.RejectOnFail),
		})
	}

	return cfgs, nil
}

func parseBool(str string) bool {
	v, err := strconv.ParseBool(str)
	if err != nil {
		return false
	}
	return v
}
