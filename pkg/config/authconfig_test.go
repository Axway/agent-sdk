package config

import (
	"testing"
	"time"

	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/exception"
	"github.com/stretchr/testify/assert"
)

func validateAuth(cfg AuthConfig) (err error) {
	exception.Block{
		Try: func() {
			cfg.validate()
		},
		Catch: func(e error) {
			err = e
		},
	}.Do()
	return
}

func TestAuhConfig(t *testing.T) {
	cfg := newAuthConfig()
	authCfg := cfg.(*AuthConfiguration)
	err := validateAuth(cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "Error auth.url not set in config", err.Error())
	assert.Equal(t, "", cfg.GetTokenURL())
	assert.Equal(t, "", cfg.GetAudience())

	authCfg.URL = "aaa"
	err = validateAuth(cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "Error auth.realm not set in config", err.Error())
	assert.Equal(t, "", cfg.GetTokenURL())
	assert.Equal(t, "", cfg.GetAudience())

	authCfg.Realm = "rrr"
	err = validateAuth(cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "Error auth.clientid not set in config", err.Error())
	assert.NotEqual(t, "", cfg.GetTokenURL())
	assert.NotEqual(t, "", cfg.GetAudience())

	authCfg.ClientID = "cccc"
	err = validateAuth(cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "Error auth.privatekey not set in config", err.Error())

	authCfg.PrivateKey = "ppp"
	err = validateAuth(cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "Error auth.publickey not set in config", err.Error())

	authCfg.PublicKey = "bbbb"
	err = validateAuth(cfg)
	assert.Nil(t, err)

	assert.Equal(t, "aaa/realms/rrr"+tokenEndpoint, cfg.GetTokenURL())
	assert.Equal(t, "aaa/realms/rrr", cfg.GetAudience())
	assert.Equal(t, "", cfg.GetKeyPassword())
	authCfg.KeyPwd = "xxx"
	assert.Equal(t, "xxx", cfg.GetKeyPassword())
	assert.Equal(t, 30*time.Second, cfg.GetTimeout())
}
