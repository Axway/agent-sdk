package watchmanager

import (
	"context"
	"time"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type watchClient struct {
	watchTopicSelfLink      string
	tokenGetter             TokenGetter
	stream                  proto.WatchService_CreateWatchClient
	processEventWaitContext context.Context
	processEventWaitCancel  context.CancelFunc
}

func newWatchClient(topicSelfLink string, tokenGetter TokenGetter, stream proto.WatchService_CreateWatchClient) *watchClient {
	processEventWaitContext, cancel := context.WithCancel(context.Background())
	client := &watchClient{
		tokenGetter:             tokenGetter,
		watchTopicSelfLink:      topicSelfLink,
		stream:                  stream,
		processEventWaitContext: processEventWaitContext,
		processEventWaitCancel:  cancel,
	}
	return client
}

func (c *watchClient) processEvents(eventChannel chan *proto.Event, errorChannel chan error) {
	<-c.processEventWaitContext.Done()
	for {
		event, err := c.stream.Recv()
		if err != nil {
			errorChannel <- err
			close(eventChannel)
			return
		}
		if event != nil {
			eventChannel <- event
		}
	}
}

func (c *watchClient) processRequest(errorChannel chan error) {
	wait := time.Duration(0)
	for {
		select {
		case <-c.stream.Context().Done():
			return
		case <-time.After(wait):
			token, err := c.tokenGetter()
			if err != nil {
				c.stream.Context().Done()
				errorChannel <- err
				return
			}
			req := c.createWatchRequest(c.watchTopicSelfLink, token)
			err = c.stream.Send(req)
			if err != nil {
				errorChannel <- err
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
