package config

import (
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
	oldUsageReportingPublishEnvVar       = "CENTRAL_PUBLISHUSAGE"
	oldUsageReportingPublishMetricEnvVar = "CENTRAL_PUBLISHMETRIC"
	oldUsageReportingIntervalEnvVar      = "CENTRAL_EVENTAGGREGATIONINTERVAL"
	newUsageReportingPublishEnvVar       = "CENTRAL_USAGEREPORTING_PUBLISH"
	newUsageReportingPublishMetricEnvVar = "CENTRAL_USAGEREPORTING_PUBLISHMETRIC"
	newUsageReportingIntervalEnvVar      = "CENTRAL_USAGEREPORTING_INTERVAL"

	// QA EnvVars
	qaUsageReportingScheduleEnvVar        = "QA_CENTRAL_USAGEREPORTING_SCHEDULE"
	qaUsageReportingOfflineScheduleEnvVar = "QA_CENTRAL_USAGEREPORTING_OFFLINESCHEDULE"

	// Config paths
	pathUsageReportingPublish       = "central.usagereporting.publish"
	pathUsageReportingPublishMetric = "central.usagereporting.publishMetric"
	pathUsageReportingInterval      = "central.usagereporting.interval"
	pathUsageReportingOffline       = "central.usagereporting.offline"
	pathUsageReportingSchedule      = "central.usagereporting.schedule"
)

// UsageReportingConfig - Interface to get usage reporting config
type UsageReportingConfig interface {
	CanPublishUsage() bool
	CanPublishMetric() bool
	GetInterval() time.Duration
	IsOfflineMode() bool
	GetSchedule() string
	GetReportSchedule() string
	GetReportGranularity() int
	UsingQAVars() bool
	validate()
}

// UsageReportingConfiguration - structure to hold all usage reporting settings
type UsageReportingConfiguration struct {
	UsageReportingConfig
	Publish           bool          `config:"publish"`
	PublishMetric     bool          `config:"publishMetric"`
	Interval          time.Duration `config:"interval"`
	Offline           bool          `config:"offline"`
	Schedule          string        `config:"schedule"`
	reportSchedule    string
	reportGranularity int
	qaVars            bool
}

// NewUsageReporting - Creates the default usage reporting config
func NewUsageReporting() UsageReportingConfig {
	return &UsageReportingConfiguration{
		Publish:           true,
		PublishMetric:     false,
		Interval:          15 * time.Minute,
		Offline:           false,
		Schedule:          "@hourly",
		reportSchedule:    "@monthly",
		reportGranularity: 3600000,
		qaVars:            false,
	}
}

func (u *UsageReportingConfiguration) validateInterval() {
	if val := os.Getenv(newUsageReportingPublishEnvVar); val != "" {
		return // this env var is set use what has been parsed
	}

	// check if the old env var had a value
	if val := os.Getenv(oldUsageReportingIntervalEnvVar); val != "" {
		if value, err := time.ParseDuration(val); err != nil {
			u.Interval = value
		}
	}
}

func (u *UsageReportingConfiguration) validatePublish() {
	if val := os.Getenv(newUsageReportingPublishEnvVar); val != "" {
		return // this env var is set use what has been parsed
	}

	// check if the old env var had a value
	if val := os.Getenv(oldUsageReportingPublishEnvVar); val != "" {
		if value, err := strconv.ParseBool(val); err != nil {
			u.Publish = value
		}
	}
}

func (u *UsageReportingConfiguration) validatePublishMetric() {
	if val := os.Getenv(newUsageReportingPublishMetricEnvVar); val != "" {
		return // this env var is set use what has been parsed
	}

	// check if the old env var had a value
	if val := os.Getenv(oldUsageReportingPublishMetricEnvVar); val != "" {
		if value, err := strconv.ParseBool(val); err != nil {
			u.PublishMetric = value
		}
	}
}

