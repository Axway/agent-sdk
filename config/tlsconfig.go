package config

import (
	"crypto/tls"
	"errors"
	"strings"

	"git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/exception"
)

// TLSCipherSuite - defined type
type TLSCipherSuite uint16

// Taken from https://www.iana.org/assignments/tls-parameters/tls-parameters.xml
var tlsCipherSuites = map[string]TLSCipherSuite{
	// ECDHE-ECDSA
	"ECDHE-ECDSA-AES-128-CBC-SHA":    TLSCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA),
	"ECDHE-ECDSA-AES-128-CBC-SHA256": TLSCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256),
	"ECDHE-ECDSA-AES-128-GCM-SHA256": TLSCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256),
	"ECDHE-ECDSA-AES-256-CBC-SHA":    TLSCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA),
	"ECDHE-ECDSA-AES-256-GCM-SHA384": TLSCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384),
	"ECDHE-ECDSA-CHACHA20-POLY1305":  TLSCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305),
	"ECDHE-ECDSA-RC4-128-SHA":        TLSCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA),

	// ECDHE-RSA
	"ECDHE-RSA-3DES-CBC3-SHA":      TLSCipherSuite(tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA),
	"ECDHE-RSA-AES-128-CBC-SHA":    TLSCipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA),
	"ECDHE-RSA-AES-128-CBC-SHA256": TLSCipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256),
	"ECDHE-RSA-AES-128-GCM-SHA256": TLSCipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256),
	"ECDHE-RSA-AES-256-CBC-SHA":    TLSCipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA),
	"ECDHE-RSA-AES-256-GCM-SHA384": TLSCipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384),
	"ECDHE-RSA-CHACHA20-POLY1305":  TLSCipherSuite(tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305),
	"ECDHE-RSA-RC4-128-SHA":        TLSCipherSuite(tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA),

	// RSA-X
	"RSA-RC4-128-SHA":   TLSCipherSuite(tls.TLS_RSA_WITH_RC4_128_SHA),
	"RSA-3DES-CBC3-SHA": TLSCipherSuite(tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA),

	// RSA-AES
	"RSA-AES-128-CBC-SHA":    TLSCipherSuite(tls.TLS_RSA_WITH_AES_128_CBC_SHA),
	"RSA-AES-128-CBC-SHA256": TLSCipherSuite(tls.TLS_RSA_WITH_AES_128_CBC_SHA256),
	"RSA-AES-128-GCM-SHA256": TLSCipherSuite(tls.TLS_RSA_WITH_AES_128_GCM_SHA256),
	"RSA-AES-256-CBC-SHA":    TLSCipherSuite(tls.TLS_RSA_WITH_AES_256_CBC_SHA),
	"RSA-AES-256-GCM-SHA384": TLSCipherSuite(tls.TLS_RSA_WITH_AES_256_GCM_SHA384),

	// TLS 1.3
	"TLS-AES-128-GCM-SHA256":       TLSCipherSuite(tls.TLS_AES_128_GCM_SHA256),
	"TLS-AES-256-GCM-SHA384":       TLSCipherSuite(tls.TLS_AES_256_GCM_SHA384),
	"TLS-CHACHA20-POLY1305-SHA256": TLSCipherSuite(tls.TLS_CHACHA20_POLY1305_SHA256),
}

// TLSDefaultCipherSuites - list of suites to use by default
func TLSDefaultCipherSuites() string {
	suites := []TLSCipherSuite{
		TLSCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384),
		TLSCipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384),
		TLSCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305),
		TLSCipherSuite(tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305),
		TLSCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256),
		TLSCipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256),
		TLSCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256),
		TLSCipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256),
	}

	values := make([]string, 0)
	for _, v := range suites {
		values = append(values, cipherAsString(v))
	}

	return strings.Join(values, ",")
}

var tlsCipherSuitesInverse = make(map[TLSCipherSuite]string, len(tlsCipherSuites))

func cipherAsString(cs TLSCipherSuite) string {
	if s, ok := tlsCipherSuitesInverse[cs]; ok {
		return s
	}
	return "unknown"
}

func cipherAsValue(cs string) TLSCipherSuite {
	if s, ok := tlsCipherSuites[cs]; ok {
		return s
	}
	return TLSCipherSuite(0) // return bogus value
}

// TLSVersion - define type for version
type TLSVersion uint16

// Define all the possible TLS version.
var tlsVersions = map[string]TLSVersion{
	"TLS1.0": tls.VersionTLS10,
	"TLS1.1": tls.VersionTLS11,
	"TLS1.2": tls.VersionTLS12,
	"TLS1.3": tls.VersionTLS13,
}

var tlsVersionsInverse = make(map[TLSVersion]string, len(tlsVersions))

// TLSVersionAsString - get the version string
func TLSVersionAsString(cs TLSVersion) string {
	if s, ok := tlsVersionsInverse[cs]; ok {
		return s
	}
	return "unknown"
}

// TLSDefaultVersionString - default value
func TLSDefaultVersionString() string {
	return TLSVersionAsString(tls.VersionTLS12)
}

