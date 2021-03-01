package transaction

import (
	"encoding/json"
	"fmt"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/api"
)

func CreatePublisher(eventChannel chan interface{}) {
	apiClient := api.NewClient(nil, "")
	go func() {
		for {
			select {
			case aggergationEvent, ok := <-eventChannel:
				if ok {
					buffer, _ := json.Marshal(aggergationEvent)
					fmt.Println(string(buffer))
					headers := make(map[string]string)
					headers["Content-Type"] = "application/json"
					request := api.Request{
						Method:  api.POST,
						URL:     agent.GetCentralConfig().GetGateKeeperURL(),
						Headers: headers,
						Body:    buffer,
					}
					_, err := apiClient.Send(request)
					if err != nil {
						fmt.Println(err.Error())
					}
				}
			}
		}
	}()
}
