package sampling

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/jobs"
	transactionUtil "github.com/Axway/agent-sdk/pkg/transaction/util"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/shopspring/decimal"
)

// Global Agent samples
var agentSamples *sample

const (
	qaSamplingPercentageEnvVar   = "QA_TRACEABILITY_SAMPLING_PERCENTAGE"
	qaErrorSamplingResetInterval = "QA_TRACEABILITY_SAMPLING_ERRORRESETINTERVAL"
)

type cacheAccess interface {
	GetManagedApplicationByName(name string) *v1.ResourceInstance
	GetAccessRequestsByApp(managedAppName string) []*v1.ResourceInstance
	GetWatchResourceCacheKeys(group, kind string) []string
	GetWatchResourceByKey(key string) *v1.ResourceInstance
	GetAPIServiceWithName(apiName string) *v1.ResourceInstance
}

// Sampling - configures the sampling of events the agent sends to Amplify
type Sampling struct {
	Percentage                 float64 `config:"percentage"`
	PerAPI                     bool    `config:"per_api"`
	PerSub                     bool    `config:"per_subscription"`
	OnlyErrors                 bool    `config:"onlyErrors" yaml:"onlyErrors"`
	ErrorSamplingEnabled       bool
	errorSamplingResetInterval time.Duration `config:"errorResetInterval"`
	countMax                   int
	shouldSampleMax            int
	cacheAccess                cacheAccess
	externalDataLookUp         map[string]string
}

// DefaultConfig - returns a default sampling config where all transactions are sent
func DefaultConfig() Sampling {
	return Sampling{
		Percentage:                 defaultSamplingRate,
		PerAPI:                     true,
		PerSub:                     true,
		OnlyErrors:                 false,
		ErrorSamplingEnabled:       false,
		errorSamplingResetInterval: getErrorSamplingResetIntervalConfig(defaultErrorSamplingResetInterval),
		countMax:                   countMax,
		shouldSampleMax:            defaultSamplingRate,
		externalDataLookUp:         make(map[string]string),
	}
}

// GetGlobalSamplingPercentage -
func GetGlobalSamplingPercentage() (float64, error) {
	return agentSamples.config.Percentage, nil
}

// GetApiAppErrorSampling -
func GetApiAppErrorSampling() map[string]struct{} {
	if agentSamples != nil {
		return agentSamples.apiAppErrorSampling
	}
	return nil
}

// GetGlobalSampling -
func GetGlobalSampling() *sample {
	period := &atomic.Int64{}
	period.Store(int64(time.Minute))
	if agentSamples == nil {
		agentSamples = &sample{
			config:             DefaultConfig(),
			currentCounts:      make(map[string]int),
			samplingLock:       sync.Mutex{},
			counterResetPeriod: period,
			counterResetStopCh: make(chan struct{}),
			endpointsSampling: endpointsSampling{
				enabled:       atomic.Bool{},
				endpointsInfo: make(map[string]bool, 0),
				endpointsLock: sync.Mutex{},
			},
			samplingTime: concurrentTime{
				endTime: time.Time{},
				mu:      sync.RWMutex{},
			},
			apiAppErrorSampling: make(map[string]struct{}),
			logger:              log.NewFieldLogger().WithComponent("agentSamples").WithPackage("sampling"),
		}
	}
	return agentSamples
}

func getExternalAppKeyData() definitions.ExternalAppData {
	if agentSamples == nil {
		sample := GetGlobalSampling()
		return sample.externalAppKeyData
	}
	return agentSamples.externalAppKeyData
}

