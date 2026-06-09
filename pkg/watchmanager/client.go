package watchmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"

	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/golang-jwt/jwt/v5"
)

const (
	initDuration = 30 * time.Second
)

type clientConfig struct {
	ctx           context.Context
	cancel        context.CancelCauseFunc
	events        chan *proto.Event
	tokenGetter   TokenGetter
	topicSelfLink string
	requests      chan *proto.Request
}

type watchClient struct {
	cfg                    clientConfig
	getTokenExpirationTime getTokenExpFunc
	isRunning              atomic.Bool
	stream                 proto.Watch_SubscribeClient
	timer                  *time.Timer
	logger                 log.FieldLogger
}

// newWatchClientFunc func signature to create a watch client
type newWatchClientFunc func(cc grpc.ClientConnInterface) proto.WatchClient

type getTokenExpFunc func(token string) (time.Duration, error)

func newWatchClient(cc grpc.ClientConnInterface, clientCfg clientConfig, newClient newWatchClientFunc) (*watchClient, error) {
	svcClient := newClient(cc)

	// If no context is provided, create a new one
	if clientCfg.ctx == nil {
		clientCfg.ctx, clientCfg.cancel = context.WithCancelCause(context.Background())
	}

	stream, err := svcClient.Subscribe(clientCfg.ctx)
	if err != nil {
		clientCfg.cancel(err)
		return nil, err
	}

	client := &watchClient{
		cfg:                    clientCfg,
		getTokenExpirationTime: getTokenExpirationTime,
		stream:                 stream,
		timer:                  time.NewTimer(initDuration),
		logger:                 log.NewFieldLogger().WithComponent("watchManager").WithPackage("sdk.client"),
	}
	client.isRunning.Store(true)
	return client, nil
}

// processEvents process incoming chimera events
func (c *watchClient) processEvents() {
	for {
		err := c.recv()
		if err != nil {
			c.handleError(err)
			return
		}
	}
}

// recv blocks until an event is received
func (c *watchClient) recv() error {
	event, err := c.stream.Recv()
	if err != nil {
		return err
	}

	if event != nil && event.Type == proto.Event_PING {
		c.logger.Trace("received watch subscription server ping event")
		go c.sendPingResponse(event.Id)
		return nil
	}

	select {
	case c.cfg.events <- event:
		return nil
	case <-c.cfg.ctx.Done():
		return c.cfg.ctx.Err()
	case <-c.stream.Context().Done():
		return c.stream.Context().Err()
	}
}

func (c *watchClient) sendPingResponse(id string) {
	req := &proto.Request{
		RequestType:  proto.RequestType_PING_RESPONSE.Enum(),
		PingResponse: &proto.PingResponse{Id: id},
	}
	if err := c.enqueueRequest(req); err != nil {
		c.logger.WithError(err).Warn("failed to send ping response")
	}
}

// processRequest sends a message to the client when the timer expires, and handles when the stream is closed.
func (c *watchClient) processRequest() error {
	// If the request channel is not supplied, create new
	// for token refresh request
	if c.cfg.requests == nil {
		c.cfg.requests = make(chan *proto.Request, 1)
	}
	lock := createInitialRequestLock()
	go c.requestLoop(lock)

	// writes the initial watch request and resets the timer
	err := c.initialRequest()
	if err != nil {
		c.handleError(err)
		return err
	}

	return lock.wait(c.cfg.ctx, initDuration)
}

func (c *watchClient) initialRequest() error {
	return c.createTokenRefreshRequest(true)
}

func (c *watchClient) requestLoop(rl *initialRequestLock) {
	var err error
	defer func() {
		rl.done(err)
	}()

	for {
		select {
		case <-c.stream.Context().Done():
			c.handleError(c.stream.Context().Err())
			return
		case <-c.timer.C:
			err = c.createTokenRefreshRequest(false)
			if err != nil {
				c.handleError(err)
				return
			}
		case req := <-c.cfg.requests:
			err = c.stream.Send(req)
			rl.done(err)
			if err != nil {
				c.handleError(err)
				return
			}
		}
	}
}

// create stream request with a new token to the grpc server and returns the expiration time
func (c *watchClient) createTokenRefreshRequest(initialRequest bool) error {
	c.timer.Stop()

	token, err := c.cfg.tokenGetter()
	if err != nil {
		return err
	}

	exp, err := c.getTokenExpirationTime(token)
	if err != nil {
		return err
	}

	req := createWatchRequest(c.cfg.topicSelfLink, token, initialRequest)
	if err = c.enqueueRequest(req); err != nil {
		return err
	}

	c.timer.Reset(exp)
	return nil
}

func (c *watchClient) enqueueRequest(req *proto.Request) error {
	select {
	case c.cfg.requests <- req:
		return nil
	case <-c.cfg.ctx.Done():
		return c.cfg.ctx.Err()
	case <-c.stream.Context().Done():
		return c.stream.Context().Err()
	}
}

// shouldCancelContext checks if the context should be canceled. Returns true only once per client lifecycle.
func (c *watchClient) shouldCancelContext() bool {
	if c.isRunning.CompareAndSwap(true, false) {
		c.timer.Stop()
		if c.cfg.ctx.Err() == nil {
			return true
		}
	}
	return false
}

// handleError stops the running timer and cancels the stream context
func (c *watchClient) handleError(err error) {
	if !c.shouldCancelContext() {
		return
	}
	c.cfg.cancel(err)
}

func createWatchRequest(watchTopicSelfLink, token string, initialRequest bool) *proto.Request {
	req := &proto.Request{
		SelfLink: watchTopicSelfLink,
		Token:    "Bearer " + token,
	}
	if initialRequest {
		SetCapabilities(req, []Capability{CapabilityPing})
	}
	return req
}

func getTokenExpirationTime(token string) (time.Duration, error) {
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	claims := jwt.MapClaims{}
	_, _, err := parser.ParseUnverified(token, claims)
	if err != nil {
		return time.Duration(0), fmt.Errorf("getTokenExpirationTime failed to parse token: %s", err)
	}

	var tm time.Time
	switch exp := claims["exp"].(type) {
	case float64:
		tm = time.Unix(int64(exp), 0)
	case json.Number:
		v, _ := exp.Int64()
		tm = time.Unix(v, 0)
	}

	exp := time.Until(tm)
	// use big.NewInt to avoid an int overflow
	i := big.NewInt(int64(exp))
	i = i.Mul(i, big.NewInt(4))
	i = i.Div(i, big.NewInt(5))
	d := time.Duration(i.Int64())

	if d.Milliseconds() < 0 {
		return time.Duration(0), fmt.Errorf("token is expired")
	}
	return d, nil
}

type initialRequestLock struct {
	once  sync.Once
	ready chan struct{}
	lock  sync.Mutex
	err   error
}

func createInitialRequestLock() *initialRequestLock {
	return &initialRequestLock{
		ready: make(chan struct{}),
	}
}

func (l *initialRequestLock) done(err error) {
	l.once.Do(func() {
		l.setError(err)
		close(l.ready)
	})
}

func (l *initialRequestLock) wait(ctx context.Context, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-l.ready:
		return l.getError()
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return fmt.Errorf("timed out waiting for initial watch request after %s", timeout)
	}
}

func (l *initialRequestLock) setError(err error) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.err = err
}

func (l *initialRequestLock) getError() error {
	l.lock.Lock()
	defer l.lock.Unlock()
	return l.err
}
