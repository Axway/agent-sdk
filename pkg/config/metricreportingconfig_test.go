package config

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/Axway/agent-sdk/pkg/util/exception"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func validateMetricReporting(cfg MetricReportingConfig) (err error) {
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

type expectedMetricConfig struct {
	publish     bool
	schedule    string
	granularity int
	qaVars      bool
}

var defaultMetricConfigExpected = expectedMetricConfig{
	publish:     true,
	schedule:    "@hourly",
	granularity: int(time.Hour) * 1,
	qaVars:      false,
}

func validateMetricConfig(t *testing.T, expVals expectedMetricConfig, cfg MetricReportingConfig) {
	assert.Equal(t, expVals.publish, cfg.CanPublish())
	assert.Equal(t, expVals.schedule, cfg.GetSchedule())
	assert.Equal(t, expVals.qaVars, cfg.UsingQAVars())
}

func TestMetricReportingConfigEnvVarMigration(t *testing.T) {
	rootCmd := &cobra.Command{
		Use: "test",
	}

	props := properties.NewProperties(rootCmd)
	AddMetricReportingProperties(props)

	// Test Interval old env vars
	os.Setenv(oldUsageReportingIntervalEnvVar, "30m")
	expected := defaultMetricConfigExpected
	expected.schedule = "*/30 * * * *"
	expected.granularity = int((30 * time.Minute).Milliseconds())

	cfg := ParseMetricReportingConfig(props)
	assert.NotNil(t, cfg)
	err := validateMetricReporting(cfg)
	assert.Nil(t, err)
	validateMetricConfig(t, expected, cfg)

	// Test Interval in hour
	os.Setenv(oldUsageReportingIntervalEnvVar, "2h")
	expected.schedule = fmt.Sprintf("%d 2 * * *", time.Now().Minute())
	expected.granularity = int((2 * time.Hour).Milliseconds())

	cfg = ParseMetricReportingConfig(props)
	assert.NotNil(t, cfg)
	err = validateMetricReporting(cfg)

	assert.Nil(t, err)
	validateMetricConfig(t, expected, cfg)

	// Test Interval new env vars
	os.Setenv(newMetricReportingScheduleEnvVar, defaultExpected.schedule)

	expected = defaultMetricConfigExpected
	cfg = ParseMetricReportingConfig(props)
	assert.NotNil(t, cfg)
	err = validateMetricReporting(cfg)
	assert.Nil(t, err)
	validateMetricConfig(t, expected, cfg)

	// Test Publish old env vars
	os.Setenv(oldUsageReportingPublishMetricEnvVar, "false")
	expected = defaultMetricConfigExpected
	expected.publish = false

	cfg = ParseMetricReportingConfig(props)
	assert.NotNil(t, cfg)
	err = validateMetricReporting(cfg)
	assert.Nil(t, err)
	validateMetricConfig(t, expected, cfg)

	// Test Publish new env vars
	os.Setenv(newMetricReportingPublishEnvVar, strconv.FormatBool(defaultExpected.publish))

	expected = defaultMetricConfigExpected
	cfg = ParseMetricReportingConfig(props)
	assert.NotNil(t, cfg)
	err = validateMetricReporting(cfg)
	assert.Nil(t, err)
	validateMetricConfig(t, expected, cfg)
}

func TestMetricReportingConfigProperties(t *testing.T) {
	rootCmd := &cobra.Command{
		Use: "test",
	}

	props := properties.NewProperties(rootCmd)

	// Test default config
	AddMetricReportingProperties(props)

	cfg := ParseMetricReportingConfig(props)
	assert.NotNil(t, cfg)

	err := validateMetricReporting(cfg)
	assert.Nil(t, err)

	validateMetricConfig(t, defaultMetricConfigExpected, cfg)

	// invalid schedule
	currentSchedule := cfg.GetSchedule()
	cfg.(*MetricReportingConfiguration).Schedule = "*/1511 * * * *"
	err = validateMetricReporting(cfg)
	assert.NotNil(t, err)
	cfg.(*MetricReportingConfiguration).Schedule = currentSchedule

	// QA schedule override
	os.Setenv(qaMetricReportingScheduleEnvVar, "*/1 * * * *")
	cfg.(*MetricReportingConfiguration).Schedule = ""
	err = validateMetricReporting(cfg)
	assert.Nil(t, err)
}

func TestNewMetricReporting(t *testing.T) {
	cfg := NewMetricReporting()
	assert.NotNil(t, cfg)
	validateMetricConfig(t, defaultMetricConfigExpected, cfg)
}
