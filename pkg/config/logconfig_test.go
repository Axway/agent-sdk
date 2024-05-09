package config

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

const (
	defLevel          = "info"
	defFormat         = "json"
	defOutput         = "stdout"
	defMaskedVals     = ""
	defPath           = "logs"
	defMaxSize        = 10485760
	defMaxAge         = 0
	defMaxFiles       = 7
	defMetricName     = "metrics.log"
	defMetricMaxSize  = 10485760
	defMetricMaxAge   = 0
	defMetricMaxFiles = 0
	defUsageName      = "usage.log"
	defUsageMaxAge    = 365
)

func TestDefaultLogConfig(t *testing.T) {
	testCases := map[string]struct {
		logName   string
		agentType AgentType
	}{
		"default discovery agent configuration": {
			agentType: DiscoveryAgent,
			logName:   "discovery_agent.log",
		},
		"default traceability agent configuration": {
			agentType: TraceabilityAgent,
			logName:   "traceability_agent.log",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			rootCmd := &cobra.Command{}
			props := properties.NewProperties(rootCmd)
			AddLogConfigProperties(props, tc.logName)
			AddMetricLogConfigProperties(props, tc.agentType)
			AddUsageConfigProperties(props, tc.agentType)

			// Parse config
			_, err := ParseAndSetupLogConfig(props, tc.agentType)
			assert.Nil(t, err, "Expected no error with default values")

			// Validate the default
			assert.Equal(t, defLevel, props.StringPropertyValue(pathLogLevel))
			assert.Equal(t, defFormat, props.StringPropertyValue(pathLogFormat))
			assert.Equal(t, defOutput, props.StringPropertyValue(pathLogOutput))
			assert.Equal(t, defMaskedVals, props.StringPropertyValue(pathLogMaskedValues))
			assert.Equal(t, tc.logName, props.StringPropertyValue(pathLogFileName))
			assert.Equal(t, defPath, props.StringPropertyValue(pathLogFilePath))
			assert.Equal(t, defMaxSize, props.IntPropertyValue(pathLogFileMaxSize))
			assert.Equal(t, defMaxAge, props.IntPropertyValue(pathLogFileMaxAge))
			assert.Equal(t, defMaxFiles, props.IntPropertyValue(pathLogFileMaxBackups))

			if tc.agentType == TraceabilityAgent {
				assert.Equal(t, defMetricName, props.StringPropertyValue(pathLogMetricsFileName))
				assert.Equal(t, defMetricMaxSize, props.IntPropertyValue(pathLogMetricsFileMaxSize))
				assert.Equal(t, defMetricMaxAge, props.IntPropertyValue(pathLogMetricsFileMaxAge))
				assert.Equal(t, defMetricMaxFiles, props.IntPropertyValue(pathLogMetricsFileMaxBackups))

				assert.Equal(t, defUsageName, props.StringPropertyValue(pathLogUsageFileName))
				assert.Equal(t, defUsageMaxAge, props.IntPropertyValue(pathLogUsageFileMaxAge))
			}
		})
	}
}

