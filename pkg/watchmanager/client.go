package watchmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/grpc"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/golang-jwt/jwt"
)

type watchClientConfig struct {
	topicSelfLink string
	tokenGetter   TokenGetter
	eventChannel  chan *proto.Event
	errorChannel  chan error
}

type watchClient struct {
	cfg          watchClientConfig
	stream       proto.Watch_SubscribeClient
	cancelStream context.CancelFunc
	timer        *time.Timer
}

func newWatchClient(cc grpc.ClientConnInterface, clientCfg watchClientConfig) (*watchClient, error) {
	svcClient := proto.NewWatchClient(cc)

	streamCtx, streamCancel := context.WithCancel(context.Background())
	stream, err := svcClient.Subscribe(streamCtx)
	if err != nil {
		streamCancel()
		return nil, err
	}

	client := &watchClient{
		cfg:          clientCfg,
		stream:       stream,
		cancelStream: streamCancel,
		timer:        time.NewTimer(0),
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
	c.cfg.eventChannel <- event
	return nil
}

// processRequest sends a message to the client when the timer expires, and handles when the stream is closed.
func (c *watchClient) processRequest() {
	for {
		select {
		case <-c.stream.Context().Done():
			c.handleError(c.stream.Context().Err())
			return
		case <-c.timer.C:
			exp, err := c.send()
			if err != nil {
				c.handleError(err)
				return
			}
			c.timer.Reset(exp)
		}
	}
}

// send a message with a new token to the grpc server and returns the expiration time
func (c *watchClient) send() (time.Duration, error) {
	token, err := c.cfg.tokenGetter()
	if err != nil {
		return time.Duration(0), err
	}
	exp, err := getTokenExpirationTime(token)
	if err != nil {
		return exp, err
	}
	req := createWatchRequest(c.cfg.topicSelfLink, token)
	err = c.stream.Send(req)
	if err != nil {
		return exp, err
	}
	return exp, nil
}

// handleError stop the running timer, send to the error channel, and close the open stream.
func (c *watchClient) handleError(err error) {
	c.timer.Stop()
	c.cfg.errorChannel <- err
	close(c.cfg.eventChannel)
	c.cancelStream()
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
	return time.Duration((exp * 4) / 5), nil
}
