package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/Axway/agent-sdk/pkg/util/exception"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/gorhill/cronexpr"
)

const (
	// DEPRECATE remove old and new env vars as well as checks below
	oldUsageReportingPublishMetricEnvVar = "CENTRAL_USAGEREPORTING_PUBLISHMETRIC"
	oldUsageReportingIntervalEnvVar      = "CENTRAL_USAGEREPORTING_INTERVAL"
	newMetricReportingPublishEnvVar      = "CENTRAL_METRICREPORTING_PUBLISH"
	newMetricReportingScheduleEnvVar     = "CENTRAL_METRICREPORTING_SCHEDULE"

	// Config paths
	pathMetricReportingPublish  = "central.metricreporting.publish"
	pathMetricReportingSchedule = "central.metricreporting.schedule"

	qaMetricReportingScheduleEnvVar = "QA_CENTRAL_METRICREPORTING_SCHEDULE"
)

// MetricReportingConfig - Interface to get metric reporting config
type MetricReportingConfig interface {
	CanPublish() bool
	GetSchedule() string
	GetReportGranularity() int
	UsingQAVars() bool
	Validate()
}

// MetricReportingConfiguration - structure to hold all metric reporting settings
type MetricReportingConfiguration struct {
	MetricReportingConfig
	Publish     bool   `config:"publish"`
	Schedule    string `config:"schedule"`
	granularity time.Duration
	qaVars      bool
}

// NewMetricReporting - Creates the default metric reporting config
func NewMetricReporting() MetricReportingConfig {
	return &MetricReportingConfiguration{
		Publish:     true,
		Schedule:    "@hourly",
		granularity: time.Hour,
		qaVars:      false,
	}
}

// Validate -
func (m *MetricReportingConfiguration) Validate() {
	m.validatePublish()
	// Parse and validate interval from deprecated config for backward compatibility
	m.validateInterval()
	m.validateSchedule()
}

func (u *MetricReportingConfiguration) validateInterval() {
	if val := os.Getenv(newMetricReportingScheduleEnvVar); val != "" {
		return // this env var is set use what has been parsed
	}

	// check if the old env var had a value
	if val := os.Getenv(oldUsageReportingIntervalEnvVar); val != "" {
		if value, err := time.ParseDuration(val); err == nil {
			log.DeprecationWarningReplace(oldUsageReportingIntervalEnvVar, newMetricReportingScheduleEnvVar)
			if value < 60*time.Second {
				exception.Throw(ErrBadConfig.FormatError(oldUsageReportingIntervalEnvVar))
			}

			intervalSchedule := fmt.Sprintf("*/%d * * * *", int(value.Minutes()))
			if value > time.Hour {
				intervalSchedule = fmt.Sprintf("%d %d * * *", time.Now().Minute(), int(value.Hours()))
			}

			_, err := cronexpr.Parse(intervalSchedule)
			if err != nil {
				exception.Throw(ErrBadConfig.FormatError(oldUsageReportingIntervalEnvVar))
			}
			u.Schedule = intervalSchedule
		}
	}
}

func (m *MetricReportingConfiguration) validatePublish() {
	if val := os.Getenv(newMetricReportingPublishEnvVar); val != "" {
		return // this env var is set use what has been parsed
	}

	// check if the old env var had a value
	if val := os.Getenv(oldUsageReportingPublishMetricEnvVar); val != "" {
		if value, err := strconv.ParseBool(val); err == nil {
			log.DeprecationWarningReplace(oldUsageReportingPublishMetricEnvVar, newMetricReportingPublishEnvVar)
			m.Publish = value
		}
	}
}

func (u *MetricReportingConfiguration) validateSchedule() {
	// check if the qa env var is set
	if val := os.Getenv(qaMetricReportingScheduleEnvVar); val != "" {
		if _, err := cronexpr.Parse(val); err != nil {
			log.Tracef("Could not use %s (%s) it is not a proper cron schedule", qaMetricReportingScheduleEnvVar, val)
		} else {
			log.Tracef("Using %s (%s) rather than the default (%s) for non-QA", qaMetricReportingScheduleEnvVar, val, u.Schedule)
			u.Schedule = val
			u.qaVars = true
		}
		return
	}

	// Check the cron expressions
	cron, err := cronexpr.Parse(u.Schedule)
	if err != nil {
		exception.Throw(ErrBadConfig.FormatError(pathMetricReportingSchedule))
	}
	checks := 5
	nextRuns := cron.NextN(time.Now(), uint(checks))
	if len(nextRuns) != checks {
		exception.Throw(ErrBadConfig.FormatError(pathMetricReportingSchedule))
	}
	for i := 1; i < checks-1; i++ {
		u.granularity = nextRuns[i].Sub(nextRuns[i-1])
		if u.granularity < time.Minute*15 {
			log.Tracef("%s must be at 15 min apart", pathMetricReportingSchedule)
			exception.Throw(ErrBadConfig.FormatError(pathMetricReportingSchedule))
		}
	}
}

// CanPublish - Returns the publish metric boolean
func (u *MetricReportingConfiguration) CanPublish() bool {
	return u.Publish
}

// GetSchedule - Returns the schedule string
func (u *MetricReportingConfiguration) GetSchedule() string {
	return u.Schedule
}

// GetReportGranularity - Returns the schedule string
func (u *MetricReportingConfiguration) GetReportGranularity() int {
	return int(u.granularity.Milliseconds())
}

// UsingQAVars - Returns the offline boolean
func (u *MetricReportingConfiguration) UsingQAVars() bool {
	return u.qaVars
}

// AddMetricReportingProperties - Adds the command properties needed for Metric Reporting Settings
func AddMetricReportingProperties(props properties.Properties) {
	props.AddBoolProperty(pathMetricReportingPublish, true, "Indicates if the agent can publish metric events to Amplify platform. Default to true")
	props.AddStringProperty(pathMetricReportingSchedule, "@hourly", "The schedule at metric events are sent to the platform")
}

// ParseUsageReportingConfig - Parses the Usage Reporting Config values from the command line
func ParseMetricReportingConfig(props properties.Properties) MetricReportingConfig {
	// Start with the default config
	cfg := NewMetricReporting().(*MetricReportingConfiguration)

	// update the config
	cfg.Publish = props.BoolPropertyValue(pathMetricReportingPublish)
	cfg.Schedule = props.StringPropertyValue(pathMetricReportingSchedule)

	return cfg
}
