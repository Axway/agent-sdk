package watch

import (
	"encoding/json"

	watchProto "github.com/Axway/agent-sdk/pkg/axway/apicentral/watch/proto"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type watchClient struct {
	config Config
	stream watchProto.WatchService_CreateWatchClient
}

func (c *watchClient) processEvents(eventChannel chan *watchProto.Event) {
	for {
		event, err := c.stream.Recv()
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
