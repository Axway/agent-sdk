package sampling

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/publisher"
)

// Global Agent samples
var agentSamples *sample

// Sampling - configures the sampling of events the agent sends to Amplify
type Sampling struct {
	Percentage      int  `config:"percentage"`
	PerAPI          bool `config:"per_api"`
	PerSub          bool `config:"per_subscription"`
	ReportAllErrors bool `config:"reportAllErrors" yaml:"reportAllErrors"`
}

// DefaultConfig - returns a default sampling config where all transactions are sent
func DefaultConfig() Sampling {
	return Sampling{
		Percentage:      defaultSamplingRate,
		PerAPI:          true,
		PerSub:          true,
		ReportAllErrors: true,
	}
}

// GetGlobalSamplingPercentage -
func GetGlobalSamplingPercentage() (int, error) {
	return agentSamples.config.Percentage, nil
}

// SetupSampling - set up the global sampling for use by traceability
func SetupSampling(cfg Sampling, offlineMode bool) error {
	invalidSampling := false
	if offlineMode {
		// In offline mode sampling is always 0
		cfg.Percentage = 0
	}

	// Validate the config to make sure it is not out of bounds
	if cfg.Percentage < 0 || cfg.Percentage > maximumSamplingRate {
		invalidSampling = true
		cfg.Percentage = defaultSamplingRate
	}

	agentSamples = &sample{
		config:        cfg,
		currentCounts: make(map[string]int),
		counterLock:   sync.Mutex{},
	}
	if invalidSampling {
		return ErrSamplingCfg.FormatError(maximumSamplingRate, defaultSamplingRate)
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
