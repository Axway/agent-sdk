package traceability

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

func TestTLSConfigVerificationModeDefault(t *testing.T) {
	tests := map[string]struct {
		verificationMode       string
		wantInsecureSkipVerify bool
	}{
		"unset defaults to verify (full)":     {verificationMode: "", wantInsecureSkipVerify: false},
		"full verifies":                       {verificationMode: "full", wantInsecureSkipVerify: false},
		"none skips verification":             {verificationMode: "none", wantInsecureSkipVerify: true},
		"certificate downgrades to full":      {verificationMode: "certificate", wantInsecureSkipVerify: false},
		"strict downgrades to full":           {verificationMode: "strict", wantInsecureSkipVerify: false},
		"unrecognized value defaults to full": {verificationMode: "bogus", wantInsecureSkipVerify: false},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			cfg := TLSConfig{VerificationMode: tc.verificationMode}
			tlsCfg := cfg.toTLSConfiguration()
			assert.Equal(t, tc.wantInsecureSkipVerify, tlsCfg.InsecureSkipVerify)
		})
	}
}

func TestTLSConfigCipherSuites(t *testing.T) {
	tests := map[string]struct {
		cipherSuites []string
		wantLen      int // 0 means "just assert non-empty", used for the default list
	}{
		"explicit cipher suites are honored":                       {cipherSuites: []string{"ECDHE-RSA-AES-128-GCM-SHA256"}, wantLen: 1},
		"unset keeps agent-sdk's own default list rather than being nil'd out": {cipherSuites: nil},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			cfg := TLSConfig{CipherSuites: tc.cipherSuites}
			tlsCfg := cfg.toTLSConfiguration()
			if tc.wantLen > 0 {
				assert.Len(t, tlsCfg.CipherSuites, tc.wantLen)
			} else {
				assert.NotEmpty(t, tlsCfg.CipherSuites)
			}
		})
	}
}

func TestTLSConfigUnsupportedFieldsWarnNotFail(t *testing.T) {
	tests := map[string]struct {
		cfg         TLSConfig
		wantWarning string
	}{
		"supported_protocols": {
			cfg:         TLSConfig{SupportedProtocols: []string{"TLSv1.2"}},
			wantWarning: "output.traceability.ssl.supported_protocols",
		},
		"certificate_authorities": {
			cfg:         TLSConfig{CertificateAuthorities: []string{"/path/to/ca.pem"}},
			wantWarning: "output.traceability.ssl.certificate_authorities",
		},
		"curve_types": {
			cfg:         TLSConfig{CurveTypes: []string{"P-256"}},
			wantWarning: "output.traceability.ssl.curve_types",
		},
		"ca_sha256": {
			cfg:         TLSConfig{CASha256: []string{"deadbeef"}},
			wantWarning: "output.traceability.ssl.ca_sha256",
		},
		"renegotiation": {
			cfg:         TLSConfig{Renegotiation: "freely"},
			wantWarning: "output.traceability.ssl.renegotiation",
		},
		"key_passphrase": {
			cfg:         TLSConfig{KeyPassphrase: "secret"},
			wantWarning: "output.traceability.ssl.key_passphrase",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			originalHooks := log.Get().ReplaceHooks(make(logrus.LevelHooks))
			defer log.Get().ReplaceHooks(originalHooks)
			hook := test.NewLocal(log.Get())

			tlsCfg := tc.cfg.toTLSConfiguration()
			assert.NotNil(t, tlsCfg)

			var found bool
			for _, entry := range hook.AllEntries() {
				if entry.Level == logrus.WarnLevel && strings.Contains(entry.Message, tc.wantWarning) {
					found = true
				}
			}
			assert.True(t, found, "expected a WARN log mentioning %q, got: %v", tc.wantWarning, hook.AllEntries())
		})
	}
}
