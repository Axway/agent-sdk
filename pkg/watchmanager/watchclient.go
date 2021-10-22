package watchmanager

import (
	"context"
	"time"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type watchClientConfig struct {
	topicSelfLink string
	tokenGetter   TokenGetter
	eventChannel  chan *proto.Event
	errorChannel  chan error
}

type watchClient struct {
	cfg                     watchClientConfig
	stream                  proto.WatchService_CreateWatchClient
	streamWaitCancel        context.CancelFunc
	processEventWaitContext context.Context
	processEventWaitCancel  context.CancelFunc
}

func newWatchClient(svcClient proto.WatchServiceClient, clientCfg watchClientConfig) (*watchClient, error) {
	streamContext, streamCancel := context.WithCancel(context.Background())
	stream, err := svcClient.CreateWatch(streamContext)
	if err != nil {
		streamCancel()
		return nil, err
	}

	processEventWaitContext, cancel := context.WithCancel(context.Background())
	client := &watchClient{
		cfg:                     clientCfg,
		stream:                  stream,
		streamWaitCancel:        streamCancel,
		processEventWaitContext: processEventWaitContext,
		processEventWaitCancel:  cancel,
	}
	return client, nil
}

func (c *watchClient) processEvents() {
	<-c.processEventWaitContext.Done()
	for {
		event, err := c.stream.Recv()
		if err != nil {
			c.closeWithError(err)
			return
		}
		c.cfg.eventChannel <- event
	}
}

func (c *watchClient) processRequest() {
	wait := time.Duration(0)
	for {
		select {
		case <-c.stream.Context().Done():
			c.closeChannel(c.stream.Context().Err())
			return
		case <-time.After(wait):
			token, err := c.cfg.tokenGetter()
			if err != nil {
				c.closeWithError(err)
				return
			}
			req := c.createWatchRequest(c.cfg.topicSelfLink, token)
			err = c.stream.Send(req)
			if err != nil {
				c.closeWithError(err)
				return
			}
			if wait == 0 {
				c.processEventWaitCancel()
				// token expiration time
				wait = time.Minute * 29
			}
		}
	}
}

func (c *watchClient) createWatchRequest(watchTopicSelfLink, token string) *proto.Request {
	trigger := &proto.Request{
		SelfLink: watchTopicSelfLink,
		Token:    "Bearer " + token,
	}

	return trigger
}

func (c *watchClient) closeChannel(err error) {
	c.cfg.errorChannel <- err
	close(c.cfg.eventChannel)
}

func (c *watchClient) closeWithError(err error) {
	c.closeChannel(err)
	c.close()
}
func (c *watchClient) close() {
	// Should trigger closing the event channel writing nil to error channel
	c.streamWaitCancel()
}
