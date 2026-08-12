package transaction

import (
	"errors"
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/transaction/metric"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	transutil "github.com/Axway/agent-sdk/pkg/transaction/util"
	"github.com/elastic/beats/v7/libbeat/common"
)

type EventReport interface {
	GetSummaryEvent() LogEvent
	GetDetailEvents() []LogEvent
	GetMetricsBatch() []interface{}
	GetEventTime() time.Time
	GetMetadata() common.MapStr
	GetFields() common.MapStr
	GetPrivateData() interface{}
	ShouldForceSample() bool
	ShouldHandleSampling() bool
	ShouldTrackMetrics() bool
	ShouldOnlyTrackMetrics() bool
}

type unitDetails struct {
	count     int64
	startTime time.Time
	endTime   time.Time
}

type eventReport struct {
	summaryEvent     *LogEvent
	proxy            *Proxy
	app              *Application
	detailEvents     []LogEvent
	metricsBatch     []interface{}
	metricsBatchLock sync.Mutex
	eventTime        time.Time
	llmModel         string
	metadata         common.MapStr
	fields           common.MapStr
	units            map[metric.UnitType]unitDetails
	privateData      interface{}
	skipSampling     bool
	forceSample      bool
	skipTracking     bool
	trackOnly        bool
}

func (e *eventReport) GetSummaryEvent() LogEvent {
	if e.summaryEvent == nil {
		return LogEvent{}
	}
	return *e.summaryEvent
}

func (e *eventReport) GetDetailEvents() []LogEvent {
	if e.detailEvents == nil {
		e.detailEvents = []LogEvent{}
	}
	return e.detailEvents
}

func (e *eventReport) GetMetricsBatch() []interface{} {
	e.metricsBatchLock.Lock()
	defer e.metricsBatchLock.Unlock()

	// reset metrics batch
	metricsBatch := e.metricsBatch
	e.metricsBatch = make([]interface{}, 0)

	return metricsBatch
}

func (e *eventReport) GetEventTime() time.Time {
	return e.eventTime
}

func (e *eventReport) GetMetadata() common.MapStr {
	if e.metadata == nil {
		e.metadata = common.MapStr{}
	}
	return e.metadata
}

func (e *eventReport) GetFields() common.MapStr {
	if e.metadata == nil {
		e.metadata = common.MapStr{}
	}
	return e.fields
}

func (e *eventReport) GetPrivateData() interface{} {
	return e.privateData
}

func (e *eventReport) ShouldHandleSampling() bool {
	return !e.skipSampling
}

func (e *eventReport) ShouldForceSample() bool {
	return e.forceSample
}

func (e *eventReport) ShouldTrackMetrics() bool {
	return !e.skipTracking
}

func (e *eventReport) ShouldOnlyTrackMetrics() bool {
	return e.trackOnly
}

type EventReportBuilder interface {
	SetSummaryEvent(summaryEvent LogEvent) EventReportBuilder
	SetProxy(proxy Proxy) EventReportBuilder
	SetApplication(app Application) EventReportBuilder
	SetDetailEvents(detailEvents []LogEvent) EventReportBuilder
	SetEventTime(eventTime time.Time) EventReportBuilder
	SetMetadata(metadata common.MapStr) EventReportBuilder
	SetFields(fields common.MapStr) EventReportBuilder
	SetPrivateData(privateData interface{}) EventReportBuilder
	SetSkipSampleHandling() EventReportBuilder
	SetForceSample() EventReportBuilder
	SetSkipMetricTracking() EventReportBuilder
	SetOnlyTrackMetrics(trackOnly bool) EventReportBuilder
	SetLLMModel(model string) EventReportBuilder
	SetUnitAmount(unit metric.UnitType, count int64, start, end time.Time) EventReportBuilder
	Build() (EventReport, error)
}

func NewEventReportBuilder() EventReportBuilder {
	return &eventReport{
		detailEvents:     []LogEvent{},
		metricsBatch:     make([]interface{}, 0),
		metricsBatchLock: sync.Mutex{},
		eventTime:        time.Now(),
		units:            map[metric.UnitType]unitDetails{},
		metadata:         common.MapStr{},
		fields:           common.MapStr{},
		privateData:      nil,
	}
}

func (e *eventReport) SetSummaryEvent(summaryEvent LogEvent) EventReportBuilder {
	e.syncProxyToAPI(summaryEvent.TransactionSummary)
	e.summaryEvent = &summaryEvent
	return e
}

// Helper method that handles all the proxy-to-API synchronization
func (e *eventReport) syncProxyToAPI(summary *Summary) {
	// Guard clauses for early return
	if summary == nil || summary.Proxy == nil {
		return
	}

	proxy := summary.Proxy

	// Resolve proxy ID
	resolvedID := transutil.ResolveIDWithPrefix(proxy.ID, proxy.Name)
	proxy.ID = resolvedID

	// Sync to API object
	if summary.API == nil {
		summary.API = &models.APIDetails{}
	}
	summary.API.ID = resolvedID
	summary.API.Name = proxy.Name
}

