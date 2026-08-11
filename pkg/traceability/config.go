package traceability

import (
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/Axway/agent-sdk/pkg/traceability/redaction"
	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// legacy lumberjack/ingestion hostname prefixes no longer supported now that
// single-entry routes HTTPS via phoenix.* hostnames (AOPS-4119)
var removedIngestionHostPrefixes = []string{"ingestion.", "ingestion-http.", "ingestion-lumberjack."}

// using the same env vars agents already use, e.g. pathHost -> TRACEABILITY_HOST.
const (
	pathHost              = "traceability.host"
	pathProtocol          = "traceability.protocol"
	pathPort              = "traceability.port" // deprecated, presence-only check
	pathLoadBalance       = "traceability.loadbalance"
	pathSlowStart         = "traceability.slowstart"
	pathBulkMaxSize       = "traceability.bulkmaxsize"
	pathClientTimeout     = "traceability.clienttimeout"
	pathTTL               = "traceability.ttl"
	pathPipelining        = "traceability.pipelining"
	pathCompressionLevel  = "traceability.compressionlevel"
	pathMaxRetries        = "traceability.maxretries"
	pathSSLVerification   = "traceability.ssl.verificationmode"
	pathSSLCipherSuites   = "traceability.ssl.ciphersuites"
	pathProxyURL          = "traceability.proxyurl"
	pathProxyLocalResolve = "traceability.proxyuselocalresolver"
	pathBackoffInit       = "traceability.backoff.init"
	pathBackoffMax        = "traceability.backoff.max"
	pathEscapeHTML        = "traceability.escapehtml"
	pathExceptionList     = "traceability.exception.list"
	pathRedactionMasking  = "traceability.redaction.masking.characters"
	pathSamplingPercent   = "traceability.sampling.percentage"
	pathSamplingPerAPI    = "traceability.sampling.per.api"
	pathSamplingPerSub    = "traceability.sampling.per.subscription"
	pathSamplingOnlyErr   = "traceability.sampling.onlyerrors"
)

// redaction lists still come from one env var each, holding a blob like [{"keyMatch":".*"}]. Same
// as before. Reason : mulesoft-agents relies on this exact format, so it's kept as-is.
const (
	envRedactionPathShow               = "TRACEABILITY_REDACTION_PATH_SHOW"
	envRedactionQueryArgShow           = "TRACEABILITY_REDACTION_QUERYARGUMENT_SHOW"
	envRedactionQueryArgSanitize       = "TRACEABILITY_REDACTION_QUERYARGUMENT_SANITIZE"
	envRedactionRequestHeaderShow      = "TRACEABILITY_REDACTION_REQUESTHEADER_SHOW"
	envRedactionRequestHeaderSanitize  = "TRACEABILITY_REDACTION_REQUESTHEADER_SANITIZE"
	envRedactionResponseHeaderShow     = "TRACEABILITY_REDACTION_RESPONSEHEADER_SHOW"
	envRedactionResponseHeaderSanitize = "TRACEABILITY_REDACTION_RESPONSEHEADER_SANITIZE"
	envRedactionJMSPropertiesShow      = "TRACEABILITY_REDACTION_JMSPROPERTIES_SHOW"
	envRedactionJMSPropertiesSanitize  = "TRACEABILITY_REDACTION_JMSPROPERTIES_SANITIZE"
)

// Config -
type Config struct {
	LoadBalance       bool
	BulkMaxSize       int
	SlowStart         bool
	Timeout           time.Duration
	TTL               time.Duration
	Pipelining        int
	CompressionLevel  int
	MaxRetries        int
	TLS               TLSConfig
	Proxy             ProxyConfig
	Backoff           Backoff
	EscapeHTML        bool
	Protocol          string
	Hosts             []string
	Redaction         redaction.Config
	Sampling          sampling.Sampling
	APIExceptionsList []string
}

// ProxyConfig holds the configuration information required to proxy
// connections through a SOCKS5 proxy server.
type ProxyConfig struct {
	// URL of the SOCKS proxy. Scheme must be socks5. Username and password can be
	// embedded in the URL.
	URL string

	// Resolve names locally instead of on the SOCKS server.
	LocalResolve bool
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
		Protocol:   "https",
		Redaction:  redaction.DefaultConfig(),
		Sampling:   sampling.DefaultConfig(),
	}
}

