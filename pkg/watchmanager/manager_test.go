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

func TestWatchManager_RegisterWatch(t *testing.T) {
	tests := []struct {
		name           string
		tokenGetter    func() (string, error)
		withHarvester  bool
		sequenceID     int64
		subscribeErr   error
		hasErr         bool
		checkCloseConn bool // assert clientMap is cleaned up after CloseConn
	}{
		{
			name:           "success with harvester",
			withHarvester:  true,
			sequenceID:     1,
			hasErr:         false,
			checkCloseConn: true,
		},
		{
			name:         "new client error returns error",
			subscribeErr: fmt.Errorf("subscribe error"),
			hasErr:       true,
		},
		{
			name: "process request error returns error",
			tokenGetter: func() (string, error) {
				return "", fmt.Errorf("token error")
			},
			hasErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tokenGetter := getMockToken
			if tc.tokenGetter != nil {
				tokenGetter = tc.tokenGetter
			}
			cfg := &Config{
				Host:        "localhost",
				Port:        8080,
				TenantID:    "tenantID",
				TokenGetter: tokenGetter,
			}

			opts := []Option{}
			if tc.withHarvester {
				sequence := &testSequenceProvider{}
				sequence.SetSequence(tc.sequenceID)
				opts = append(opts, WithHarvester(&hClient{}, sequence))
			}

			wm, err := New(cfg, opts...)
			assert.Nil(t, err)
			assert.NotNil(t, wm)

			manager := wm.(*watchManager)
			stream := &mockStream{context: context.Background()}
			manager.newWatchClientFunc = newMockWatchClient(stream, tc.subscribeErr)

			events, errors := make(chan *proto.Event), make(chan error)
			_, err = manager.RegisterWatch("/watch/topic", events, errors)
			if tc.hasErr {
				assert.NotNil(t, err)
				assert.Equal(t, 0, len(manager.clientMap))
			} else {
				assert.Nil(t, err)
				assert.Equal(t, 1, len(manager.clientMap))
				if tc.checkCloseConn {
					manager.CloseConn()
					assert.Equal(t, 0, len(manager.clientMap))
				}
			}
		})
	}
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

func TestWatchManager_Status(t *testing.T) {
	tests := []struct {
		name           string
		registerWatch  bool // register a watch client before calling Status
		stopClient     bool // mark the registered client as not running before calling Status
		expectedStatus bool
	}{
		{
			name:           "no clients returns false",
			registerWatch:  false,
			expectedStatus: false,
		},
		{
			name:          "running client does not panic",
			registerWatch: true,
			// Status depends on grpc connectivity; just assert it does not panic
		},
		{
			name:           "stopped client returns false",
			registerWatch:  true,
			stopClient:     true,
			expectedStatus: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				Host:        "localhost",
				Port:        8080,
				TenantID:    "tenantID",
				TokenGetter: getMockToken,
			}
			wm, err := New(cfg)
			assert.Nil(t, err)

			if tc.registerWatch {
				manager := wm.(*watchManager)
				stream := &mockStream{context: context.Background()}
				manager.newWatchClientFunc = newMockWatchClient(stream, nil)

				events, errors := make(chan *proto.Event), make(chan error)
				id, regErr := manager.RegisterWatch("/watch/topic", events, errors)
				assert.Nil(t, regErr)

				if tc.stopClient {
					client, _ := manager.getClient(id)
					client.isRunning.Store(false)
					assert.False(t, wm.Status())
				} else {
					_ = wm.Status()
				}
			} else {
				assert.Equal(t, tc.expectedStatus, wm.Status())
			}
		})
	}
}

func TestWatchManager_CloseWatch(t *testing.T) {
	tests := []struct {
		name          string
		registerWatch bool
		invalidID     bool
		hasErr        bool
	}{
		{
			name:          "success",
			registerWatch: true,
			hasErr:        false,
		},
		{
			name:      "invalid ID returns error",
			invalidID: true,
			hasErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				Host:        "localhost",
				Port:        8080,
				TenantID:    "tenantID",
				TokenGetter: getMockToken,
			}
			wm, err := New(cfg)
			assert.Nil(t, err)

			id := "nonexistent-id"
			if tc.registerWatch {
				manager := wm.(*watchManager)
				stream := &mockStream{context: context.Background()}
				manager.newWatchClientFunc = newMockWatchClient(stream, nil)

				events, errors := make(chan *proto.Event), make(chan error)
				id, err = manager.RegisterWatch("/watch/topic", events, errors)
				assert.Nil(t, err)
			}

			err = wm.CloseWatch(id)
			if tc.hasErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestWatchManager_nilConfig(t *testing.T) {
	var cfg *Config
	_, err := New(cfg)
	assert.NotNil(t, err)
}


func TestWatchManager_getDialer(t *testing.T) {
	tests := []struct {
		name   string
		opts   []Option
		hasErr bool
	}{
		{
			name:   "with proxy",
			opts:   []Option{WithProxy("http://myproxy:8080")},
			hasErr: false,
		},
		{
			name:   "with single entry address",
			opts:   []Option{WithSingleEntryAddr("single-entry:443")},
			hasErr: false,
		},
		{
			name:   "invalid proxy returns error",
			opts:   []Option{WithProxy("://bad-proxy")},
			hasErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				Host:        "localhost",
				Port:        8080,
				TenantID:    "tenantID",
				TokenGetter: getMockToken,
			}
			wm, err := New(cfg, tc.opts...)
			if tc.hasErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, wm)
			}
		})
	}
}

func TestWatchManager_negativeSequenceID(t *testing.T) {
	cfg := &Config{
		Host:        "localhost",
		Port:        8080,
		TenantID:    "tenantID",
		TokenGetter: getMockToken,
	}
	sequence := &testSequenceProvider{}
	sequence.SetSequence(-1)
	cbChan := make(chan struct{}, 1)
	cb := func() { cbChan <- struct{}{} }
	wm, err := New(cfg, WithHarvester(&hClient{}, sequence), WithEventSyncError(cb))
	assert.Nil(t, err)

	manager := wm.(*watchManager)
	stream := &mockStream{context: context.Background()}
	manager.newWatchClientFunc = newMockWatchClient(stream, nil)

	events, errors := make(chan *proto.Event), make(chan error)
	_, err = manager.RegisterWatch("/watch/topic", events, errors)
	assert.NotNil(t, err)

	select {
	case <-cbChan:
	case <-time.After(time.Second):
		t.Fatal("onEventSyncError not called")
	}
	assert.Equal(t, 0, len(manager.clientMap))
}


func TestWatchManager_onHarvesterErr_nilCallback(t *testing.T) {
	cfg := &Config{
		Host:        "localhost",
		Port:        8080,
		TenantID:    "tenantID",
		TokenGetter: getMockToken,
	}
	wm, err := New(cfg)
	assert.Nil(t, err)
	manager := wm.(*watchManager)
	// should not panic when onEventSyncError is nil
	assert.NotPanics(t, func() { manager.onHarvesterErr() })
}

type hClient struct {
	err error
}

func (h hClient) EventCatchUp(ctx context.Context, link string, events chan *proto.Event) error {
	return h.err
}

func (h hClient) ReceiveSyncEvents(ctx context.Context, topicSelfLink string, sequenceID int64, eventCh chan *proto.Event) (int64, error) {
	return 0, nil
}
