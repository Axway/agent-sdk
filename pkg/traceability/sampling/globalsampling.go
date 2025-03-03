package sampling

import (
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/shopspring/decimal"
)

// Global Agent samples
var agentSamples *sample

const (
	qaSamplingPercentageEnvVar = "QA_TRACEABILITY_SAMPLING_PERCENTAGE"
)

// Sampling - configures the sampling of events the agent sends to Amplify
type Sampling struct {
	Percentage      float64 `config:"percentage"`
	PerAPI          bool    `config:"per_api"`
	PerSub          bool    `config:"per_subscription"`
	OnlyErrors      bool    `config:"onlyErrors" yaml:"onlyErrors"`
	countMax        int
	shouldSampleMax int
}

// DefaultConfig - returns a default sampling config where all transactions are sent
func DefaultConfig() Sampling {
	return Sampling{
		Percentage:      defaultSamplingRate,
		PerAPI:          true,
		PerSub:          true,
		OnlyErrors:      false,
		countMax:        countMax,
		shouldSampleMax: defaultSamplingRate,
	}
}

// GetGlobalSamplingPercentage -
func GetGlobalSamplingPercentage() (float64, error) {
	return agentSamples.config.Percentage, nil
}

// GetGlobalSampling -
func GetGlobalSampling() *sample {
	if agentSamples == nil {
		agentSamples = &sample{
			currentCounts:      make(map[string]int),
			samplingLock:       sync.Mutex{},
			counterResetPeriod: time.Minute,
			counterResetStopCh: make(chan struct{}),
		}
	}
	return agentSamples
}

func getSamplingPercentageConfig(percentage float64, apicDeployment string) (float64, error) {
	maxAllowedSampling := float64(maximumSamplingRate)
	if !strings.HasPrefix(apicDeployment, "prod") {
		if val := os.Getenv(qaSamplingPercentageEnvVar); val != "" {
			if qaSamplingPercentage, err := strconv.ParseFloat(val, 64); err == nil {
				log.Tracef("Using %s (%s) rather than the default (%d) for non-production", qaSamplingPercentageEnvVar, val, defaultSamplingRate)
				percentage = qaSamplingPercentage
				maxAllowedSampling = 100
			} else {
				log.Tracef("Could not use %s (%s) it is not a proper sampling percentage", qaSamplingPercentageEnvVar, val)
			}
		}
	}

	// Validate the config to make sure it is not out of bounds
	if percentage < 0 || percentage > maxAllowedSampling {
		return defaultSamplingRate, ErrSamplingCfg.FormatError(maximumSamplingRate, defaultSamplingRate)
	}

	return percentage, nil
}

// SetupSampling - set up the global sampling for use by traceability
func SetupSampling(cfg Sampling, offlineMode bool, apicDeployment string) error {
	var err error

	if offlineMode {
		cfg = Sampling{
			Percentage: 0,
			PerAPI:     false,
			PerSub:     false,
			OnlyErrors: false,
		}
	} else {
		cfg.Percentage, err = getSamplingPercentageConfig(cfg.Percentage, apicDeployment)
		cfg.countMax = int(100 * math.Pow(10, float64(numberOfDecimals(cfg.Percentage))))
		cfg.shouldSampleMax = int(float64(cfg.countMax) * cfg.Percentage / 100)
	}

	if agentSamples == nil {
		agentSamples = &sample{
			config:             cfg,
			currentCounts:      make(map[string]int),
			samplingLock:       sync.Mutex{},
			counterResetPeriod: time.Minute,
			counterResetStopCh: make(chan struct{}),
		}
	} else {
		agentSamples.config = cfg
	}

	if err != nil {
		return err
	}
	return nil
}

// ShouldSampleTransaction - receives the transaction details and returns true to sample it false to not
func ShouldSampleTransaction(details TransactionDetails) (bool, error) {
	if agentSamples == nil {
		return false, ErrGlobalSamplingCfg
	}
	return agentSamples.ShouldSampleTransaction(details), nil
}

// FilterEvents - returns an array of events that are part of the sample
func FilterEvents(events []publisher.Event) ([]publisher.Event, error) {
	if agentSamples == nil {
		return events, ErrGlobalSamplingCfg
	}
	return agentSamples.FilterEvents(events), nil
}

func numberOfDecimals(v float64) int {
	dec := decimal.NewFromFloat(v)
	x := dec.Exponent()
	// Exponent returns positive values if number is a multiple of 10
	if x > 0 {
		return 0
	}
	// and negative if it contains non-zero decimals
	return int(x) * (-1)
}
