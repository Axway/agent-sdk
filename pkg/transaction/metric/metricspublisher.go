package metric

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/google/uuid"
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
	lighthouseUsageEvent, ok := event.(LighthouseUsageEvent)
	if ok {
		pj.publishToLighthouse(lighthouseUsageEvent)
	} else {
		log.Error("event was not a lighthouse event")
		// pj.publishToGatekeeper(event)
	}
}

func (pj *publisher) publishToLighthouse(event LighthouseUsageEvent) {
	token, err := agent.GetCentralAuthToken()
	if err != nil {
		log.Error("Error in sending usage/metric event: ", err.Error())
		return
	}

	b, contentType, err := createMultipartFormData(event)

	headers := map[string]string{
		"Content-Type":  contentType,
		"Authorization": "Bearer " + token,
	}

	request := api.Request{
		Method:  api.POST,
		URL:     agent.GetCentralConfig().GetLighthouseURL() + "/api/v1/usage/automatic",
		Headers: headers,
		Body:    b.Bytes(),
	}
	log.Debugf("Payload for Usage event : %s\n", string(b.Bytes()))
	response, err := pj.apiClient.Send(request)
	if err != nil {
		log.Error("Error in sending usage/metric event: ", err.Error())
		return
	}
	if response.Code >= 400 {
		resBody := string(response.Body)
		log.Error("Failed to publish usage event: ", resBody)
	}
}

func createMultipartFormData(event LighthouseUsageEvent) (b bytes.Buffer, contentType string, err error) {
	buffer, _ := json.Marshal(event)
	w := multipart.NewWriter(&b)
	defer w.Close()
	w.WriteField("organizationId", event.OrgGUID)

	var fw io.Writer
	if fw, err = CreateFilePart(w, uuid.New().String()+".json"); err != nil {
		return
	}
	if _, err = io.Copy(fw, bytes.NewReader(buffer)); err != nil {
		return
	}
	contentType = w.FormDataContentType()

	return
}

//CreateFilePart - adds the file part to the request
func CreateFilePart(w *multipart.Writer, filename string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	h.Set("Content-Type", "application/json")
	return w.CreatePart(h)
}

// NewMetricPublisher - Creates publisher job
func NewMetricPublisher(eventChannel chan interface{}) Publisher {
	centralCfg := agent.GetCentralConfig()
	publisherJob := &publisher{
		eventChannel: eventChannel,
		apiClient:    api.NewClient(centralCfg.GetTLSConfig(), centralCfg.GetProxyURL()),
	}
	_, err := jobs.RegisterSingleRunJob(publisherJob)
	if err != nil {
		panic(err)
	}

	return publisherJob
}