func (e *eventReport) SetProxy(proxy Proxy) EventReportBuilder {
	e.proxy = &proxy
	return e
}

func (e *eventReport) SetApplication(app Application) EventReportBuilder {
	e.app = &app
	return e
}

func (e *eventReport) SetDetailEvents(detailEvents []LogEvent) EventReportBuilder {
	e.detailEvents = detailEvents
	return e
}

func (e *eventReport) SetEventTime(eventTime time.Time) EventReportBuilder {
	e.eventTime = eventTime
	return e
}

func (e *eventReport) SetMetadata(metadata common.MapStr) EventReportBuilder {
	e.metadata = metadata
	return e
}

func (e *eventReport) SetFields(fields common.MapStr) EventReportBuilder {
	e.fields = fields
	return e
}

func (e *eventReport) SetPrivateData(privateData interface{}) EventReportBuilder {
	e.privateData = privateData
	return e
}

func (e *eventReport) SetSkipSampleHandling() EventReportBuilder {
	e.skipSampling = true
	return e.SetForceSample()
}

func (e *eventReport) SetForceSample() EventReportBuilder {
	e.forceSample = true
	return e
}

func (e *eventReport) SetSkipMetricTracking() EventReportBuilder {
	e.skipTracking = true
	return e
}

func (e *eventReport) SetLLMModel(model string) EventReportBuilder {
	e.llmModel = model
	return e
}

func (e *eventReport) SetUnitAmount(unit metric.UnitType, count int64, start, end time.Time) EventReportBuilder {
	e.units[unit] = unitDetails{
		count:     count,
		startTime: start,
		endTime:   end,
	}
	return e
}

func (e *eventReport) SetOnlyTrackMetrics(trackOnly bool) EventReportBuilder {
	e.trackOnly = trackOnly
	return e
}

func (e *eventReport) Build() (EventReport, error) {
	if e.skipTracking && e.trackOnly {
		return nil, errors.New("can't set skip tracking and track only in a single event")
	}

	// check for reported units and add to metric batch
	if err := e.handleUnits(); err != nil {
		return nil, err
	}

	// if only metrics are reported, no need to check for summary
	if e.trackOnly {
		return e, nil
	}

	if e.summaryEvent == nil && (e.proxy == nil || e.app == nil) {
		return nil, errors.New("need api and app info to create summary event")
	}

	// create summary event
	if e.summaryEvent == nil && e.proxy != nil && e.app != nil {
		e.summaryEvent = &LogEvent{
			TransactionSummary: &Summary{
				Proxy: e.proxy,
				Team: &Team{
					ID: agent.GetCentralConfig().GetTeamID(),
				},
				Application: e.app,
			},
		}
	}

	return e, nil
}

// handleUnits converts any reported units into metric batch entries
func (e *eventReport) handleUnits() error {
	if len(e.units) == 0 {
		return nil
	}
	if e.proxy == nil || e.app == nil {
		return errors.New("need api and app info to report unit metrics")
	}

	apiDetails := models.APIDetails{
		ID:    e.proxy.ID,
		Name:  e.proxy.Name,
		Stage: e.proxy.Stage,
	}
	appDetails := models.AppDetails{
		ID:   e.app.ID,
		Name: e.app.Name,
	}

	if e.llmModel != "" {
		e.handleLLMMetric(apiDetails, appDetails)
	} else {
		e.appendCustomMetrics(apiDetails, appDetails)
	}
	return nil
}

// handleLLMMetric creates an llm metric for a single model
func (e *eventReport) handleLLMMetric(apiDetails models.APIDetails, appDetails models.AppDetails) {
	llmUnits := make(map[metric.UnitType]int64, len(e.units))
	var start, end time.Time
	for name, u := range e.units {
		llmUnits[name] = u.count
		if start.IsZero() || u.startTime.Before(start) {
			start = u.startTime
		}
		if u.endTime.After(end) {
			end = u.endTime
		}
	}

	e.metricsBatch = append(e.metricsBatch, metric.LLMMetricDetail{
		APIDetails: apiDetails,
		AppDetails: appDetails,
		Model:      e.llmModel,
		Units:      llmUnits,
		Observation: models.ObservationDetails{
			Start: start.UnixMilli(),
			End:   end.UnixMilli(),
		},
	})
}

// appendCustomMetrics reports each unit as an individual custom metric detail.
func (e *eventReport) appendCustomMetrics(apiDetails models.APIDetails, appDetails models.AppDetails) {
	for name, u := range e.units {
		e.metricsBatch = append(e.metricsBatch, models.CustomMetricDetail{
			APIDetails: apiDetails,
			AppDetails: appDetails,
			Observation: models.ObservationDetails{
				Start: u.startTime.UnixMilli(),
				End:   u.endTime.UnixMilli(),
			},
			UnitDetails: models.Unit{
				Name: name.String(),
			},
			Count: u.count,
		})
	}
}
