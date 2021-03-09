package metric

import (
	"encoding/json"
	"fmt"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/jobs"
)

// Publisher - interface for metric publisher
type Publisher interface {
	jobs.Job
}

type publisher struct {
	apiClient    api.Client
	eventChannel chan interface{}
}

// Status - returns the status of publisher job
func (pj *publisher) Status() error {
	return nil
}

// Ready - indicates the publisher is ready to process
func (pj *publisher) Ready() bool {
	return true
}

// Execute - process the publishing of events sent on event channel
func (pj *publisher) Execute() error {
	for {
		select {
		case event, ok := <-pj.eventChannel:
			if ok {
				pj.publishEvent(event)
			}
		}
	}
}

func (pj *publisher) publishEvent(event interface{}) {
	buffer, _ := json.Marshal(event)
	fmt.Println(string(buffer))
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	headers["x-org-id"] = agent.GetCentralConfig().GetTenantID()
	request := api.Request{
		Method:  api.POST,
		URL:     agent.GetCentralConfig().GetGateKeeperURL(),
		Headers: headers,
		Body:    buffer,
	}
	_, err := pj.apiClient.Send(request)
	if err != nil {
		fmt.Println(err.Error())
	}
}

// NewMetricPublisher - Creates publisher job
func NewMetricPublisher(eventChannel chan interface{}) Publisher {
	publisherJob := &publisher{
		eventChannel: eventChannel,
		apiClient:    api.NewClient(nil, ""),
	}
	_, err := jobs.RegisterSingleRunJob(publisherJob)
	if err != nil {
		panic(err)
	}

	return publisherJob
}
