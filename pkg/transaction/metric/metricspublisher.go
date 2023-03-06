package metric

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strconv"

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
}

func (pj *metricPublisher) publishEvent(event interface{}) error {
	if lighthouseUsageEvent, ok := event.(LighthouseUsageEvent); ok {
		return pj.publishToCache(lighthouseUsageEvent)
	}
	log.Error("event was not a lighthouse event")
	return nil
}

func (pj *metricPublisher) publishToCache(event LighthouseUsageEvent) error {
	// Open and load the existing usage file
	savedEvents := pj.report.loadEvents()

	for key, report := range event.Report {
		savedEvents.Report[key] = report
	}
	// Put all reports into the new event
	event.Report = savedEvents.Report

	// Update the cache
	pj.report.updateEvents(event)

	return nil
}

func (pj *metricPublisher) publishToLighthouse(event LighthouseUsageEvent) error {
	token, err := agent.GetCentralAuthToken()
	if err != nil {
		return err
	}

	b, contentType, err := pj.createMultipartFormData(event)
	if err != nil {
		return err
	}

	headers := map[string]string{
		"Content-Type":  contentType,
		"Authorization": "Bearer " + token,
	}

	request := api.Request{
		Method:  api.POST,
		URL:     agent.GetCentralConfig().GetUsageReportingConfig().GetURL() + "/api/v1/usage/automatic",
		Headers: headers,
		Body:    b.Bytes(),
	}
	log.Debugf("Payload for Usage event : %s\n", b.String())
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
func newMetricPublisher(storage storageCache, report *cacheReport) *metricPublisher {
	centralCfg := agent.GetCentralConfig()
	publisher := &metricPublisher{
		apiClient: api.NewSingleEntryClient(centralCfg.GetTLSConfig(), centralCfg.GetProxyURL(), centralCfg.GetClientTimeout()),
		storage:   storage,
		report:    report,
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
		log.Errorf("could not register usage report creation job: %s", err.Error())
	}
}

// saveReport - creates a new file with the latest cached events then clears all reports from the cache, lock outside of this
func (c *metricPublisher) saveReport() error {
	savedEvents := c.report.getEvents()

	// no reports yet, skip creating the event
	if len(savedEvents.Report) == 0 {
		return nil
	}
	savedEvents = c.report.validateReport(savedEvents)

	// create the path to save the file
	outputFilePath := ""
	i := 0
	fileExists := true
	for fileExists {
		outputFilePath = c.report.generateReportPath(savedEvents.Timestamp, i)
		_, err := os.Stat(outputFilePath)
		i++
		fileExists = !os.IsNotExist(err)
	}

	// create the new file to save the events
	file, err := os.Create(filepath.Clean(outputFilePath))
	if err != nil {
		return err
	}

	// marshal the event into json bytes
	cacheBytes, err := json.Marshal(savedEvents)
	if err != nil {
		file.Close()
		return err
	}

	// save the bytes and close the file
	_, err = io.Copy(file, bytes.NewReader(cacheBytes))
	file.Close()
	if err != nil {
		return err
	}

	// clear out all reports
	savedEvents.Report = make(map[string]LighthouseUsageReport)
	c.report.setEvents(savedEvents)
	return nil
}

// sendReport - creates a new report with the latest cached events then clears all reports from the cache, lock outside of this
func (c *metricPublisher) sendReport() error {
	savedEvents := c.report.getEvents()

	fmt.Printf("\n********\n %+v \n********\n", savedEvents)
	defer func() {
		savedEvents := c.report.getEvents()
		fmt.Printf("\n********\n %+v \n********\n", savedEvents)
	}()

	// no reports yet, skip creating the event
	if len(savedEvents.Report) == 0 {
		return nil
	}
	savedEvents = c.report.validateReport(savedEvents)
	if err := c.publishToLighthouse(savedEvents); err != nil {
		log.Error("could not publish usage, will send at next scheduled publishing")
		return nil
	}

	savedEvents.Report = make(map[string]LighthouseUsageReport)
	c.report.setEvents(savedEvents)
	return nil
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

	c.report.reportCacheLock.Lock()
	defer c.report.reportCacheLock.Unlock()
	savedEvents := c.report.getEvents()
	if len(savedEvents.Report) > 0 {
		// A report should have ran while agent was down
		err := c.saveReport()
		if err != nil {
			log.Errorf("error hit generating report, report still in cache: %s", err.Error())
		}
		return true
	}
	return true
}

// Execute - process the offline report generation
func (c *metricPublisher) Execute() error {
	c.report.reportCacheLock.Lock()
	defer c.report.reportCacheLock.Unlock()
	if c.report.offline {
		return c.saveReport()
	}
	return c.sendReport()
}