func TestLogConfigValidations(t *testing.T) {
	testCases := map[string]struct {
		errInfo          string
		agentType        AgentType
		metricsEnabled   bool
		usageEnabled     bool
		level            string
		format           string
		output           string
		maxSize          int
		maxBackups       int
		maxAge           int
		metricMaxSize    int
		metricMaxBackups int
		metricMaxAge     int
		usageMaxAge      int
	}{
		"expect err, bad log level": {
			errInfo: "log.level",
			level:   "debug1",
		},
		"expect err, bad log format": {
			errInfo: "log.format",
			format:  "line1",
		},
		"expect err, bad log output": {
			errInfo: "log.output",
			output:  "unknown",
		},
		"expect err, bad max log size": {
			errInfo: "log.file.rotateeverybytes",
			maxSize: 1,
		},
		"expect err, bad max log backups": {
			errInfo:    "log.file.keepfiles",
			maxBackups: -1,
		},
		"expect err, bad max log age": {
			errInfo: "log.file.cleanbackupsevery",
			maxAge:  -1,
		},
		"success": {
			metricsEnabled: true,
		},
		"expect err, traceability agent, bad metric log size": {
			agentType:      TraceabilityAgent,
			metricsEnabled: true,
			errInfo:        "log.metricfile.rotateeverybytes",
			metricMaxSize:  1,
		},
		"expect err, traceability agent, bad metric log backups": {
			agentType:        TraceabilityAgent,
			metricsEnabled:   true,
			errInfo:          "log.metricfile.keepfiles",
			metricMaxBackups: -1,
		},
		"expect err, traceability agent, bad metric log age": {
			agentType:      TraceabilityAgent,
			metricsEnabled: true,
			errInfo:        "log.metricfile.cleanbackupsevery",
			metricMaxAge:   -1,
		},
		"expect err, traceability agent, bad published transactions log age": {
			agentType:      TraceabilityAgent,
			usageEnabled:   true,
			metricsEnabled: true,
			errInfo:        "log.usagefile.cleanbackupsevery",
			usageMaxAge:    -1,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			if tc.agentType == 0 {
				tc.agentType = DiscoveryAgent
			}
			if tc.level == "" {
				tc.level = defLevel
			}
			if tc.format == "" {
				tc.format = defFormat
			}
			if tc.output == "" {
				tc.output = defOutput
			}
			if tc.maxSize == 0 {
				tc.maxSize = defMaxSize
			}
			if tc.maxBackups == 0 {
				tc.maxBackups = defMaxFiles
			}
			if tc.maxAge == 0 {
				tc.maxAge = defMaxAge
			}
			if tc.metricMaxSize == 0 {
				tc.metricMaxSize = defMetricMaxSize
			}
			if tc.metricMaxBackups == 0 {
				tc.metricMaxBackups = defMetricMaxFiles
			}
			if tc.metricMaxAge == 0 {
				tc.metricMaxAge = defMetricMaxAge
			}
			if tc.usageMaxAge == 0 {
				tc.usageMaxAge = defUsageMaxAge
			}

			log.GlobalLoggerConfig = log.LoggerConfig{}
			props := properties.NewProperties(&cobra.Command{})
			props.AddStringProperty(pathLogLevel, tc.level, "")
			props.AddStringProperty(pathLogFormat, tc.format, "")
			props.AddStringProperty(pathLogOutput, tc.output, "")
			props.AddStringProperty(pathLogFileName, "agent.log", "")
			props.AddStringProperty(pathLogFilePath, defPath, "")
			props.AddIntProperty(pathLogFileMaxSize, tc.maxSize, "")
			props.AddIntProperty(pathLogFileMaxBackups, tc.maxBackups, "")
			props.AddIntProperty(pathLogFileMaxAge, tc.maxAge, "")

			if tc.agentType == TraceabilityAgent && tc.metricsEnabled {
				props.AddBoolProperty(pathLogMetricsFileEnabled, true, "")
				props.AddStringProperty(pathLogMetricsFileName, "metrics.log", "")
				props.AddIntProperty(pathLogMetricsFileMaxSize, tc.metricMaxSize, "")
				props.AddIntProperty(pathLogMetricsFileMaxBackups, tc.metricMaxBackups, "")
				props.AddIntProperty(pathLogMetricsFileMaxAge, tc.metricMaxAge, "")
			}

			if tc.agentType == TraceabilityAgent && tc.usageEnabled {
				props.AddBoolProperty(pathLogUsageFileEnabled, true, "")
				props.AddStringProperty(pathLogUsageFileName, "usage.log", "")
				props.AddIntProperty(pathLogUsageFileMaxAge, tc.usageMaxAge, "")
			}
			_, err := ParseAndSetupLogConfig(props, tc.agentType)
			if tc.errInfo != "" {
				if !assert.NotNil(t, err) {
					return
				}
				assert.Contains(t, err.Error(), tc.errInfo)
				return
			}
			assert.Nil(t, err)
		})
	}
}
