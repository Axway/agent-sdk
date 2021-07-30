package metric

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"strconv"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/google/uuid"
)

// publisher - interface for metric publisher
type publisher interface {
	publishEvent(event interface{}) error
}

type metricPublisher struct {
	apiClient api.Client
	storage   storageCache
}

func (pj *metricPublisher) publishEvent(event interface{}) error {
	if lighthouseUsageEvent, ok := event.(LighthouseUsageEvent); ok {
		if agent.GetCentralConfig().GetEventAggregationOffline() {
			return pj.publishToFile(lighthouseUsageEvent)
		}
		return pj.publishToLighthouse(lighthouseUsageEvent)
	}
	log.Error("event was not a lighthouse event")
	return nil
}

func (pj *metricPublisher) publishToFile(event LighthouseUsageEvent) error {
	// Open and load the existing usage file
	savedEvents, loaded := pj.storage.loadOfflineEvents()

	if loaded {
		// Add the report from the latest event to the saved events
		for key, report := range event.Report {
			savedEvents.Report[key] = report
		}
		savedEvents.Timestamp = event.Timestamp
	} else {
		savedEvents = event
	}

	// Update the cache
	pj.storage.updateOfflineEvents(savedEvents)

	return nil
}

func (pj *metricPublisher) publishToLighthouse(event LighthouseUsageEvent) error {
	token, err := agent.GetCentralAuthToken()
	if err != nil {
		return err
	}

	b, contentType, err := pj.createMultipartFormData(event)

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
		return err
	}
	if response.Code >= 400 {
		resBody := string(response.Body)
		return errors.New("Request failed with code: " + strconv.Itoa(response.Code) + ", content: " + resBody)
	}
	return nil
}

func (pj *metricPublisher) createMultipartFormData(event LighthouseUsageEvent) (b bytes.Buffer, contentType string, err error) {
	buffer, _ := json.Marshal(event)
	w := multipart.NewWriter(&b)
	defer w.Close()
	w.WriteField("organizationId", event.OrgGUID)

	var fw io.Writer
	if fw, err = pj.createFilePart(w, uuid.New().String()+".json"); err != nil {
		return
	}
	if _, err = io.Copy(fw, bytes.NewReader(buffer)); err != nil {
		return
	}
	contentType = w.FormDataContentType()

	return
}

// createFilePart - adds the file part to the request
func (pj *metricPublisher) createFilePart(w *multipart.Writer, filename string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	h.Set("Content-Type", "application/json")
	return w.CreatePart(h)
}

// newMetricPublisher - Creates publisher job
func newMetricPublisher(storage storageCache) publisher {
	centralCfg := agent.GetCentralConfig()
	publisher := &metricPublisher{
		apiClient: api.NewClient(centralCfg.GetTLSConfig(), centralCfg.GetProxyURL()),
		storage:   storage,
	}

	return publisher
}