// AddConfigProperties sets up all the traceability env vars. Call this once before ParseConfig. It's not automatic, since discovery-only agents don't need it.
func AddConfigProperties(props properties.Properties) {
	def := DefaultConfig()

	props.AddStringSliceProperty(pathHost, def.Hosts, "Comma separated list of traceability hosts to publish to")
	props.AddStringProperty(pathProtocol, def.Protocol, "Protocol used to publish traceability events")
	props.AddStringProperty(pathPort, "", "Deprecated, use "+pathHost)
	props.AddBoolProperty(pathLoadBalance, def.LoadBalance, "Enables round robin load balancing across traceability hosts")
	props.AddBoolProperty(pathSlowStart, def.SlowStart, "Enables slow start for the traceability client")
	props.AddIntProperty(pathBulkMaxSize, def.BulkMaxSize, "Maximum number of events published in a single traceability request")
	props.AddDurationProperty(pathClientTimeout, def.Timeout, "Traceability client timeout")
	props.AddDurationProperty(pathTTL, def.TTL, "Traceability client connection TTL", properties.WithLowerLimit(0))
	props.AddIntProperty(pathPipelining, def.Pipelining, "Traceability client pipelining", properties.WithLowerLimitInt(0))
	props.AddIntProperty(pathCompressionLevel, def.CompressionLevel, "Traceability client compression level",
		properties.WithLowerLimitInt(0), properties.WithUpperLimitInt(9))
	props.AddIntProperty(pathMaxRetries, def.MaxRetries, "Maximum number of retries for a failed traceability request",
		properties.WithLowerLimitInt(-1))
	props.AddStringProperty(pathSSLVerification, "", "TLS verification mode for the traceability client")
	props.AddStringSliceProperty(pathSSLCipherSuites, []string{}, "Cipher suites allowed for the traceability client")
	props.AddStringProperty(pathProxyURL, def.Proxy.URL, "SOCKS5 proxy URL for the traceability client")
	props.AddBoolProperty(pathProxyLocalResolve, def.Proxy.LocalResolve, "Resolve names locally instead of on the SOCKS proxy server")
	props.AddDurationProperty(pathBackoffInit, def.Backoff.Init, "Initial backoff duration for a failed traceability request", properties.WithLowerLimit(0))
	props.AddDurationProperty(pathBackoffMax, def.Backoff.Max, "Maximum backoff duration for a failed traceability request")
	props.AddBoolProperty(pathEscapeHTML, def.EscapeHTML, "Escapes HTML characters in traceability events")
	props.AddStringSliceProperty(pathExceptionList, def.APIExceptionsList, "APIs excluded from traceability logging")

	props.AddStringProperty(pathRedactionMasking, def.Redaction.MaskingCharacters, "Characters used to mask sanitized values")

	props.AddStringProperty(pathSamplingPercent, strconv.FormatFloat(def.Sampling.Percentage, 'f', -1, 64),
		"Percentage of transactions to sample")
	props.AddBoolProperty(pathSamplingPerAPI, def.Sampling.PerAPI, "Applies sampling per API")
	props.AddBoolProperty(pathSamplingPerSub, def.Sampling.PerSub, "Applies sampling per subscription")
	props.AddBoolProperty(pathSamplingOnlyErr, def.Sampling.OnlyErrors, "Only samples transactions that resulted in an error")
}

