package sampling

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/publisher"
)

// Global Agent samples
var agentSamples *sample

// Sampling - configures the sampling of events the agent sends to Amplify
type Sampling struct {
	Percentage      int  `config:"percentage"    validate:"min=0, max=100"`
	PerAPI          bool `config:"per_api"`
	ReportAllErrors bool `config:"reportAllErrors" yaml:"reportAllErrors"`
}

//DefaultConfig - returns a default sampling config where all transactions are sent
func DefaultConfig() Sampling {
	return Sampling{
		Percentage:      defaultSamplingRate,
		PerAPI:          true,
		ReportAllErrors: true,
	}
}

// GetGlobalSamplingPercentage -
func GetGlobalSamplingPercentage() (int, error) {
	return agentSamples.config.Percentage, nil
}

// SetupSampling - set up the global sampling for use by traceability
func SetupSampling(cfg Sampling, offlineMode bool) error {
	if offlineMode {
		// In offline mode sampling is always 0
		cfg.Percentage = 0
	}

	// Validate the config to make sure it is not out of bounds
	if cfg.Percentage < 0 || cfg.Percentage > countMax {
		return fmt.Errorf("sampling percentage must be between 0 and 100")
	}
	agentSamples = &sample{
		config:        cfg,
		currentCounts: make(map[string]int),
		counterLock:   sync.Mutex{},
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
