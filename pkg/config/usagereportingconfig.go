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
	oldUsageReportingPublishEnvVar  = "CENTRAL_PUBLISHUSAGE"
	newUsageReportingPublishEnvVar  = "CENTRAL_USAGEREPORTING_PUBLISH"
	oldUsageReportingScheduleEnvVar = "CENTRAL_USAGEREPORTING_USAGESCHEDULE"
	newUsageReportingScheduleEnvVar = "CENTRAL_USAGEREPORTING_SCHEDULE"

	// QA EnvVars
	qaUsageReportingScheduleEnvVar        = "QA_CENTRAL_USAGEREPORTING_OFFLINESCHEDULE"
	qaUsageReportingOfflineScheduleEnvVar = "QA_CENTRAL_USAGEREPORTING_OFFLINEREPORTSCHEDULE"
	qaUsageReportingUsageScheduleEnvVar   = "QA_CENTRAL_USAGEREPORTING_SCHEDULE"

	// Config paths
	pathUsageReportingPublish         = "central.usagereporting.publish"
	pathUsageReportingSchedule        = "central.usagereporting.schedule"
	pathUsageReportingOffline         = "central.usagereporting.offline"
	pathUsageReportingOfflineSchedule = "central.usagereporting.offlineSchedule"
)

// UsageReportingConfig - Interface to get usage reporting config
type UsageReportingConfig interface {
	GetURL() string
	CanPublish() bool
	GetReportInterval() time.Duration
	GetSchedule() string
	IsOfflineMode() bool
	GetOfflineSchedule() string
	GetReportSchedule() string
	GetReportGranularity() int
	UsingQAVars() bool
	Validate()
}

// UsageReportingConfiguration - structure to hold all usage reporting settings
type UsageReportingConfiguration struct {
	UsageReportingConfig
	Publish           bool   `config:"publish"`
	Schedule          string `config:"schedule"`
	Offline           bool   `config:"offline"`
	OfflineSchedule   string `config:"offlineSchedule"`
	URL               string
	reportSchedule    string
	reportGranularity int
	qaVars            bool
}

// NewUsageReporting - Creates the default usage reporting config
func NewUsageReporting(platformURL string) UsageReportingConfig {
	return &UsageReportingConfiguration{
		URL:             platformURL,
		Publish:         true,
		Schedule:        "@daily",
		Offline:         false,
		OfflineSchedule: "@hourly",
		reportSchedule:  "@monthly",
		qaVars:          false,
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

// Validate -
func (u *UsageReportingConfiguration) Validate() {
	u.validatePublish()

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
			log.Tracef("Using %s (%s) rather than the default (%s) for non-QA", qaUsageReportingUsageScheduleEnvVar, val, u.Schedule)
			u.Schedule = val
			u.qaVars = true
		}
		return
	}

	if val := os.Getenv(newUsageReportingScheduleEnvVar); val == "" {
		if val = os.Getenv(oldUsageReportingScheduleEnvVar); val != "" {
			log.DeprecationWarningReplace(oldUsageReportingScheduleEnvVar, newUsageReportingScheduleEnvVar)
			u.Schedule = val
		}
	}

	// Check the cron expressions
	cron, err := cronexpr.Parse(u.Schedule)
	if err != nil {
		exception.Throw(ErrBadConfig.FormatError(pathUsageReportingSchedule))
	}
	checks := 5
	nextRuns := cron.NextN(time.Now(), uint(checks))
	if len(nextRuns) != checks {
		exception.Throw(ErrBadConfig.FormatError(pathUsageReportingSchedule))
	}
	for i := 1; i < checks-1; i++ {
		delta := nextRuns[i].Sub(nextRuns[i-1])
		if delta < time.Hour {
			log.Tracef("%s must be at 1 hour apart", pathUsageReportingSchedule)
			exception.Throw(ErrBadConfig.FormatError(pathUsageReportingSchedule))
		}
	}
}

func (u *UsageReportingConfiguration) validateOffline() {
	if _, err := cronexpr.Parse(u.OfflineSchedule); err != nil {
		exception.Throw(ErrBadConfig.FormatError(pathUsageReportingOfflineSchedule))
	}

	// reporting is offline, lets read the QA env vars
	if val := os.Getenv(qaUsageReportingScheduleEnvVar); val != "" {
		if _, err := cronexpr.Parse(val); err != nil {
			log.Tracef("Could not use %s (%s) it is not a proper cron schedule", qaUsageReportingScheduleEnvVar, val)
		} else {
			log.Tracef("Using %s (%s) rather than the default (%s) for non-QA", qaUsageReportingScheduleEnvVar, val, u.OfflineSchedule)
			u.OfflineSchedule = val
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
	cron, err := cronexpr.Parse(u.OfflineSchedule)
	if err != nil {
		exception.Throw(ErrBadConfig.FormatError(pathUsageReportingOfflineSchedule))
	}
	nextTwoRuns := cron.NextN(time.Now(), 2)
	if len(nextTwoRuns) != 2 {
		exception.Throw(ErrBadConfig.FormatError(pathUsageReportingOfflineSchedule))
	}
	u.reportGranularity = int(nextTwoRuns[1].Sub(nextTwoRuns[0]).Milliseconds())

	// if no QA env vars are set then validate the schedule is at least hourly
	if nextTwoRuns[1].Sub(nextTwoRuns[0]) < time.Hour && !u.qaVars {
		exception.Throw(ErrBadConfig.FormatError(pathUsageReportingOfflineSchedule))
	}
}

// GetURL - Returns the usage reporting URL
func (u *UsageReportingConfiguration) GetURL() string {
	return u.URL
}

// CanPublish - Returns the publish boolean
func (u *UsageReportingConfiguration) CanPublish() bool {
	return u.Publish
}

// IsOfflineMode - Returns the offline boolean
func (u *UsageReportingConfiguration) IsOfflineMode() bool {
	return u.Offline
}

// GetSchedule - Returns the schedule string
func (u *UsageReportingConfiguration) GetOfflineSchedule() string {
	return u.OfflineSchedule
}

// GetSchedule - Returns the schedule string for publishing reports
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

// AddUsageReportingProperties - Adds the command properties needed for Usage Reporting Settings
func AddUsageReportingProperties(props properties.Properties) {
	props.AddBoolProperty(pathUsageReportingPublish, true, "Indicates if the agent can publish usage events to Amplify platform. Default to true")
	props.AddStringProperty(pathUsageReportingSchedule, "@daily", "The schedule at usage events are sent to the platform")
	props.AddBoolProperty(pathUsageReportingOffline, false, "Turn this on to save the usage events to disk for manual upload")
	props.AddStringProperty(pathUsageReportingOfflineSchedule, "@hourly", "The schedule at which usage events are generated, for offline mode only")
}

// ParseUsageReportingConfig - Parses the Usage Reporting Config values from the command line
func ParseUsageReportingConfig(props properties.Properties) UsageReportingConfig {
	// Start with the default config
	platformURL := strings.TrimRight(props.StringPropertyValue(pathPlatformURL), urlCutSet)
	cfg := NewUsageReporting(platformURL).(*UsageReportingConfiguration)

	// update the config
	cfg.Publish = props.BoolPropertyValue(pathUsageReportingPublish)
	cfg.Schedule = props.StringPropertyValue(pathUsageReportingSchedule)
	cfg.Offline = props.BoolPropertyValue(pathUsageReportingOffline)
	cfg.OfflineSchedule = props.StringPropertyValue(pathUsageReportingOfflineSchedule)

	return cfg
}
