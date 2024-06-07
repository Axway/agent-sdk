package traceability

import (
	"net/url"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/traceability/redaction"
	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
)

// Config -
type HostConfig struct {
	Protocol string   `config:"protocol"`
	Hosts    []string `config:"hosts"`
}

// Config -
type Config struct {
	Index             string            `config:"index"`
	LoadBalance       bool              `config:"loadbalance"`
	BulkMaxSize       int               `config:"bulk_max_size"`
	SlowStart         bool              `config:"slow_start"`
	Timeout           time.Duration     `config:"client_timeout"    validate:"min=0"`
	TTL               time.Duration     `config:"ttl"               validate:"min=0"`
	Pipelining        int               `config:"pipelining"        validate:"min=0"`
	CompressionLevel  int               `config:"compression_level" validate:"min=0, max=9"`
	MaxRetries        int               `config:"max_retries"       validate:"min=-1"`
	TLS               *tlscommon.Config `config:"ssl"`
	Proxy             ProxyConfig       `config:",inline"`
	Backoff           Backoff           `config:"backoff"`
	EscapeHTML        bool              `config:"escape_html"`
	Protocol          string            `config:"protocol"`
	Hosts             []string          `config:"hosts"`
	Redaction         redaction.Config  `config:"redaction" yaml:"redaction"`
	Sampling          sampling.Sampling `config:"sampling" yaml:"sampling"`
	APIExceptionsList []string          `config:"apiExceptionsList"`
}

// ProxyConfig holds the configuration information required to proxy
// connections through a SOCKS5 proxy server.
type ProxyConfig struct {
	// URL of the SOCKS proxy. Scheme must be socks5. Username and password can be
	// embedded in the URL.
	URL string `config:"proxy_url"`

	// Resolve names locally instead of on the SOCKS server.
	LocalResolve bool `config:"proxy_use_local_resolver"`
}

// Backoff -
type Backoff struct {
	Init time.Duration
	Max  time.Duration
}

var outputConfig *Config

// DefaultConfig -
func DefaultConfig() *Config {
	return &Config{
		LoadBalance:      false,
		Pipelining:       0,
		BulkMaxSize:      512,
		SlowStart:        false,
		CompressionLevel: 3,
		Timeout:          60 * time.Second,
		MaxRetries:       3,
		TTL:              0 * time.Second,
		Backoff: Backoff{
			Init: 1 * time.Second,
			Max:  60 * time.Second,
		},
		EscapeHTML: false,
		Protocol:   "tcp",
		Redaction:  redaction.DefaultConfig(),
		Sampling:   sampling.DefaultConfig(),
	}
}

func readConfig(cfg *common.Config, info beat.Info) (*Config, error) {
	outputConfig = DefaultConfig()

	err := cfgwarn.CheckRemoved6xSettings(cfg, "port")
	if err != nil {
		return nil, err
	}

	if err := cfg.Unpack(outputConfig); err != nil {
		return nil, err
	}

	if len(outputConfig.Hosts) == 1 && agent.GetCentralConfig().GetTraceabilityHost() != "" {
		outputConfig.Protocol = "tcp"
	}

	if outputConfig.Index == "" {
		outputConfig.Index = info.IndexPrefix
	}

	// Setup the redaction regular expressions
	redaction.SetupGlobalRedaction(outputConfig.Redaction)

	// Setup the sampling config, if central config can not be found assume online mode
	if agent.GetCentralConfig() != nil && agent.GetCentralConfig().GetUsageReportingConfig() != nil {
		err = sampling.SetupSampling(outputConfig.Sampling, agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode())
	} else {
		err = sampling.SetupSampling(outputConfig.Sampling, false)
	}

	if err != nil {
		log.Warn(err.Error())
	}

	// Force piplining to 0
	if outputConfig.Pipelining > 0 {
		log.Warn("Pipelining is not supported by Amplify Visibility yet, forcing to synchronous")
		outputConfig.Pipelining = 0
	}

	// if set, check for valid proxyURL
	if outputConfig.Proxy.URL != "" {
		if _, err := url.ParseRequestURI(outputConfig.Proxy.URL); err != nil {
			return nil, ErrInvalidConfig.FormatError("traceability.proxyURL")
		}
	}

	// set up the api exceptions list for logging events
	exception, err := setUpAPIExceptionList(outputConfig.APIExceptionsList)
	if err != nil {
		err = ErrInvalidRegex.FormatError("apiExceptionValue", exception, err)
		log.Error(err)
	}

	return outputConfig, nil
}

// IsHTTPTransport - Returns true if the protocol is set to http/https
func IsHTTPTransport() bool {
	if outputConfig == nil {
		return false
	}
	return (outputConfig.Protocol == "https" || outputConfig.Protocol == "http")
}

// GetMaxRetries - Returns the max retries configured for transport
func GetMaxRetries() int {
	if outputConfig == nil {
		return 3
	}
	return outputConfig.MaxRetries
}
