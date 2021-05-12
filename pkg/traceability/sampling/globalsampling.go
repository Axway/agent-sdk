package sampling

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/publisher"
)

// Global Agent samples
var agentSamples *sample

// Sampling - configures the sampling of events the agent sends to Amplify
type Sampling struct {
	Percentage int `config:"percentage"    validate:"min=0, max=100"`
}

//DefaultConfig - returns a default sampling config where all transactions are sent
func DefaultConfig() Sampling {
	return Sampling{
		Percentage: 100,
	}
}

// SetupSampling - set up redactionRegex based on the redactionConfig
func SetupSampling(cfg Sampling) error {
	// Validate the config to make sure it is not out of bounds
	if cfg.Percentage < 0 || cfg.Percentage > 100 {
		return fmt.Errorf("sampling percentage must be between 0 and 100")
	}
	agentSamples = &sample{
		config:       cfg,
		currentCount: 0, // counter of events, only up to sample are returned
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
