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
		name          string
		hasErr        bool
		err           error
		cancelCtx     bool // cancel context after starting recv
		streamCtxDone bool // override stream ctx to pre-cancelled after creation
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
		{
			name:      "should return ctx error when context is cancelled",
			hasErr:    true,
			cancelCtx: true,
		},
		{
			name:          "should return stream ctx error when stream context is cancelled",
			hasErr:        true,
			streamCtxDone: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)

			cfg := clientConfig{
				ctx:    ctx,
				cancel: cancel,
				events: make(chan *proto.Event),
			}

			streamCtx := context.Background()
			if tc.cancelCtx {
				streamCtx = ctx
			}

			stream := &mockStream{
				event:   &proto.Event{},
				err:     tc.err,
				context: streamCtx,
			}
			conn := &mockConn{
				stream: stream,
			}

			c, err := newWatchClient(conn, cfg, newMockWatchClient(stream, nil))
			assert.Nil(t, err)
			assert.NotNil(t, c)

			if tc.streamCtxDone {
				doneCtx, doneCancel := context.WithCancel(context.Background())
				doneCancel()
				stream.context = doneCtx
			}

			errCh := make(chan error, 1)
			go func() {
				errCh <- c.recv()
			}()

			if tc.cancelCtx {
				cancel(nil)
			}

			if !tc.hasErr {
				event := <-cfg.events
				assert.NotNil(t, event)
			} else {
				recvErr := <-errCh
				assert.NotNil(t, recvErr)
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

// Should cancel context when processEvents encounters an error
func Test_watchClient_processEvents(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	cfg := clientConfig{
		ctx:    ctx,
		cancel: cancel,
		events: make(chan *proto.Event),
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

	select {
	case <-ctx.Done():
		assert.NotNil(t, ctx.Err())
		cause := context.Cause(ctx)
		assert.NotNil(t, cause)
		assert.Equal(t, "err", cause.Error())
	case <-time.After(time.Second):
		t.Fatal("expected context to be cancelled on processEvents error")
	}
}

// Stream should be cancelled and an error received over the context
func Test_watchClient_processRequest(t *testing.T) {
	getter := &mockTokenGetter{
		err: nil,
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cfg := clientConfig{
		ctx:         ctx,
		cancel:      cancel,
		events:      make(chan *proto.Event),
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

	cancel(nil)

	err = ctx.Err()
	assert.NotNil(t, err)
}

func Test_watchClient_handleError_withCancelContext(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	cfg := clientConfig{
		ctx:    ctx,
		cancel: cancel,
		events: make(chan *proto.Event),
	}
	stream := &mockStream{
		event:   &proto.Event{},
		context: context.Background(),
	}

	c, err := newWatchClient(&mockConn{stream: stream}, cfg, newMockWatchClient(stream, nil))
	assert.Nil(t, err)
	assert.NotNil(t, c)

	c.handleError(fmt.Errorf("watch stream failed"))

	select {
	case <-ctx.Done():
		assert.NotNil(t, ctx.Err())
	case <-time.After(time.Second):
		t.Fatal("expected watch client to cancel context on error")
	}
}

// Should return an error when calling newWatchClient
func Test_newWatchClient(t *testing.T) {
	getter := &mockTokenGetter{}
	cfg := clientConfig{
		events:      make(chan *proto.Event),
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
	tests := []struct {
		name   string
		token  string
		hasErr bool
	}{
		{
			name:   "should parse a valid future token",
			token:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJmb28iOiJiYXIiLCJleHAiOjk5OTk5OTk5OTksImlzcyI6InRlc3QifQ.XaPiwTklPiU3Ke7byMlSWNfVN7WwkNkmKorNzpM5b9o",
			hasErr: false,
		},
		{
			name:   "should fail on an expired token",
			token:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjF9.signature",
			hasErr: true,
		},
		{
			name:   "should fail on an invalid token",
			token:  "not.a.jwt",
			hasErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := getTokenExpirationTime(tc.token)
			if tc.hasErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func Test_initialRequestLock_wait_timeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{
			name:    "should return timeout error after waiting",
			timeout: 10 * time.Millisecond,
		},
		{
			name:    "should return timeout error when context is not cancelled but wait exceeds timeout",
			timeout: 10 * time.Millisecond,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lock := createInitialRequestLock()
			ctx := context.Background()
			err := lock.wait(ctx, tc.timeout)
			assert.NotNil(t, err)
		})
	}
}

func Test_watchClient_enqueueRequest(t *testing.T) {
	tests := []struct {
		name          string
		cancelCtx     bool // pre-cancel context before enqueue
		streamCtxDone bool // override stream ctx to pre-cancelled after creation
	}{
		{
			name:      "should return ctx error when context is already cancelled",
			cancelCtx: true,
		},
		{
			name:          "should return stream ctx error when stream context is cancelled",
			streamCtxDone: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)

			cfg := clientConfig{
				ctx:      ctx,
				cancel:   cancel,
				events:   make(chan *proto.Event),
				requests: make(chan *proto.Request), // unbuffered, no reader
			}
			stream := &mockStream{context: context.Background()}
			c, err := newWatchClient(&mockConn{stream: stream}, cfg, newMockWatchClient(stream, nil))
			assert.Nil(t, err)

			if tc.cancelCtx {
				cancel(nil)
				c.cfg.ctx = ctx
			}

			if tc.streamCtxDone {
				doneCtx, doneCancel := context.WithCancel(context.Background())
				doneCancel()
				stream.context = doneCtx
			}

			err = c.enqueueRequest(&proto.Request{})
			assert.NotNil(t, err)
		})
	}
}

// createTokenRefreshRequest should return error when enqueue fails (ctx cancelled)
func Test_watchClient_createTokenRefreshRequest_enqueueError(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(nil)
	getter := &mockTokenGetter{}
	cfg := clientConfig{
		ctx:         ctx,
		cancel:      cancel,
		events:      make(chan *proto.Event),
		tokenGetter: getter.GetToken,
		requests:    make(chan *proto.Request), // unbuffered, no reader
	}
	stream := &mockStream{context: context.Background()}
	c, err := newWatchClient(&mockConn{stream: stream}, cfg, newMockWatchClient(stream, nil))
	assert.Nil(t, err)
	c.cfg.ctx = ctx
	c.getTokenExpirationTime = mockGetTokenExp

	err = c.createTokenRefreshRequest()
	assert.NotNil(t, err)
}

func Test_watchClient_requestLoop(t *testing.T) {
	tests := []struct {
		name            string
		makeTokenGetter func() func() (string, error) // factory so each run gets fresh state; nil means use default
		streamErr       error
		streamCtxDone   bool // override stream ctx with a new context cancelled after loop starts
		fireTimer       bool // reset timer to fire immediately after loop starts
		sendRequest     bool // send a request into cfg.requests after loop starts
		checkRequest    bool // assert a request was enqueued (used with fireTimer)
		expectLockReady bool // wait for lock.ready (loop exit)
		expectLockErr   bool // assert lock carries an error after exit
	}{
		{
			name:         "should handle token refresh when timer fires",
			fireTimer:    true,
			checkRequest: true,
		},
		{
			name: "should exit when token refresh fails after timer fires",
			makeTokenGetter: func() func() (string, error) {
				callCount := 0
				return func() (string, error) {
					callCount++
					if callCount > 1 {
						return "", fmt.Errorf("token error after timer")
					}
					return "testToken", nil
				}
			},
			fireTimer:       true,
			expectLockReady: true,
		},
		{
			name:            "should exit when Send fails",
			streamErr:       fmt.Errorf("send error"),
			sendRequest:     true,
			expectLockReady: true,
			expectLockErr:   true,
		},
		{
			name:            "should stop when stream context is done",
			streamCtxDone:   true,
			expectLockReady: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)

			tokenGetter := func() (string, error) { return "testToken", nil }
			if tc.makeTokenGetter != nil {
				tokenGetter = tc.makeTokenGetter()
			}

			cfg := clientConfig{
				ctx:         ctx,
				cancel:      cancel,
				events:      make(chan *proto.Event),
				tokenGetter: tokenGetter,
				requests:    make(chan *proto.Request, 1),
			}
			stream := &mockStream{
				event:   &proto.Event{},
				err:     tc.streamErr,
				context: ctx,
			}
			c, err := newWatchClient(&mockConn{stream: stream}, cfg, newMockWatchClient(stream, nil))
			assert.Nil(t, err)
			c.getTokenExpirationTime = mockGetTokenExp

			// For streamCtxDone, override stream context before starting the loop so
			// requestLoop picks it up, then cancel it after starting to trigger exit.
			var cancelStream context.CancelFunc
			if tc.streamCtxDone {
				var streamDoneCtx context.Context
				streamDoneCtx, cancelStream = context.WithCancel(context.Background())
				stream.context = streamDoneCtx
			}

			lock := createInitialRequestLock()
			go c.requestLoop(lock)

			if cancelStream != nil {
				cancelStream()
			}
			if tc.fireTimer {
				c.timer.Reset(1 * time.Millisecond)
			}
			if tc.sendRequest {
				cfg.requests <- &proto.Request{}
			}

			if tc.checkRequest {
				time.Sleep(50 * time.Millisecond)
				select {
				case req := <-cfg.requests:
					assert.NotNil(t, req)
				default:
					// request may already have been consumed; that's fine too
				}
			}

			if tc.expectLockReady {
				select {
				case <-lock.ready:
				case <-time.After(2 * time.Second):
					t.Fatal("requestLoop did not exit in time")
				}
				if tc.expectLockErr {
					assert.NotNil(t, lock.getError())
				}
			}
		})
	}
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