func SetExternalAppKeyData(key definitions.ExternalAppData) {
	if agentSamples == nil {
		sample := GetGlobalSampling()
		sample.externalAppKeyData = key
	} else {
		agentSamples.externalAppKeyData = key
	}
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

func getErrorSamplingResetIntervalConfig(interval time.Duration) time.Duration {
	if val := os.Getenv(qaErrorSamplingResetInterval); val != "" {
		if qaInterval, err := time.ParseDuration(val); err == nil {
			log.Tracef("Using %s (%s) rather than the default (1h) for non-production", qaErrorSamplingResetInterval, val)
			interval = qaInterval
		} else {
			log.Tracef("Could not use %s (%s) it is not a proper duration", qaErrorSamplingResetInterval, val)
		}

		// Validate the config to make sure it is not out of bounds
		if interval < 10*time.Second {
			return defaultErrorSamplingResetInterval
		}
	}

	return interval
}

// SetupSampling - set up the global sampling for use by traceability
func SetupSampling(cfg Sampling, offlineMode bool, apicDeployment string, cacheAccess cacheAccess) error {
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

	cfg.cacheAccess = cacheAccess

	if agentSamples == nil {
		period := &atomic.Int64{}
		period.Store(int64(time.Minute))
		agentSamples = &sample{
			config:             cfg,
			currentCounts:      make(map[string]int),
			samplingLock:       sync.Mutex{},
			counterResetPeriod: period,
			counterResetStopCh: make(chan struct{}),
			endpointsSampling: endpointsSampling{
				enabled:       atomic.Bool{},
				endpointsInfo: make(map[string]bool, 0),
				endpointsLock: sync.Mutex{},
			},
			samplingTime: concurrentTime{
				endTime: time.Time{},
				mu:      sync.RWMutex{},
			},
			apiAppErrorSampling: make(map[string]struct{}),
			logger:              log.NewFieldLogger().WithComponent("agentSamples").WithPackage("sampling"),
		}
	} else {
		agentSamples.config = cfg
	}

	if cfg.ErrorSamplingEnabled {
		// start api/app error sampling reset job if error sampling is enabled
		resetJob := newAPIAppErrorSamplingResetJob()
		// jobs.RegisterScheduledJobWithName(resetJob, "@hourly", "API/App Error Sampling Reset")
		jobs.RegisterIntervalJobWithName(resetJob, cfg.errorSamplingResetInterval, "API/App Error Sampling Reset") // TODO: change back to scheduled job after testing
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

func RemoveApiAppKey(apiServiceName, managedAppName string) {
	if agentSamples != nil && agentSamples.apiAppErrorSampling != nil {
		externalAPIID := agentSamples.config.getExternalAPIID(apiServiceName)
		externalAppID := agentSamples.config.getExternalAppID(managedAppName, getExternalAppKeyData())
		k := FormatApiAppKey(externalAPIID, externalAppID)

		agentSamples.samplingLock.Lock()
		defer agentSamples.samplingLock.Unlock()
		agentSamples.logger.WithField(apiAppKey, k).Trace("removing api-app key pair")
		delete(agentSamples.apiAppErrorSampling, k)
	}
}

func FormatApiAppKey(apiID, appID string) string {
	formattedSvcName := strings.TrimPrefix(apiID, transactionUtil.SummaryEventProxyIDPrefix)
	formattedAppName := strings.TrimPrefix(appID, transactionUtil.SummaryEventApplicationIDPrefix)
	return fmt.Sprintf("%s - %s", formattedSvcName, formattedAppName)
}

func (s *Sampling) getExternalAppID(appName string, externalAppKey definitions.ExternalAppData) string {
	if val, ok := s.externalDataLookUp[appName]; ok {
		return val
	}

	if s.cacheAccess == nil {
		return ""
	}

	externalAppID := ""
	switch externalAppKey.ResourceType {
	case management.ManagedApplicationGVK().Kind:
		ri := s.cacheAccess.GetManagedApplicationByName(appName)
		managedApp := &management.ManagedApplication{}
		managedApp.FromInstance(ri)
		externalAppID, _ = util.GetAgentDetailsValue(managedApp, externalAppKey.Key)
	case management.AccessRequestGVK().Kind:
		ris := s.cacheAccess.GetAccessRequestsByApp(appName)
		if len(ris) > 0 {
			accReqRI := ris[0]
			accReq := &management.AccessRequest{}
			accReq.FromInstance(accReqRI)
			externalAppID, _ = util.GetAgentDetailsValue(accReq, externalAppKey.Key)
		}
	case management.CredentialGVK().Kind:
		keys := s.cacheAccess.GetWatchResourceCacheKeys(management.CredentialGVK().Group, management.CredentialGVK().Kind)
		for _, key := range keys {
			ri := s.cacheAccess.GetWatchResourceByKey(key)
			credential := &management.Credential{}
			credential.FromInstance(ri)
			if credential.Spec.ManagedApplication == appName {
				externalAppID, _ = util.GetAgentDetailsValue(credential, externalAppKey.Key)
			}
		}
	}

	if externalAppID != "" {
		s.externalDataLookUp[appName] = externalAppID
	}

	return externalAppID
}

func (s *Sampling) getExternalAPIID(apiServiceName string) string {
	if val, ok := s.externalDataLookUp[apiServiceName]; ok {
		return val
	}

	if s.cacheAccess == nil {
		return apiServiceName // Return the input as-is if no cache available
	}

	ri := s.cacheAccess.GetAPIServiceWithName(apiServiceName)
	if ri != nil {
		apiService := &management.APIService{}
		apiService.FromInstance(ri)
		externalAPIID, _ := util.GetAgentDetailsValue(apiService, definitions.AttrExternalAPIID)
		s.externalDataLookUp[apiServiceName] = externalAPIID
		return externalAPIID
	}

	return ""
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
