package watchmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/golang-jwt/jwt"
)

type clientConfig struct {
	errors        chan error
	events        chan *proto.Event
	requests      chan *proto.Request
	tokenGetter   TokenGetter
	topicSelfLink string
}

type watchClient struct {
	cancelStreamCtx        context.CancelFunc
	cfg                    clientConfig
	getTokenExpirationTime getTokenExpFunc
	isRunning              bool
	stream                 proto.Watch_SubscribeClient
	streamCtx              context.Context
	timer                  *time.Timer
	mutex                  sync.Mutex
}

// newWatchClientFunc func signature to create a watch client
type newWatchClientFunc func(cc grpc.ClientConnInterface) proto.WatchClient

type getTokenExpFunc func(token string) (time.Duration, error)

func newWatchClient(cc grpc.ClientConnInterface, clientCfg clientConfig, newClient newWatchClientFunc) (*watchClient, error) {
	svcClient := newClient(cc)

	streamCtx, streamCancel := context.WithCancel(context.Background())
	stream, err := svcClient.Subscribe(streamCtx)
	if err != nil {
		streamCancel()
		return nil, err
	}

	client := &watchClient{
		cancelStreamCtx:        streamCancel,
		cfg:                    clientCfg,
		getTokenExpirationTime: getTokenExpirationTime,
		isRunning:              true,
		stream:                 stream,
		streamCtx:              streamCtx,
		timer:                  time.NewTimer(0),
	}

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
	c.cfg.events <- event
	return nil
}

// processRequest sends a message to the client when the timer expires, and handles when the stream is closed.
func (c *watchClient) processRequest() error {
	var err error
	wg := sync.WaitGroup{}
	wg.Add(1)
	wait := true

	// If the request channel is not supplied, create new
	// for token refresh request
	if c.cfg.requests == nil {
		c.cfg.requests = make(chan *proto.Request, 1)
	}

	go func() {
		for {
			select {
			case <-c.streamCtx.Done():
				c.handleError(c.streamCtx.Err())
				return
			case <-c.stream.Context().Done():
				c.handleError(c.stream.Context().Err())
				return
			case <-c.timer.C:
				err = c.createTokenRefreshRequest()
				if err != nil {
					c.handleError(err)
					return
				}
			case req := <-c.cfg.requests:
				err = c.stream.Send(req)
				if wait {
					wg.Done()
					wait = false
				}
				if err != nil {
					c.handleError(err)
					return
				}
			}
		}
	}()

	wg.Wait()
	return err
}

// create stream request with a new token to the grpc server and returns the expiration time
func (c *watchClient) createTokenRefreshRequest() error {
	c.timer.Stop()

	token, err := c.cfg.tokenGetter()
	if err != nil {
		return err
	}

	exp, err := c.getTokenExpirationTime(token)
	if err != nil {
		return err
	}

	req := createWatchRequest(c.cfg.topicSelfLink, token)
	// write the token request to the channel
	c.cfg.requests <- req
	c.timer.Reset(exp)
	return nil
}

// handleError stop the running timer, send to the error channel, and close the open stream.
func (c *watchClient) handleError(err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.isRunning {
		c.isRunning = false
		c.timer.Stop()
		c.cfg.errors <- err
		c.cancelStreamCtx()
	}
}

func createWatchRequest(watchTopicSelfLink, token string) *proto.Request {
	return &proto.Request{
		SelfLink: watchTopicSelfLink,
		Token:    "Bearer " + token,
	}
}

func getTokenExpirationTime(token string) (time.Duration, error) {
	parser := new(jwt.Parser)
	parser.SkipClaimsValidation = true

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