// ParseConfig reads traceability config from env vars. AddConfigProperties must be called first.
func ParseConfig(props properties.Properties) (*Config, error) {
	outputConfig = DefaultConfig()

	if props.StringPropertyValue(pathPort) != "" {
		log.Warn("output.traceability.port is no longer supported; use output.traceability.hosts")
	}

	outputConfig.Hosts = props.StringSlicePropertyValue(pathHost)
	outputConfig.Protocol = props.StringPropertyValue(pathProtocol)
	outputConfig.LoadBalance = props.BoolPropertyValue(pathLoadBalance)
	outputConfig.SlowStart = props.BoolPropertyValue(pathSlowStart)
	outputConfig.BulkMaxSize = props.IntPropertyValue(pathBulkMaxSize)
	outputConfig.Timeout = props.DurationPropertyValue(pathClientTimeout)
	outputConfig.TTL = props.DurationPropertyValue(pathTTL)
	outputConfig.Pipelining = props.IntPropertyValue(pathPipelining)
	outputConfig.CompressionLevel = props.IntPropertyValue(pathCompressionLevel)
	outputConfig.MaxRetries = props.IntPropertyValue(pathMaxRetries)
	outputConfig.EscapeHTML = props.BoolPropertyValue(pathEscapeHTML)
	outputConfig.APIExceptionsList = props.StringSlicePropertyValue(pathExceptionList)

	outputConfig.TLS.VerificationMode = props.StringPropertyValue(pathSSLVerification)
	outputConfig.TLS.CipherSuites = props.StringSlicePropertyValue(pathSSLCipherSuites)

	outputConfig.Proxy.URL = props.StringPropertyValue(pathProxyURL)
	outputConfig.Proxy.LocalResolve = props.BoolPropertyValue(pathProxyLocalResolve)

	outputConfig.Backoff.Init = props.DurationPropertyValue(pathBackoffInit)
	outputConfig.Backoff.Max = props.DurationPropertyValue(pathBackoffMax)

	outputConfig.Redaction.MaskingCharacters = props.StringPropertyValue(pathRedactionMasking)
	outputConfig.Redaction.Path.Allowed = parseShowList(envRedactionPathShow)
	outputConfig.Redaction.Args.Allowed = parseShowList(envRedactionQueryArgShow)
	outputConfig.Redaction.Args.Sanitize = parseSanitizeList(envRedactionQueryArgSanitize)
	outputConfig.Redaction.RequestHeaders.Allowed = parseShowList(envRedactionRequestHeaderShow)
	outputConfig.Redaction.RequestHeaders.Sanitize = parseSanitizeList(envRedactionRequestHeaderSanitize)
	outputConfig.Redaction.ResponseHeaders.Allowed = parseShowList(envRedactionResponseHeaderShow)
	outputConfig.Redaction.ResponseHeaders.Sanitize = parseSanitizeList(envRedactionResponseHeaderSanitize)
	outputConfig.Redaction.JMSProperties.Allowed = parseShowList(envRedactionJMSPropertiesShow)
	outputConfig.Redaction.JMSProperties.Sanitize = parseSanitizeList(envRedactionJMSPropertiesSanitize)

	if percentage, err := strconv.ParseFloat(props.StringPropertyValue(pathSamplingPercent), 64); err == nil {
		outputConfig.Sampling.Percentage = percentage
	}
	outputConfig.Sampling.PerAPI = props.BoolPropertyValue(pathSamplingPerAPI)
	outputConfig.Sampling.PerSub = props.BoolPropertyValue(pathSamplingPerSub)
	outputConfig.Sampling.OnlyErrors = props.BoolPropertyValue(pathSamplingOnlyErr)

	if agent.GetCentralConfig().GetTraceabilityHost() != "" && len(outputConfig.Hosts) == 0 {
		outputConfig.Protocol = agent.GetCentralConfig().GetTraceabilityProtocol()
		outputConfig.Hosts = []string{agent.GetCentralConfig().GetTraceabilityHost()}
	}

	// Setup the redaction regular expressions
	redaction.SetupGlobalRedaction(outputConfig.Redaction)

	if agent.GetCentralConfig() != nil && agent.GetCentralConfig().GetErrorSamplingEnabled() {
		log.Trace("automatic error sampling has been enabled")
		outputConfig.Sampling.ErrorSamplingEnabled = agent.GetCentralConfig().GetErrorSamplingEnabled()
	}

	// Setup the sampling config, if central config can not be found assume online mode
	var err error
	if agent.GetCentralConfig() != nil && agent.GetCentralConfig().GetUsageReportingConfig() != nil {
		err = sampling.SetupSampling(outputConfig.Sampling, agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode(), agent.GetCentralConfig().GetAPICDeployment(), sampling.WithCacheAccess(agent.GetCacheManager()))
	} else {
		err = sampling.SetupSampling(outputConfig.Sampling, false, agent.GetCentralConfig().GetAPICDeployment(), sampling.WithCacheAccess(agent.GetCacheManager()))
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
	exceptionValue, err := setUpAPIExceptionList(outputConfig.APIExceptionsList)
	if err != nil {
		err = ErrInvalidRegex.FormatError("apiExceptionValue", exceptionValue, err)
		log.Error(err)
	}

	return outputConfig, nil
}

// for Mulesoft after removing go ucfg
var redactionListSpacing = regexp.MustCompile(`(keyMatch|valueMatch):(\S)`)

func normalizeRedactionListValue(val string) string {
	return redactionListSpacing.ReplaceAllString(val, "$1: $2")
}

func parseShowList(envVar string) []redaction.Show {
	shows := []redaction.Show{}
	val := strings.TrimSpace(os.Getenv(envVar))
	if val == "" {
		return shows
	}
	if err := yaml.Unmarshal([]byte(normalizeRedactionListValue(val)), &shows); err != nil {
		log.Warnf("could not parse %s, ignoring: %s", envVar, err.Error())
		return []redaction.Show{}
	}
	return shows
}

func parseSanitizeList(envVar string) []redaction.Sanitize {
	sanitize := []redaction.Sanitize{}
	val := strings.TrimSpace(os.Getenv(envVar))
	if val == "" {
		return sanitize
	}
	if err := yaml.Unmarshal([]byte(normalizeRedactionListValue(val)), &sanitize); err != nil {
		log.Warnf("could not parse %s, ignoring: %s", envVar, err.Error())
		return []redaction.Sanitize{}
	}
	return sanitize
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

// ValidateCfg - validates the config does not use the removed tcp/lumberjack transport
func (c *Config) ValidateCfg() error {
	if c.Protocol == "tcp" {
		return ErrTCPProtocolRemoved
	}
	for _, host := range c.Hosts {
		if err := validateHost(host); err != nil {
			return err
		}
	}
	return nil
}

func validateHost(host string) error {
	h, p, err := net.SplitHostPort(host)
	if err != nil {
		return nil
	}
	if p == tcpPort {
		return ErrPort5044Removed.FormatError(host)
	}
	for _, prefix := range removedIngestionHostPrefixes {
		if strings.HasPrefix(h, prefix) {
			return ErrIngestionHostRemoved.FormatError(host)
		}
	}
	return nil
}