func (u *UsageReportingConfiguration) validate() {
	u.validateInterval() // DEPRECATE
	eventAggSeconds := u.Interval
	if eventAggSeconds < 60000 {
		exception.Throw(ErrBadConfig.FormatError(pathUsageReportingInterval))
	}

	u.validatePublish()       // DEPRECATE
	u.validatePublishMetric() // DEPRECATE

	if _, err := cronexpr.Parse(u.Schedule); err != nil {
		exception.Throw(ErrBadConfig.FormatError(pathUsageReportingSchedule))
	}

	if u.Offline {
		// reporting is offline, lets read the QA env vars
		if val := os.Getenv(qaUsageReportingScheduleEnvVar); val != "" {
			if _, err := cronexpr.Parse(val); err != nil {
				log.Tracef("Could not use %s (%s) it is not a proper cron schedule", qaUsageReportingScheduleEnvVar, val)
			} else {
				log.Tracef("Using %s (%s) rather than the default (%s) for non-QA", qaUsageReportingScheduleEnvVar, val, u.Schedule)
				u.Schedule = val
				u.qaVars = true
			}
		}

		if val := os.Getenv(qaUsageReportingOfflineScheduleEnvVar); val != "" {
			if _, err := cronexpr.Parse(val); err != nil {
				log.Tracef("Could not use %s (%s) it is not a proper cron schedule", qaUsageReportingOfflineScheduleEnvVar, val)
			} else {
				log.Tracef("Using %s (%s) rather than the default (%s) for non-QA", qaUsageReportingOfflineScheduleEnvVar, val, u.reportSchedule)
				u.reportSchedule = val
				u.qaVars = true
			}
		}

		// Check the cron expressions
		cron, err := cronexpr.Parse(u.Schedule)
		if err != nil {
			exception.Throw(ErrBadConfig.FormatError(pathUsageReportingSchedule))
		}
		nextTwoRuns := cron.NextN(time.Now(), 2)
		if len(nextTwoRuns) != 2 {
			exception.Throw(ErrBadConfig.FormatError(pathUsageReportingSchedule))
		}
		u.reportGranularity = int(nextTwoRuns[1].Sub(nextTwoRuns[0]).Milliseconds())
	}
}

// CanPublishUsage - Returns the publish boolean
func (u *UsageReportingConfiguration) CanPublishUsage() bool {
	return u.Publish
}

// CanPublishMetric - Returns the publish metric boolean
func (u *UsageReportingConfiguration) CanPublishMetric() bool {
	return u.PublishMetric
}

// GetInterval - Returns the publish interval
func (u *UsageReportingConfiguration) GetInterval() time.Duration {
	return u.Interval
}

// IsOfflineMode - Returns the offline boolean
func (u *UsageReportingConfiguration) IsOfflineMode() bool {
	return u.Offline
}

// GetSchedule - Returns the schedule string
func (u *UsageReportingConfiguration) GetSchedule() string {
	return u.Schedule
}

// GetReportSchedule - Returns the offline schedule string for creating reports
func (u *UsageReportingConfiguration) GetReportSchedule() string {
	return u.reportSchedule
}

// GetReportGranularity - Returns the granularity used in the offline reports
func (u *UsageReportingConfiguration) GetReportGranularity() int {
	return u.reportGranularity
}

// UsingQAVars - Returns the offline boolean
func (u *UsageReportingConfiguration) UsingQAVars() bool {
	return u.qaVars
}

// AddUsageReportingProperties - Adds the command properties needed for Uage Reporting Settings
func AddUsageReportingProperties(props properties.Properties) {
	props.AddBoolProperty(pathUsageReportingPublish, true, "Indicates if the agent can publish usage event to Amplify platform. Default to true")
	props.AddBoolProperty(pathUsageReportingPublishMetric, false, "Indicates if the agent can publish metric event to Amplify platform. Default to false")
	props.AddDurationProperty(pathUsageReportingInterval, 15*time.Minute, "The time interval at which usage and metric event will be generated")
	props.AddBoolProperty(pathUsageReportingOffline, false, "Turn this on to save the usage events to disk for manual upload")
	props.AddStringProperty(pathUsageReportingSchedule, "@hourly", "The schedule at which usage events are generated, for offline mode only")
}

// ParseUsageReportingConfig - Parses the Usage Reporting Config values from the command line
func ParseUsageReportingConfig(props properties.Properties) UsageReportingConfig {
	// Start with the default config
	cfg := NewUsageReporting().(*UsageReportingConfiguration)

	// update the config
	cfg.Publish = props.BoolPropertyValue(pathUsageReportingPublish)
	cfg.PublishMetric = props.BoolPropertyValue(pathUsageReportingPublishMetric)
	cfg.Interval = props.DurationPropertyValue(pathUsageReportingInterval)
	cfg.Offline = props.BoolPropertyValue(pathUsageReportingOffline)
	cfg.Schedule = props.StringPropertyValue(pathUsageReportingSchedule)

	return cfg
}
