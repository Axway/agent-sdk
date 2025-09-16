package watchmanager

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func Test_watchClient_recv(t *testing.T) {
	tests := []struct {
		name   string
		hasErr bool
		err    error
	}{
		{
			name:   "should call recv and return nil",
			hasErr: false,
			err:    nil,
		},
		{
			name:   "should return an error when calling recv",
			hasErr: true,
			err:    fmt.Errorf("error"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := clientConfig{
				events: make(chan *proto.Event),
				errors: make(chan error),
			}
			stream := &mockStream{
				event: &proto.Event{},
				err:   tc.err,
			}
			conn := &mockConn{
				stream: stream,
			}

			c, err := newWatchClient(conn, cfg, newMockWatchClient(stream, nil))
			assert.Nil(t, err)
			assert.NotNil(t, c)

			errCh := make(chan error)
			go func() {
				err := c.recv()
				errCh <- err
			}()

			if !tc.hasErr {
				event := <-cfg.events
				assert.NotNil(t, event)
				assert.Nil(t, err)
			} else {
				err = <-errCh
				assert.NotNil(t, err)
			}
		})
	}
}

func Test_watchClient_send(t *testing.T) {
	tests := []struct {
		name        string
		getTokenErr error
		streamErr   error
		hasErr      bool
		hasSendErr  bool
		getToken    getTokenExpFunc
	}{
		{
			name:        "should call send without an error",
			getTokenErr: nil,
			streamErr:   nil,
			hasErr:      false,
			getToken:    mockGetTokenExp,
		},
		{
			name:        "should fail when unable to parse the token",
			getTokenErr: nil,
			streamErr:   nil,
			hasErr:      true,
			getToken:    mockGetTokenExpFail,
		},
		{
			name:        "should fail when unable to retrieve a token",
			getTokenErr: fmt.Errorf("err"),
			streamErr:   nil,
			hasErr:      true,
			getToken:    mockGetTokenExp,
		},
		{
			name:        "should return an error when Send fails",
			getTokenErr: nil,
			streamErr:   fmt.Errorf("err"),
			hasSendErr:  true,
			getToken:    mockGetTokenExp,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			getter := &mockTokenGetter{
				err: tc.getTokenErr,
			}

			cfg := clientConfig{
				events:      make(chan *proto.Event),
				errors:      make(chan error),
				tokenGetter: getter.GetToken,
				requests:    make(chan *proto.Request, 1),
			}

			stream := &mockStream{
				event: &proto.Event{},
				err:   tc.streamErr,
			}

			conn := &mockConn{
				stream: stream,
			}

			wg := sync.WaitGroup{}
			c, err := newWatchClient(conn, cfg, newMockWatchClient(stream, nil))
			c.getTokenExpirationTime = tc.getToken
			assert.Nil(t, err)
			assert.NotNil(t, c)

			wg.Add(1)
			if !tc.hasErr {
				go func() {
					defer wg.Done()
					err := c.processRequest()
					if tc.hasSendErr {
						assert.NotNil(t, err)
					} else {
						assert.Nil(t, err)
					}
				}()

				// allow the request channel to listen
				time.Sleep(time.Second)
			}

			err = c.createTokenRefreshRequest()
			if tc.hasErr {
				assert.NotNil(t, err)
			} else if !tc.hasSendErr {
				wg.Wait()
				assert.Nil(t, err)
				assert.NotNil(t, stream.getRequest())
			}
		})
	}

}

// Should write an error to the error channel when calling processEvents
func Test_watchClient_processEvents(t *testing.T) {
	cfg := clientConfig{
		events: make(chan *proto.Event),
		errors: make(chan error),
	}
	stream := &mockStream{
		event: &proto.Event{},
		err:   fmt.Errorf("err"),
	}
	conn := &mockConn{
		stream: stream,
	}

	c, err := newWatchClient(conn, cfg, newMockWatchClient(stream, nil))
	assert.Nil(t, err)
	assert.NotNil(t, c)

	go c.processEvents()

	err = <-cfg.errors
	assert.NotNil(t, err)
}

// Should write an error to the error channel when the stream context is cancelled.
func Test_watchClient_processRequest(t *testing.T) {
	getter := &mockTokenGetter{
		err: nil,
	}
	ctx, cancel := context.WithCancel(context.Background())
	cfg := clientConfig{
		ctx:         ctx,
		cancel:      cancel,
		events:      make(chan *proto.Event),
		errors:      make(chan error),
		tokenGetter: getter.GetToken,
	}
	stream := &mockStream{
		event:   &proto.Event{},
		err:     fmt.Errorf("err"),
		context: ctx,
	}
	conn := &mockConn{
		stream: stream,
	}

	c, err := newWatchClient(conn, cfg, newMockWatchClient(stream, nil))
	assert.Nil(t, err)
	assert.NotNil(t, c)

	go c.processRequest()

	cancel()

	err = <-cfg.errors
	assert.NotNil(t, err)
}

// Should return an error when calling newWatchClient
func Test_newWatchClient(t *testing.T) {
	getter := &mockTokenGetter{}
	cfg := clientConfig{
		events:      make(chan *proto.Event),
		errors:      make(chan error),
		tokenGetter: getter.GetToken,
	}
	stream := &mockStream{
		event:   &proto.Event{},
		err:     fmt.Errorf("err"),
		context: context.Background(),
	}
	conn := &mockConn{
		stream: stream,
	}

	_, err := newWatchClient(conn, cfg, newMockWatchClient(stream, fmt.Errorf("err")))
	assert.NotNil(t, err)
}

func Test_getTokenExpirationTime(t *testing.T) {
	futureTokenString := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJmb28iOiJiYXIiLCJleHAiOjk5OTk5OTk5OTksImlzcyI6InRlc3QifQ.XaPiwTklPiU3Ke7byMlSWNfVN7WwkNkmKorNzpM5b9o"
	_, err := getTokenExpirationTime(futureTokenString)
	assert.Nil(t, err)
}

type mockTokenGetter struct {
	err error
}

func (m *mockTokenGetter) GetToken() (string, error) {
	return "testToken", m.err
}

func mockGetTokenExp(_ string) (time.Duration, error) {
	return 30 * time.Second, nil
}

func mockGetTokenExpFail(_ string) (time.Duration, error) {
	return 0 * time.Second, fmt.Errorf("err")
}
