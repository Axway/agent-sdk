package config

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/Axway/agent-sdk/pkg/util/exception"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func validateUsageReporting(cfg UsageReportingConfig) (err error) {
	exception.Block{
		Try: func() {
			cfg.Validate()
		},
		Catch: func(e error) {
			err = e
		},
	}.Do()
	return
}

type expected struct {
	url                string
	publish            bool
	metric             bool
	subscriptionMetric bool
	interval           time.Duration
	offline            bool
	schedule           string
	reportSchedule     string
	granularity        int
	qaVars             bool
}

var defaultExpected = expected{
	publish:            true,
	metric:             true,
	subscriptionMetric: false,
	interval:           15 * time.Minute,
	offline:            false,
	schedule:           "@hourly",
	reportSchedule:     "@monthly",
	granularity:        int((15 * time.Minute).Milliseconds()),
	qaVars:             false,
}

func validateconfig(t *testing.T, expVals expected, cfg UsageReportingConfig) {
	assert.Equal(t, expVals.url, cfg.GetURL())
	assert.Equal(t, expVals.publish, cfg.CanPublishUsage())
	assert.Equal(t, expVals.metric, cfg.CanPublishMetric())
	assert.Equal(t, expVals.interval, cfg.GetInterval())
	assert.Equal(t, expVals.offline, cfg.IsOfflineMode())
	assert.Equal(t, expVals.schedule, cfg.GetSchedule())
	assert.Equal(t, expVals.reportSchedule, cfg.GetReportSchedule())
	assert.Equal(t, expVals.granularity, cfg.GetReportGranularity())
	assert.Equal(t, expVals.qaVars, cfg.UsingQAVars())
}

func TestUsageReportingConfigEnvVarMigration(t *testing.T) {
	rootCmd := &cobra.Command{
		Use: "test",
	}

	props := properties.NewProperties(rootCmd)
	AddUsageReportingProperties(props)

	// Test Interval old env vars
	os.Setenv(oldUsageReportingIntervalEnvVar, "30m")
	expected := defaultExpected
	expected.interval = 30 * time.Minute
	expected.granularity = int((30 * time.Minute).Milliseconds())

	cfg := ParseUsageReportingConfig(props)
	assert.NotNil(t, cfg)
	err := validateUsageReporting(cfg)
	assert.Nil(t, err)
	validateconfig(t, expected, cfg)

	// Test Interval new env vars
	os.Setenv(newUsageReportingIntervalEnvVar, defaultExpected.interval.String())

	expected = defaultExpected
	cfg = ParseUsageReportingConfig(props)
	assert.NotNil(t, cfg)
	err = validateUsageReporting(cfg)
	assert.Nil(t, err)
	validateconfig(t, expected, cfg)

	// Test Publish old env vars
	os.Setenv(oldUsageReportingPublishEnvVar, "false")
	expected = defaultExpected
	expected.publish = false

	cfg = ParseUsageReportingConfig(props)
	assert.NotNil(t, cfg)
	err = validateUsageReporting(cfg)
	assert.Nil(t, err)
	validateconfig(t, expected, cfg)

	// Test Publish new env vars
	os.Setenv(newUsageReportingPublishEnvVar, strconv.FormatBool(defaultExpected.publish))

	expected = defaultExpected
	cfg = ParseUsageReportingConfig(props)
	assert.NotNil(t, cfg)
	err = validateUsageReporting(cfg)
	assert.Nil(t, err)
	validateconfig(t, expected, cfg)

	// Test PublishMetric old env vars
	os.Setenv(oldUsageReportingPublishMetricEnvVar, "true")
	expected = defaultExpected
	expected.metric = true

	cfg = ParseUsageReportingConfig(props)
	assert.NotNil(t, cfg)
	err = validateUsageReporting(cfg)
	assert.Nil(t, err)
	validateconfig(t, expected, cfg)

	// Test PublishMetric new env vars
	os.Setenv(newUsageReportingPublishMetricEnvVar, strconv.FormatBool(defaultExpected.metric))

	expected = defaultExpected
	cfg = ParseUsageReportingConfig(props)
	assert.NotNil(t, cfg)
	err = validateUsageReporting(cfg)
	assert.Nil(t, err)
	validateconfig(t, expected, cfg)
}

func TestUsageReportingConfigProperties(t *testing.T) {
	rootCmd := &cobra.Command{
		Use: "test",
	}

	props := properties.NewProperties(rootCmd)

	// Test default config
	AddUsageReportingProperties(props)

	cfg := ParseUsageReportingConfig(props)
	assert.NotNil(t, cfg)

	err := validateUsageReporting(cfg)
	assert.Nil(t, err)

	validateconfig(t, defaultExpected, cfg)

	// invalid Interval
	currentInterval := cfg.GetInterval()
	cfg.(*UsageReportingConfiguration).Interval = time.Millisecond
	err = validateUsageReporting(cfg)
	assert.NotNil(t, err)
	cfg.(*UsageReportingConfiguration).Interval = currentInterval

	// invalid UsageSchedule
	currentUsageSchedule := cfg.GetUsageSchedule()
	cfg.(*UsageReportingConfiguration).UsageSchedule = "*/1511 * * * *"
	err = validateUsageReporting(cfg)
	assert.NotNil(t, err)
	cfg.(*UsageReportingConfiguration).UsageSchedule = "0,15,30,45,55 * * * *"
	err = validateUsageReporting(cfg)
	assert.NotNil(t, err)
	cfg.(*UsageReportingConfiguration).UsageSchedule = currentUsageSchedule

	// QA UsageSchedule override
	os.Setenv(qaUsageReportingUsageScheduleEnvVar, "*/1 * * * *")
	cfg.(*UsageReportingConfiguration).UsageSchedule = "*/1 * * * *"
	err = validateUsageReporting(cfg)
	assert.Nil(t, err)

	// offline settings, valid
	cfg.(*UsageReportingConfiguration).Offline = true
	err = validateUsageReporting(cfg)
	assert.Nil(t, err)

	// invalid Schedule
	currentSchedule := cfg.GetSchedule()
	cfg.(*UsageReportingConfiguration).Schedule = "*/1511 * * * *"
	err = validateUsageReporting(cfg)
	assert.NotNil(t, err)
	cfg.(*UsageReportingConfiguration).Schedule = currentSchedule

	// QA Schedule override
	os.Setenv(qaUsageReportingScheduleEnvVar, "*/1 * * * *")
	cfg.(*UsageReportingConfiguration).Schedule = "*/1 * * * *"
	err = validateUsageReporting(cfg)
	assert.Nil(t, err)

	// QA Report Schedule override
	os.Setenv(qaUsageReportingOfflineScheduleEnvVar, "*/5 * * * *")
	cfg.(*UsageReportingConfiguration).reportSchedule = "*/5 * * * *"
	err = validateUsageReporting(cfg)
	assert.Nil(t, err)
}

func TestNewUsageReporting(t *testing.T) {
	cfg := NewUsageReporting("https://platform.axway.com")
	expected := defaultExpected
	expected.url = "https://platform.axway.com"
	assert.NotNil(t, cfg)
	validateconfig(t, expected, cfg)
}
