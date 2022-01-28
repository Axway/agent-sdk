package watchmanager

import (
	"context"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/golang-jwt/jwt"

	"github.com/stretchr/testify/assert"
)

func getMockToken() (string, error) {
	claims := &jwt.StandardClaims{
		ExpiresAt: time.Now().Add(time.Minute * 1).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signKey := []byte("testsecret")
	token, err := t.SignedString(signKey)
	return token, err
}

// test register watch
func TestWatchManager_RegisterWatch(t *testing.T) {
	cfg := &Config{
		Host:        "localhost",
		Port:        8080,
		TenantID:    "tenantID",
		TokenGetter: getMockToken,
	}
	wm, err := New(cfg)
	assert.Nil(t, err)
	assert.NotNil(t, wm)

	manager := wm.(*watchManager)
	stream := &mockStream{
		context: context.Background(),
	}
	manager.newWatchClientFunc = newMockWatchClient(stream, nil)

	events, errors := make(chan *proto.Event), make(chan error)
	id, err := manager.RegisterWatch("/watch/topic", events, errors)
	assert.Nil(t, err)

	err = manager.CloseWatch(id)
	assert.Nil(t, err)
}

func TestConfig(t *testing.T) {
	cfg := Config{
		Host:        "",
		Port:        0,
		TenantID:    "",
		TokenGetter: nil,
	}

	err := cfg.validateCfg()
	assert.NotNil(t, err)

	cfg.Host = "abc.com"
	err = cfg.validateCfg()
	assert.NotNil(t, err)

	cfg.TenantID = "123"
	err = cfg.validateCfg()
	assert.NotNil(t, err)

	cfg.TokenGetter = func() (string, error) {
		return "abc", nil
	}
	err = cfg.validateCfg()
	assert.Nil(t, err)
}
