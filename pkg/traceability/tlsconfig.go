package traceability

import (
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// unsupportedTLSField pairs a config key with whether it was set.
type unsupportedTLSField struct {
	name string
	set  bool
}

// TLSConfig replaces libbeat's tlscommon.Config.
type TLSConfig struct {
	// Enabled is accepted but not acted on: protocol already determines whether TLS
	// applies.
	Enabled          *bool    `config:"enabled"`
	VerificationMode string   `config:"verification_mode"`
	CipherSuites     []string `config:"cipher_suites"`

	SupportedProtocols     []string `config:"supported_protocols"`
	CertificateAuthorities []string `config:"certificate_authorities"`
	CurveTypes             []string `config:"curve_types"`
	Renegotiation          string   `config:"renegotiation"`
	CASha256               []string `config:"ca_sha256"`
	KeyPassphrase          string   `config:"key_passphrase"`
}

// toTLSConfiguration replaces libbeat's tlscommon.LoadTLSConfig.
func (t TLSConfig) toTLSConfiguration() *config.TLSConfiguration {
	tlsCfg := config.NewTLSConfig().(*config.TLSConfiguration)
	tlsCfg.InsecureSkipVerify = t.VerificationMode == "none"

	switch t.VerificationMode {
	case "", "full", "none":
		// supported, nothing to warn about
	case "certificate", "strict":
		log.Warnf("output.traceability.ssl.verification_mode %q is no longer supported; using 'full' semantics", t.VerificationMode)
	default:
		log.Warnf("output.traceability.ssl.verification_mode %q is not recognized; defaulting to 'full'", t.VerificationMode)
	}

	if len(t.CipherSuites) > 0 {
		tlsCfg.CipherSuites = config.NewCipherArray(t.CipherSuites)
	}

	for _, f := range []unsupportedTLSField{
		{"supported_protocols", len(t.SupportedProtocols) > 0},
		{"certificate_authorities", len(t.CertificateAuthorities) > 0},
		{"curve_types", len(t.CurveTypes) > 0},
		{"ca_sha256", len(t.CASha256) > 0},
		{"renegotiation", t.Renegotiation != ""},
		{"key_passphrase", t.KeyPassphrase != ""},
	} {
		if f.set {
			log.Warnf("output.traceability.ssl.%s is no longer supported and will be ignored", f.name)
		}
	}

	return tlsCfg
}
