package config

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/util/exception"
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
	assert.Equal(t, "[Error Code 1401] - error with config central.auth.url, please set and/or check its value", err.Error())
	assert.Equal(t, "", cfg.GetTokenURL())
	assert.Equal(t, "", cfg.GetAudience())

	authCfg.URL = "aaa"
	err = validateAuth(cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "[Error Code 1401] - error with config central.auth.realm, please set and/or check its value", err.Error())
	assert.Equal(t, "", cfg.GetTokenURL())
	assert.Equal(t, "", cfg.GetAudience())

	authCfg.Realm = "rrr"
	err = validateAuth(cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "[Error Code 1401] - error with config central.auth.clientId, please set and/or check its value", err.Error())
	assert.NotEqual(t, "", cfg.GetTokenURL())
	assert.NotEqual(t, "", cfg.GetAudience())

	authCfg.ClientID = "cccc"
	err = validateAuth(cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "[Error Code 1401] - error with config central.auth.privateKey, please set and/or check its value", err.Error())

	fs, err := ioutil.TempFile(".", "test*")
	authCfg.PrivateKey = "./" + fs.Name()
	err = validateAuth(cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "[Error Code 1401] - error with config central.auth.publicKey, please set and/or check its value", err.Error())

	authCfg.PublicKey = "./" + fs.Name()
	err = validateAuth(cfg)
	assert.Nil(t, err)

	assert.Equal(t, "aaa/realms/rrr"+tokenEndpoint, cfg.GetTokenURL())
	assert.Equal(t, "aaa/realms/rrr", cfg.GetAudience())
	assert.Equal(t, "", cfg.GetKeyPassword())
	authCfg.KeyPwd = "xxx"
	assert.Equal(t, "xxx", cfg.GetKeyPassword())
	assert.Equal(t, 30*time.Second, cfg.GetTimeout())

	// cleanup files
	err = os.Remove("./" + fs.Name())
	assert.Nil(t, err)
}
