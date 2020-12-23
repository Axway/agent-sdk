package config

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTLSConfig(t *testing.T) {
	cfg := NewTLSConfig()

	cfgValidator, ok := cfg.(IConfigValidator)
	assert.True(t, ok)
	assert.NotNil(t, cfgValidator)

	err := cfgValidator.ValidateCfg()
	assert.Nil(t, err)

	assert.Equal(t, cfg.IsInsecureSkipVerify(), false)
	assert.Equal(t, cfg.GetMinVersion(), TLSDefaultMinVersion)
	assert.Equal(t, cfg.GetMaxVersion(), TLSVersion(0))
	assert.Equal(t, cfg.GetNextProtos(), []string{})
	assert.Equal(t, cfg.GetCipherSuites(), TLSDefaultCipherSuites)
}

func TestBuildTLSConfig(t *testing.T) {
	cfg := NewTLSConfig()
	cfgValidator, ok := cfg.(IConfigValidator)
	assert.True(t, ok)
	assert.NotNil(t, cfgValidator)

	err := cfgValidator.ValidateCfg()
	assert.Nil(t, err)

	cfg2 := cfg.BuildTLSConfig()

	assert.Equal(t, cfg.IsInsecureSkipVerify(), cfg2.InsecureSkipVerify)
	assert.Equal(t, uint16(cfg.GetMinVersion()), cfg2.MinVersion)
	assert.Equal(t, uint16(cfg.GetMaxVersion()), cfg2.MaxVersion)
	assert.Equal(t, cfg.GetNextProtos(), cfg2.NextProtos)

	cfg3, ok := cfg.(*TLSConfiguration)
	assert.Equal(t, ok, true)
	assert.Equal(t, cfg3.buildUintArrayFromSuites(), cfg2.CipherSuites)
}

func TestValidate(t *testing.T) {
	cfg := NewTLSConfig()
	cfgValidator, ok := cfg.(IConfigValidator)
	assert.True(t, ok)
	assert.NotNil(t, cfgValidator)

	err := cfgValidator.ValidateCfg()
	assert.Nil(t, err)

	cfg2, ok := cfg.(*TLSConfiguration)
	assert.Equal(t, ok, true)

	min := cfg2.MinVersion

	cfg2.MinVersion = TLSVersion(0)
	cfgValidator2, _ := cfg.(IConfigValidator)

	err = cfgValidator2.ValidateCfg()
	assert.Nil(t, err)

	cfg2.MinVersion = TLSVersion(455)
	cfgValidator2, _ = cfg.(IConfigValidator)

	err = cfgValidator2.ValidateCfg()
	assert.NotNil(t, err)
	assert.Equal(t, "Error: ssl.minVersion not valid in config", err.Error())
	cfg2.MinVersion = min

	max := cfg2.MaxVersion
	cfg2.MaxVersion = TLSVersion(0)
	cfgValidator2, _ = cfg.(IConfigValidator)
	err = cfgValidator2.ValidateCfg()
	assert.Nil(t, err)

	cfg2.MaxVersion = TLSVersion(455)
	cfgValidator2, _ = cfg.(IConfigValidator)
	err = cfgValidator2.ValidateCfg()
	assert.NotNil(t, err)
	assert.Equal(t, "Error: ssl.maxVersion not valid in config", err.Error())
	cfg2.MaxVersion = max

	cfg2.CipherSuites = []TLSCipherSuite{TLSCipherSuite(888)}
	cfgValidator2, _ = cfg.(IConfigValidator)
	err = cfgValidator2.ValidateCfg()
	assert.NotNil(t, err)
	assert.Equal(t, "Error: ssl.cipherSuites not valid in config", err.Error())
}

func TestUnpackTLSVersion(t *testing.T) {
	ver := TLSVersion(0)
	err := ver.Unpack("TLS1.2")
	assert.Nil(t, err)

	err = ver.Unpack("TLS1.8")
	assert.NotNil(t, err)
	assert.Equal(t, "invalid tls version 'TLS1.8'", err.Error())
}

func TestDefaultMinVersionString(t *testing.T) {
	assert.Equal(t, TLSDefaultMinVersionString(), tlsVersionsInverse[TLSDefaultMinVersion])
}

func TestTLSVersionAsValue(t *testing.T) {
	assert.Equal(t, TLSVersionAsValue("0"), TLSVersion(0))
	assert.Equal(t, uint16(TLSVersionAsValue("TLS1.2")), uint16(tls.VersionTLS12))
}

func TestTLSDefaultCipherSuitesStringSlice(t *testing.T) {
	cfg := NewTLSConfig()
	suites := TLSDefaultCipherSuitesStringSlice()

	assert.Equal(t, cfg.GetCipherSuites(), NewCipherArray(suites))
}

func TestUnpackTLSCipherSuiteString(t *testing.T) {
	suite := TLSCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA)
	err := suite.Unpack("ECDHE-ECDSA-AES-128-CBC-SHA")
	assert.Nil(t, err)

	err = suite.Unpack("WRONG-SHA")
	assert.NotNil(t, err)
	assert.Equal(t, "invalid tls cipher suite 'WRONG-SHA'", err.Error())
}

// func (cs *TLSCipherSuite) String() string {
// 	if s, found := tlsCipherSuitesInverse[*cs]; found {
// 		return s
// 	}
// 	return "unknown"
// }
