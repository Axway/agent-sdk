package watchmanager

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/golang-jwt/jwt/v5"

	"github.com/stretchr/testify/assert"
)

func getMockToken() (string, error) {
	claims := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 1)),
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
	sequence := &testSequenceProvider{}
	sequence.SetSequence(1)
	wm, err := New(cfg, WithHarvester(&hClient{}, sequence))
	assert.Nil(t, err)
	assert.NotNil(t, wm)

	manager := wm.(*watchManager)
	stream := &mockStream{
		context: context.Background(),
	}
	manager.newWatchClientFunc = newMockWatchClient(stream, nil)

	events, errors := make(chan *proto.Event), make(chan error)
	_, err = manager.RegisterWatch("/watch/topic", events, errors)
	assert.Nil(t, err)

	assert.Equal(t, len(manager.clientMap), 1)

	manager.CloseConn()

	assert.Equal(t, len(manager.clientMap), 0)
}

func TestWatchManager_OnError(t *testing.T) {
	cfg := &Config{
		Host:        "localhost",
		Port:        8080,
		TenantID:    "tenantID",
		TokenGetter: getMockToken,
	}
	sequence := &testSequenceProvider{}
	sequence.SetSequence(1)
	hc := &hClient{
		err: fmt.Errorf("error"),
	}
	cbChan := make(chan struct{})
	cb := func() {
		go func() {
			cbChan <- struct{}{}
		}()
	}
	wm, err := New(cfg, WithHarvester(hc, sequence), WithEventSyncError(cb))
	assert.Nil(t, err)
	assert.NotNil(t, wm)

	manager := wm.(*watchManager)
	stream := &mockStream{
		context: context.Background(),
	}
	manager.newWatchClientFunc = newMockWatchClient(stream, nil)

	events, errors := make(chan *proto.Event), make(chan error)
	_, err = manager.RegisterWatch("/watch/topic", events, errors)
	assert.NotNil(t, err)

	// expect that the callback func for a harvester error was called
	v := <-cbChan
	assert.NotNil(t, v)

	assert.Equal(t, len(manager.clientMap), 0)
}

func TestWatchManager_zeroSequenceID(t *testing.T) {
	cfg := &Config{
		Host:        "localhost",
		Port:        8080,
		TenantID:    "tenantID",
		TokenGetter: getMockToken,
	}
	sequence := &testSequenceProvider{}
	hc := &hClient{
		err: fmt.Errorf("error"),
	}
	cbChan := make(chan struct{})
	cb := func() {
		go func() {
			cbChan <- struct{}{}
		}()
	}
	wm, err := New(cfg, WithHarvester(hc, sequence), WithEventSyncError(cb))
	assert.Nil(t, err)
	assert.NotNil(t, wm)

	manager := wm.(*watchManager)
	stream := &mockStream{
		context: context.Background(),
	}
	manager.newWatchClientFunc = newMockWatchClient(stream, nil)

	events, errors := make(chan *proto.Event), make(chan error)
	_, err = manager.RegisterWatch("/watch/topic", events, errors)
	assert.NotNil(t, err)

	// expect that the callback func for a harvester error was called
	v := <-cbChan
	assert.NotNil(t, v)

	assert.Equal(t, len(manager.clientMap), 0)
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

type hClient struct {
	err error
}

func (h hClient) EventCatchUp(link string, events chan *proto.Event) error {
	return h.err
}

func (h hClient) ReceiveSyncEvents(topicSelfLink string, sequenceID int64, eventCh chan *proto.Event) (int64, error) {
	return 0, nil
}
