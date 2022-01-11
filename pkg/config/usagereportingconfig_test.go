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
			cfg.validate()
		},
		Catch: func(e error) {
			err = e
		},
	}.Do()
	return
}

type expected struct {
	url            string
	publish        bool
	metric         bool
	interval       time.Duration
	offline        bool
	schedule       string
	reportSchedule string
	granularity    int
	qaVars         bool
}

var defaultExpected = expected{
	url:            "https://lighthouse.admin.axway.com",
	publish:        true,
	metric:         true,
	interval:       15 * time.Minute,
	offline:        false,
	schedule:       "@hourly",
	reportSchedule: "@monthly",
	granularity:    900000,
	qaVars:         false,
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

	// Test URL old env vars
	os.Setenv(oldUsageReportingURLEnvVar, "http://lighthouse-old.com")
	expected := defaultExpected
	expected.url = "http://lighthouse-old.com"

	cfg := ParseUsageReportingConfig(props)
	assert.NotNil(t, cfg)
	err := validateUsageReporting(cfg)
	assert.Nil(t, err)

	validateconfig(t, expected, cfg)

	// Test URL new env vars
	os.Setenv(newUsageReportingURLEnvVar, defaultExpected.url)

	expected = defaultExpected
	cfg = ParseUsageReportingConfig(props)
	assert.NotNil(t, cfg)
	err = validateUsageReporting(cfg)
	assert.Nil(t, err)
	validateconfig(t, expected, cfg)

	// Test Interval old env vars
	os.Setenv(oldUsageReportingIntervalEnvVar, "30m")
	expected = defaultExpected
	expected.interval = 30 * time.Minute

	cfg = ParseUsageReportingConfig(props)
	assert.NotNil(t, cfg)
	err = validateUsageReporting(cfg)
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

	// invalid URL
	currentURL := cfg.GetURL()
	cfg.(*UsageReportingConfiguration).URL = "notAURL"
	err = validateUsageReporting(cfg)
	assert.NotNil(t, err)
	cfg.(*UsageReportingConfiguration).URL = currentURL

	// invalid Interval
	currentInterval := cfg.GetInterval()
	cfg.(*UsageReportingConfiguration).Interval = time.Millisecond
	err = validateUsageReporting(cfg)
	assert.NotNil(t, err)
	cfg.(*UsageReportingConfiguration).Interval = currentInterval

	// offline settings, valid
	cfg.(*UsageReportingConfiguration).Offline = true
	err = validateUsageReporting(cfg)
	assert.Nil(t, err)

	// invalid Schedule
	currentSchedule := cfg.GetSchedule()
	cfg.(*UsageReportingConfiguration).Schedule = "*/15 * * * *"
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
	cfg := NewUsageReporting()
	assert.NotNil(t, cfg)
	validateconfig(t, defaultExpected, cfg)
}
