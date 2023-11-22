package metric

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"

	"github.com/google/uuid"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type metricPublisher struct {
	apiClient api.Client
	storage   storageCache
	report    *cacheReport
	jobID     string
	ready     bool
	offline   bool
	logger    log.FieldLogger
}

func (c *metricPublisher) publishEvent(event interface{}) error {
	if lighthouseUsageEvent, ok := event.(LighthouseUsageEvent); ok {
		return c.publishToCache(lighthouseUsageEvent)
	}
	c.logger.Error("event was not a lighthouse event")
	return nil
}

func (c *metricPublisher) publishToCache(event LighthouseUsageEvent) error {
	return c.report.addReport(event)
}

func (c *metricPublisher) publishToLighthouse(event LighthouseUsageEvent) error {
	token, err := agent.GetCentralAuthToken()
	if err != nil {
		return err
	}

	b, contentType, err := c.createMultipartFormData(event)
	if err != nil {
		return err
	}

	headers := map[string]string{
		"Content-Type":  contentType,
		"Authorization": "Bearer " + token,
	}

	request := api.Request{
		Method:  api.POST,
		URL:     agent.GetCentralConfig().GetUsageReportingConfig().GetURL() + "/api/v1/usage",
		Headers: headers,
		Body:    b.Bytes(),
	}
	c.logger.Debugf("Payload for Usage event : %s\n", b.String())
	response, err := c.apiClient.Send(request)
	if err != nil {
		c.logger.WithError(err).Error("publishing usage")
		return err
	}
	if response.Code >= 400 {
		resBody := string(response.Body)
		err := fmt.Errorf("request failed with unexpected status code")
		c.logger.WithField("statusCode", response.Code).WithError(err).Error(resBody)
		return err
	}
	return nil
}

func (c *metricPublisher) createMultipartFormData(event LighthouseUsageEvent) (b bytes.Buffer, contentType string, err error) {
	buffer, _ := json.Marshal(event)
	w := multipart.NewWriter(&b)
	defer w.Close()
	w.WriteField("organizationId", event.OrgGUID)

	var fw io.Writer
	if fw, err = c.createFilePart(w, uuid.New().String()+".json"); err != nil {
		return
	}
	if _, err = io.Copy(fw, bytes.NewReader(buffer)); err != nil {
		return
	}
	contentType = w.FormDataContentType()

	return
}

// createFilePart - adds the file part to the request
func (c *metricPublisher) createFilePart(w *multipart.Writer, filename string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	h.Set("Content-Type", "application/json")
	return w.CreatePart(h)
}

// newMetricPublisher - Creates publisher job
func newMetricPublisher(storage storageCache, report *cacheReport) *metricPublisher {
	centralCfg := agent.GetCentralConfig()
	publisher := &metricPublisher{
		apiClient: api.NewClient(centralCfg.GetTLSConfig(), centralCfg.GetProxyURL(),
			api.WithTimeout(centralCfg.GetClientTimeout()),
			api.WithSingleURL()),
		storage: storage,
		report:  report,
		offline: agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode(),
		logger:  log.NewFieldLogger().WithComponent("metricPublisher").WithPackage("metric"),
	}

	publisher.registerReportJob()
	return publisher
}

func (c *metricPublisher) isReady() bool {
	return c.ready
}

func (c *metricPublisher) registerReportJob() {
	if !util.IsNotTest() {
		return // skip setting up the job in test
	}

	schedule := agent.GetCentralConfig().GetUsageReportingConfig().GetUsageSchedule()
	if agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode() {
		schedule = agent.GetCentralConfig().GetUsageReportingConfig().GetReportSchedule()
	}

	// start the job according to the cron schedule
	var err error
	c.jobID, err = jobs.RegisterScheduledJobWithName(c, schedule, "Usage Reporting")
	if err != nil {
		c.logger.WithError(err).Error("could not register usage report creation job")
	}
}

// Status - returns an error if the status of the offline report job is in error
func (c *metricPublisher) Status() error {
	return nil
}

// Ready - indicates that the offline report job is ready to process
//
//	additionally runs the initial report gen if the last trigger would
//	have ran but the agent was down
func (c *metricPublisher) Ready() bool {
	if agent.GetCentralConfig().GetEnvironmentID() == "" {
		return false
	}

	defer func() {
		c.ready = true
	}() // once any existing reports are saved off this isReady

	err := c.Execute()
	if err != nil {
		c.logger.WithError(err).Errorf("error hit generating report, report still in cache")
	}
	return true
}

// Execute - process the offline report generation
func (c *metricPublisher) Execute() error {
	if c.offline {
		return c.report.saveReport()
	}
	return c.report.sendReport(c.publishToLighthouse)
}