// TLSVersionAsValue - get the version value
func TLSVersionAsValue(cs string) TLSVersion {
	// value of 0 means to use the default. Leave it alone.
	if cs == "0" {
		return TLSVersion(0)
	}
	if s, ok := tlsVersions[cs]; ok {
		return s
	}
	return TLSVersion(1) // return a bogus value for validation checking
}

// Init creates a inverse representation of the values mapping.
func init() {
	for cipherName, i := range tlsCipherSuites {
		tlsCipherSuitesInverse[i] = cipherName
	}
	for versionName, i := range tlsVersions {
		tlsVersionsInverse[i] = versionName
	}
}

// TLSConfig - interface
type TLSConfig interface {
	GetNextProtos() []string
	IsInsecureSkipVerify() bool
	GetCipherSuites() []TLSCipherSuite
	GetMinVersion() TLSVersion
	GetMaxVersion() TLSVersion
	BuildTLSConfig() *tls.Config
	Validate()
}

// TLSConfiguration - A Config structure is used to configure a TLS client or server.
// After one has been passed to a TLS function it must not be modified. A Config may be reused;
// the tls package will also not modify it.
type TLSConfiguration struct {
	// NextProtos is a list of supported application level protocols, in order of preference.
	NextProtos []string

	// InsecureSkipVerify controls whether a client verifies the server's certificate chain and host name.
	// If InsecureSkipVerify is true, TLS accepts any certificate presented by the server and any host
	// name in that certificate. In this mode, TLS is susceptible to man-in-the-middle attacks.
	// This should be used only for testing.
	InsecureSkipVerify bool

	// CipherSuites is a list of supported cipher suites for TLS versions up to TLS 1.2. If CipherSuites
	// is nil, a default list of secure cipher suites is used, with a preference order based on hardware
	// performance. The default cipher suites might change over Go versions. Note that TLS 1.3
	// ciphersuites are not configurable.
	CipherSuites []TLSCipherSuite

	// MinVersion contains the minimum SSL/TLS version that is acceptable. If zero, then TLS 1.0 is taken as the minimum.
	MinVersion TLSVersion

	// MaxVersion contains the maximum SSL/TLS version that is acceptable. If zero, then the maximum
	// version supported by this package is used, which is currently TLS 1.3.
	MaxVersion TLSVersion
}

func newTLSConfig() TLSConfig {
	return &TLSConfiguration{}
}

// BuildTLSConfig takes the TLSConfiguration and transform it into a `tls.Config`.
func (c *TLSConfiguration) BuildTLSConfig() *tls.Config {
	if c == nil {
		// use default TLS settings, if config is empty.
		return &tls.Config{}
	}

	ciphers := make([]uint16, len(c.CipherSuites))
	for i, num := range c.CipherSuites {
		ciphers[i] = uint16(num)
	}
	return &tls.Config{
		MinVersion:         uint16(c.MinVersion),
		MaxVersion:         uint16(c.MaxVersion),
		InsecureSkipVerify: c.InsecureSkipVerify,
		CipherSuites:       ciphers,
	}
}

// NewCipherArray - create an array of TLSCipherSuite
func NewCipherArray(ciphers string) []TLSCipherSuite {
	cipherArray := strings.Split(ciphers, ",")
	result := make([]TLSCipherSuite, 0)
	for _, v := range cipherArray {
		trimmedV := strings.TrimSpace(v)
		result = append(result, cipherAsValue(trimmedV))
	}
	return result
}

// GetNextProtos -
func (c *TLSConfiguration) GetNextProtos() []string {
	return nil
}

// IsInsecureSkipVerify -
func (c *TLSConfiguration) IsInsecureSkipVerify() bool {
	return c.InsecureSkipVerify
}

// GetCipherSuites -
func (c *TLSConfiguration) GetCipherSuites() []TLSCipherSuite {
	return c.CipherSuites
}

// GetMinVersion -
func (c *TLSConfiguration) GetMinVersion() TLSVersion {
	return c.MinVersion
}

// GetMaxVersion -
func (c *TLSConfiguration) GetMaxVersion() TLSVersion {
	return c.MaxVersion
}

// Validate -
func (c *TLSConfiguration) Validate() {
	if c.MinVersion != 0 && !c.isValidMinVersion() {
		exception.Throw(errors.New("Error ssl.minVersion not valid in config"))
	}

	if c.MaxVersion != 0 && !c.isValidMaxVersion() {
		exception.Throw(errors.New("Error ssl.maxVersion not valid in config"))
	}

	if len(c.CipherSuites) != 0 && !c.isValidCiphers() {
		exception.Throw(errors.New("Error ssl.cipherSuites not valid in config"))
	}
}

func (c *TLSConfiguration) isValidMinVersion() bool {
	if c.MinVersion == 0 {
		return true
	}

	_, ok := tlsVersionsInverse[c.MinVersion]
	return ok
}

func (c *TLSConfiguration) isValidMaxVersion() bool {
	if c.MaxVersion == 0 {
		return true
	}

	_, ok := tlsVersionsInverse[c.MaxVersion]
	return ok
}

func (c *TLSConfiguration) isValidCiphers() bool {
	for _, v := range c.CipherSuites {
		if _, ok := tlsCipherSuitesInverse[v]; !ok {
			return false
		}
	}

	return true
}
