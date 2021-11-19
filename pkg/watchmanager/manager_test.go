package watchmanager

import (
	"context"
	"testing"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	"github.com/stretchr/testify/assert"
)

// test register watch
func TestWatchManager_RegisterWatch(t *testing.T) {
	cfg := &Config{
		Host:     "localhost",
		Port:     8080,
		TenantID: "tenantID",
		TokenGetter: func() (string, error) {
			return "abc", nil
		},
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
