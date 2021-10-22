package watchmanager

import (
	"context"
	"time"

	"google.golang.org/grpc"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type watchClientConfig struct {
	topicSelfLink string
	tokenGetter   TokenGetter
	eventChannel  chan *proto.Event
	errorChannel  chan error
}

type watchClient struct {
	cfg          watchClientConfig
	stream       proto.WatchService_CreateWatchClient
	cancelStream context.CancelFunc
	timer        *time.Timer
}

func newWatchClient(cc grpc.ClientConnInterface, clientCfg watchClientConfig) (*watchClient, error) {
	svcClient := proto.NewWatchServiceClient(cc)

	streamCtx, streamCancel := context.WithCancel(context.Background())
	stream, err := svcClient.CreateWatch(streamCtx)
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
			err := c.send()
			if err != nil {
				c.handleError(err)
				return
			}
			c.timer.Reset(29 * time.Minute)
		}
	}
}

// send a message with a new token to the grpc server
func (c *watchClient) send() error {
	token, err := c.cfg.tokenGetter()
	if err != nil {
		return err
	}
	req := createWatchRequest(c.cfg.topicSelfLink, token)
	err = c.stream.Send(req)
	if err != nil {
		return err
	}
	return nil
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
