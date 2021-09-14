package watchmanager

import (
	"encoding/json"
	"fmt"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

type watchClient struct {
	config Config
	stream proto.WatchService_CreateWatchClient
}

func (c *watchClient) processEvents(eventChannel chan *proto.Event) {
	for {
		event, err := c.stream.Recv()
		fmt.Println("event", event)
		if err != nil {
			log.Errorf("Error while receiving watch events - %s", err.Error())
			close(eventChannel)
			return
		}
		prettyJSON, _ := json.MarshalIndent(event, "", "    ")
		log.Tracef("Received Watch Event : %+v", string(prettyJSON))
		eventChannel <- event
	}
}
