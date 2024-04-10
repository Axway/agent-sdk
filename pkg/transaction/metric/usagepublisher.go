package metric

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type usagePublisher struct {
	apiClient api.Client
	storage   storageCache
	report    *cacheReport
	jobID     string
	ready     bool
	offline   bool
	logger    log.FieldLogger
}

func (c *usagePublisher) publishEvent(event interface{}) error {
	if usageEvent, ok := event.(UsageEvent); ok {
		return c.publishToCache(usageEvent)
	}
	c.logger.Error("event was not a usage event")
	return nil
}

func (c *usagePublisher) publishToCache(event UsageEvent) error {
	return c.report.addReport(event)
}

func (c *usagePublisher) publishToPlatformUsage(event UsageEvent) error {
	token, err := agent.GetCentralAuthToken()
	if err != nil {
		return err
	}

	event = aggregateReports(event)
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

	if response.Code == 202 {
		c.logger.WithField("statusCode", 202).Debugf("successful request with payload: %s", b.String())
		return nil
	} else if response.Code >= 500 {
		err := fmt.Errorf("server error")
		c.logger.WithField("statusCode", response.Code).WithError(err).Error(string(response.Body))
		return err
	}

	usageResp := UsageResponse{}
	err = json.Unmarshal(response.Body, &usageResp)
	if err != nil {
		c.logger.WithField("responseBody", string(response.Body)).WithField("statusCode", response.Code).
			WithError(err).Error("Could not unmarshal response body")
		return err
	}
	if strings.HasPrefix(usageResp.Description, "The file exceeds the maximum upload size of ") || usageResp.Description == "Environment ID not found" {
		err := fmt.Errorf("request failed with unexpected status code. Not scheduling for retry in the next batch")
		c.logger.WithField("statusCode", response.Code).WithError(err).Error(usageResp.Description)
		return nil
	}
	err = fmt.Errorf("request failed with unexpected status code")
	c.logger.WithField("statusCode", response.Code).WithError(err).Error(usageResp.Description)
	return err
}

func (c *usagePublisher) createMultipartFormData(event UsageEvent) (b bytes.Buffer, contentType string, err error) {
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

func aggregateReports(event UsageEvent) UsageEvent {

	// order all the keys, this will be used to find first and last timestamp
	orderedKeys := make([]string, 0, len(event.Report))
	for k := range event.Report {
		orderedKeys = append(orderedKeys, k)
	}
	sort.Strings(orderedKeys)

	// create a single report which has all eventReports appended
	finalReport := map[string]UsageReport{
		orderedKeys[0]: {
			Product: event.Report[orderedKeys[0]].Product,
			Usage:   make(map[string]int64),
			Meta:    event.Report[orderedKeys[0]].Meta,
		},
	}

	for _, report := range event.Report {
		for usageKey, usageVal := range report.Usage {
			finalReport[orderedKeys[0]].Usage[usageKey] += usageVal
		}
	}
	event.Report = finalReport

	startTime, _ := time.Parse(ISO8601, orderedKeys[0])
	endTime := now()
	event.Granularity = int(endTime.Sub(startTime).Milliseconds())
	event.Timestamp = ISO8601Time(endTime)
	return event
}

// createFilePart - adds the file part to the request
func (c *usagePublisher) createFilePart(w *multipart.Writer, filename string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	h.Set("Content-Type", "application/json")
	return w.CreatePart(h)
}

// newUsagePublisher - Creates publisher job
func newUsagePublisher(storage storageCache, report *cacheReport) *usagePublisher {
	centralCfg := agent.GetCentralConfig()
	publisher := &usagePublisher{
		apiClient: api.NewClient(centralCfg.GetTLSConfig(), centralCfg.GetProxyURL(),
			api.WithTimeout(centralCfg.GetClientTimeout()),
			api.WithSingleURL()),
		storage: storage,
		report:  report,
		offline: agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode(),
		logger:  log.NewFieldLogger().WithComponent("usagePublisher").WithPackage("metric"),
	}

	publisher.registerReportJob()
	return publisher
}

func (c *usagePublisher) isReady() bool {
	return c.ready
}

func (c *usagePublisher) registerReportJob() {
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
func (c *usagePublisher) Status() error {
	return nil
}

// Ready - indicates that the offline report job is ready to process
//
//	additionally runs the initial report gen if the last trigger would
//	have ran but the agent was down
func (c *usagePublisher) Ready() bool {
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
func (c *usagePublisher) Execute() error {
	if c.offline {
		return c.report.saveReport()
	}
	return c.report.sendReport(c.publishToPlatformUsage)
}
