package config

import (
	"net/url"
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
	oldUsageReportingURLEnvVar           = "CENTRAL_LIGHTHOUSEURL"
	oldUsageReportingPublishEnvVar       = "CENTRAL_PUBLISHUSAGE"
	oldUsageReportingPublishMetricEnvVar = "CENTRAL_PUBLISHMETRIC"
	oldUsageReportingIntervalEnvVar      = "CENTRAL_EVENTAGGREGATIONINTERVAL"
	newUsageReportingURLEnvVar           = "CENTRAL_USAGEREPORTING_URL"
	newUsageReportingPublishEnvVar       = "CENTRAL_USAGEREPORTING_PUBLISH"
	newUsageReportingPublishMetricEnvVar = "CENTRAL_USAGEREPORTING_PUBLISHMETRIC"
	newUsageReportingIntervalEnvVar      = "CENTRAL_USAGEREPORTING_INTERVAL"

	// QA EnvVars
	qaUsageReportingScheduleEnvVar        = "QA_CENTRAL_USAGEREPORTING_OFFLINESCHEDULE"
	qaUsageReportingOfflineScheduleEnvVar = "QA_CENTRAL_USAGEREPORTING_OFFLINEREPORTSCHEDULE"

	// Config paths
	pathUsageReportingURL           = "central.usagereporting.url"
	pathUsageReportingPublish       = "central.usagereporting.publish"
	pathUsageReportingPublishMetric = "central.usagereporting.publishMetric"
	pathUsageReportingInterval      = "central.usagereporting.interval"
	pathUsageReportingOffline       = "central.usagereporting.offline"
	pathUsageReportingSchedule      = "central.usagereporting.offlineSchedule"
)

// UsageReportingConfig - Interface to get usage reporting config
type UsageReportingConfig interface {
	GetURL() string
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
	URL               string        `config:"url"`
	Publish           bool          `config:"publish"`
	PublishMetric     bool          `config:"publishMetric"`
	Interval          time.Duration `config:"interval"`
	Offline           bool          `config:"offline"`
	Schedule          string        `config:"offlineSchedule"`
	reportSchedule    string
	reportGranularity int
	qaVars            bool
}

// NewUsageReporting - Creates the default usage reporting config
func NewUsageReporting() UsageReportingConfig {
	return &UsageReportingConfiguration{
		URL:               "https://lighthouse.admin.axway.com",
		Publish:           true,
		PublishMetric:     true,
		Interval:          15 * time.Minute,
		Offline:           false,
		Schedule:          "@hourly",
		reportSchedule:    "@monthly",
		reportGranularity: 900000,
		qaVars:            false,
	}
}

func (u *UsageReportingConfiguration) validateURL() {
	if val := os.Getenv(newUsageReportingURLEnvVar); val != "" {
		return // this env var is set use what has been parsed
	}

	// check if the old env var had a value
	if val := os.Getenv(oldUsageReportingURLEnvVar); val != "" {
		log.DeprecationWarningReplace(oldUsageReportingURLEnvVar, newUsageReportingURLEnvVar)
		u.URL = val
	}
}

func (u *UsageReportingConfiguration) validateInterval() {
	if val := os.Getenv(newUsageReportingIntervalEnvVar); val != "" {
		return // this env var is set use what has been parsed
	}

	// check if the old env var had a value
	if val := os.Getenv(oldUsageReportingIntervalEnvVar); val != "" {
		if value, err := time.ParseDuration(val); err == nil {
			log.DeprecationWarningReplace(oldUsageReportingIntervalEnvVar, newUsageReportingIntervalEnvVar)
			u.Interval = value
		}
	}
}

func (u *UsageReportingConfiguration) validatePublish() {
	if val := os.Getenv(newUsageReportingPublishEnvVar); val != "" {
		return // this env var is set use what has been parsed
	}

	// check if the old env var had a value
	val := os.Getenv(oldUsageReportingPublishEnvVar)
	if val != "" {
		if value, err := strconv.ParseBool(val); err == nil {
			log.DeprecationWarningReplace(oldUsageReportingPublishEnvVar, newUsageReportingPublishEnvVar)
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
		if value, err := strconv.ParseBool(val); err == nil {
			log.DeprecationWarningReplace(oldUsageReportingPublishMetricEnvVar, newUsageReportingPublishMetricEnvVar)
			u.PublishMetric = value
		}
	}
}

func (u *UsageReportingConfiguration) validate() {
	u.validateURL() // DEPRECATE
	if u.URL != "" {
		if _, err := url.ParseRequestURI(u.URL); err != nil {
			exception.Throw(ErrBadConfig.FormatError(pathUsageReportingURL))
		}
	}

	u.validateInterval() // DEPRECATE
	eventAggSeconds := u.Interval
	if eventAggSeconds < 60*time.Second {
		exception.Throw(ErrBadConfig.FormatError(pathUsageReportingInterval))
	}

	u.validatePublish()       // DEPRECATE
	u.validatePublishMetric() // DEPRECATE

	if u.Offline {
		u.validateOffline()
	}
}

func (u *UsageReportingConfiguration) validateOffline() {
	if _, err := cronexpr.Parse(u.Schedule); err != nil {
		exception.Throw(ErrBadConfig.FormatError(pathUsageReportingSchedule))
	}

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

	// if no QA env vars are set then validate the schedule is at least hourly
	if nextTwoRuns[1].Sub(nextTwoRuns[0]) < time.Hour && !u.qaVars {
		exception.Throw(ErrBadConfig.FormatError(pathUsageReportingSchedule))
	}
}

// GetURL - Returns the usage reporting URL
func (u *UsageReportingConfiguration) GetURL() string {
	return u.URL
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
	props.AddStringProperty(pathUsageReportingURL, "https://lighthouse.admin.axway.com", "The URL to publish usage events to in the Amplify platform. Default https://lighthouse.admin.axway.com")
	props.AddBoolProperty(pathUsageReportingPublish, true, "Indicates if the agent can publish usage events to Amplify platform. Default to true")
	props.AddBoolProperty(pathUsageReportingPublishMetric, true, "Indicates if the agent can publish metric events to Amplify platform. Default to false")
	props.AddDurationProperty(pathUsageReportingInterval, 15*time.Minute, "The time interval at which usage and metric events will be generated")
	props.AddBoolProperty(pathUsageReportingOffline, false, "Turn this on to save the usage events to disk for manual upload")
	props.AddStringProperty(pathUsageReportingSchedule, "@hourly", "The schedule at which usage events are generated, for offline mode only")
}

// ParseUsageReportingConfig - Parses the Usage Reporting Config values from the command line
func ParseUsageReportingConfig(props properties.Properties) UsageReportingConfig {
	// Start with the default config
	cfg := NewUsageReporting().(*UsageReportingConfiguration)

	// update the config
	cfg.URL = props.StringPropertyValue(pathUsageReportingURL)
	cfg.Publish = props.BoolPropertyValue(pathUsageReportingPublish)
	cfg.PublishMetric = props.BoolPropertyValue(pathUsageReportingPublishMetric)
	cfg.Interval = props.DurationPropertyValue(pathUsageReportingInterval)
	cfg.Offline = props.BoolPropertyValue(pathUsageReportingOffline)
	cfg.Schedule = props.StringPropertyValue(pathUsageReportingSchedule)

	return cfg
}
