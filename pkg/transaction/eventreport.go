package transaction

import (
	"errors"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/elastic/beats/v7/libbeat/common"
)

type EventReport interface {
	GetSummaryEvent() LogEvent
	GetDetailEvents() []LogEvent
	GetMetricDetails() []interface{}
	GetEventTime() time.Time
	GetMetadata() common.MapStr
	GetFields() common.MapStr
	GetPrivateData() interface{}
	ShouldForceSample() bool
	ShouldHandleSampling() bool
	ShouldTrackMetrics() bool
	ShouldOnlyTrackMetrics() bool
	AddMetricDetail(metricDetail interface{})
}

type eventReport struct {
	summaryEvent  *LogEvent
	proxy         *Proxy
	app           *Application
	detailEvents  []LogEvent
	metricDetails []interface{}
	eventTime     time.Time
	metadata      common.MapStr
	fields        common.MapStr
	privateData   interface{}
	skipSampling  bool
	forceSample   bool
	skipTracking  bool
	trackOnly     bool
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

func (e *eventReport) GetMetricDetails() []interface{} {
	return e.metricDetails
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

func (e *eventReport) AddMetricDetail(metricDetail interface{}) {
	e.metricDetails = append(e.metricDetails, metricDetail)
}

type EventReportBuilder interface {
	SetSummaryEvent(summaryEvent LogEvent) EventReportBuilder
	SetDetailEvents(detailEvents []LogEvent) EventReportBuilder
	SetEventTime(eventTime time.Time) EventReportBuilder
	SetMetadata(metadata common.MapStr) EventReportBuilder
	SetFields(fields common.MapStr) EventReportBuilder
	SetPrivateData(privateData interface{}) EventReportBuilder
	SetSkipSampleHandling() EventReportBuilder
	SetForceSample() EventReportBuilder
	SetSkipMetricTracking() EventReportBuilder
	SetOnlyTrackMetrics(trackOnly bool) EventReportBuilder
	Build() (EventReport, error)
}

func NewEventReportBuilder() EventReportBuilder {
	return &eventReport{
		detailEvents:  []LogEvent{},
		metricDetails: []interface{}{},
		eventTime:     time.Now(),
		metadata:      common.MapStr{},
		fields:        common.MapStr{},
		privateData:   nil,
	}
}

func (e *eventReport) SetSummaryEvent(summaryEvent LogEvent) EventReportBuilder {
	e.summaryEvent = &summaryEvent
	return e
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

func (e *eventReport) SetOnlyTrackMetrics(trackOnly bool) EventReportBuilder {
	e.trackOnly = trackOnly
	return e
}

func (e *eventReport) Build() (EventReport, error) {
	if e.skipTracking && e.trackOnly {
		return nil, errors.New("can't set skip tracking and track only in a single event")
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
