package config

import (
	"os"
	"strconv"
	"strings"
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
	qaUsageReportingScheduleEnvVar        = "QA_CENTRAL_USAGEREPORTING_OFFLINESCHEDULE"
	qaUsageReportingOfflineScheduleEnvVar = "QA_CENTRAL_USAGEREPORTING_OFFLINEREPORTSCHEDULE"
	qaUsageReportingUsageScheduleEnvVar   = "QA_CENTRAL_USAGEREPORTING_USAGESCHEDULE"

	// Config paths
	pathUsageReportingPublish       = "central.usagereporting.publish"
	pathUsageReportingPublishMetric = "central.usagereporting.publishMetric"
	pathUsageReportingUsageSchedule = "central.usagereporting.usageSchedule"
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
	GetReportInterval() time.Duration
	GetUsageSchedule() string
	IsOfflineMode() bool
	GetSchedule() string
	GetReportSchedule() string
	GetReportGranularity() int
	UsingQAVars() bool
	Validate()
}

// UsageReportingConfiguration - structure to hold all usage reporting settings
type UsageReportingConfiguration struct {
	UsageReportingConfig
	Publish           bool          `config:"publish"`
	PublishMetric     bool          `config:"publishMetric"`
	Interval          time.Duration `config:"interval"`
	UsageSchedule     string        `config:"usageSchedule"`
	Offline           bool          `config:"offline"`
	Schedule          string        `config:"offlineSchedule"`
	URL               string
	reportSchedule    string
	reportGranularity int
	qaVars            bool
}

// NewUsageReporting - Creates the default usage reporting config
func NewUsageReporting(platformURL string) UsageReportingConfig {
	return &UsageReportingConfiguration{
		URL:            platformURL,
		Publish:        true,
		PublishMetric:  true,
		Interval:       15 * time.Minute,
		UsageSchedule:  "@daily",
		Offline:        false,
		Schedule:       "@hourly",
		reportSchedule: "@monthly",
		qaVars:         false,
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

// Validate -
func (u *UsageReportingConfiguration) Validate() {
	u.validateInterval() // DEPRECATE
	eventAgg := u.Interval
	if eventAgg < 60*time.Second {
		exception.Throw(ErrBadConfig.FormatError(pathUsageReportingInterval))
	}

	u.validatePublish() // DEPRECATE
	u.validatePublishMetric()

	if !u.Offline {
		u.validateUsageSchedule()
	} else {
		u.validateOffline()
	}
}

func (u *UsageReportingConfiguration) validateUsageSchedule() {
	// check if the qa env var is set
	if val := os.Getenv(qaUsageReportingUsageScheduleEnvVar); val != "" {
		if _, err := cronexpr.Parse(val); err != nil {
			log.Tracef("Could not use %s (%s) it is not a proper cron schedule", qaUsageReportingUsageScheduleEnvVar, val)
		} else {
			log.Tracef("Using %s (%s) rather than the default (%s) for non-QA", qaUsageReportingUsageScheduleEnvVar, val, u.UsageSchedule)
			u.UsageSchedule = val
			u.qaVars = true
		}
		return
	}

	// Check the cron expressions
	cron, err := cronexpr.Parse(u.UsageSchedule)
	if err != nil {
		exception.Throw(ErrBadConfig.FormatError(pathUsageReportingUsageSchedule))
	}
	checks := 5
	nextRuns := cron.NextN(time.Now(), uint(checks))
	if len(nextRuns) != checks {
		exception.Throw(ErrBadConfig.FormatError(pathUsageReportingUsageSchedule))
	}
	for i := 1; i < checks-1; i++ {
		delta := nextRuns[i].Sub(nextRuns[i-1])
		if delta < time.Hour {
			log.Tracef("%s must be at 1 hour apart", pathUsageReportingUsageSchedule)
			exception.Throw(ErrBadConfig.FormatError(pathUsageReportingUsageSchedule))
		}
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

// GetUsageSchedule - Returns the schedule string for publishing reports
func (u *UsageReportingConfiguration) GetUsageSchedule() string {
	return u.UsageSchedule
}

// GetReportSchedule - Returns the offline schedule string for creating reports
func (u *UsageReportingConfiguration) GetReportSchedule() string {
	return u.reportSchedule
}

// GetReportGranularity - Returns the granularity used in the offline reports
func (u *UsageReportingConfiguration) GetReportGranularity() int {
	if u.reportGranularity == 0 {
		return int(u.Interval.Milliseconds())
	}
	return u.reportGranularity
}

// UsingQAVars - Returns the offline boolean
func (u *UsageReportingConfiguration) UsingQAVars() bool {
	return u.qaVars
}

// AddUsageReportingProperties - Adds the command properties needed for Usage Reporting Settings
func AddUsageReportingProperties(props properties.Properties) {
	props.AddBoolProperty(pathUsageReportingPublish, true, "Indicates if the agent can publish usage events to Amplify platform. Default to true")
	props.AddBoolProperty(pathUsageReportingPublishMetric, true, "Indicates if the agent can publish metric events to Amplify platform. Default to true")
	props.AddDurationProperty(pathUsageReportingInterval, 15*time.Minute, "The time interval at which usage and metric events will be generated", properties.WithLowerLimit(5*time.Minute))
	props.AddStringProperty(pathUsageReportingUsageSchedule, "@daily", "The schedule at usage events are sent to the platform")
	props.AddBoolProperty(pathUsageReportingOffline, false, "Turn this on to save the usage events to disk for manual upload")
	props.AddStringProperty(pathUsageReportingSchedule, "@hourly", "The schedule at which usage events are generated, for offline mode only")
}

// ParseUsageReportingConfig - Parses the Usage Reporting Config values from the command line
func ParseUsageReportingConfig(props properties.Properties) UsageReportingConfig {
	// Start with the default config
	platformURL := strings.TrimRight(props.StringPropertyValue(pathPlatformURL), urlCutSet)
	cfg := NewUsageReporting(platformURL).(*UsageReportingConfiguration)

	// update the config
	cfg.Publish = props.BoolPropertyValue(pathUsageReportingPublish)
	cfg.PublishMetric = props.BoolPropertyValue(pathUsageReportingPublishMetric)
	cfg.Interval = props.DurationPropertyValue(pathUsageReportingInterval)
	cfg.UsageSchedule = props.StringPropertyValue(pathUsageReportingUsageSchedule)
	cfg.Offline = props.BoolPropertyValue(pathUsageReportingOffline)
	cfg.Schedule = props.StringPropertyValue(pathUsageReportingSchedule)

	return cfg
}
