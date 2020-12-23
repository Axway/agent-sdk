package traceability

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
)

// Config -
type Config struct {
	Index            string            `config:"index"`
	LoadBalance      bool              `config:"loadbalance"`
	BulkMaxSize      int               `config:"bulk_max_size"`
	SlowStart        bool              `config:"slow_start"`
	Timeout          time.Duration     `config:"timeout"`
	TTL              time.Duration     `config:"ttl"               validate:"min=0"`
	Pipelining       int               `config:"pipelining"        validate:"min=0"`
	CompressionLevel int               `config:"compression_level" validate:"min=0, max=9"`
	MaxRetries       int               `config:"max_retries"       validate:"min=-1"`
	TLS              *tlscommon.Config `config:"ssl"`
	Proxy            ProxyConfig       `config:",inline"`
	Backoff          Backoff           `config:"backoff"`
	EscapeHTML       bool              `config:"escape_html"`
	Protocol         string            `config:"protocol"`
	Hosts            []string          `config:"hosts"`
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
		Timeout:          30 * time.Second,
		MaxRetries:       3,
		TTL:              0 * time.Second,
		Backoff: Backoff{
			Init: 1 * time.Second,
			Max:  60 * time.Second,
		},
		EscapeHTML: false,
		Protocol:   "tcp",
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

	if outputConfig.Index == "" {
		outputConfig.Index = info.IndexPrefix
	}

	// Force piplining to 0
	if outputConfig.Pipelining > 0 {
		logp.Warn("Pipelining is not supported by AMPLIFY Visibility yet, forcing to synchronous")
		outputConfig.Pipelining = 0
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
