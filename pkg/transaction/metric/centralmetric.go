package metric

import (
	"strings"
	"time"

	"github.com/Axway/agent-sdk/pkg/transaction/models"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/sirupsen/logrus"
)

// metricInfo - the base object holding the metricInfo
type metricInfo struct {
	Subscription  *models.Subscription  `json:"subscription,omitempty"`
	App           *models.AppDetails    `json:"app,omitempty"`
	Product       *models.Product       `json:"product,omitempty"`
	API           *models.APIDetails    `json:"api,omitempty"`
	AssetResource *models.AssetResource `json:"assetResource,omitempty"`
	ProductPlan   *models.ProductPlan   `json:"productPlan,omitempty"`
	Quota         *models.Quota         `json:"quota,omitempty"`
	Unit          *models.Unit          `json:"unit,omitempty"`
	StatusCode    string                `json:"statusCode,omitempty"`
}

// centralMetricEvent - the event that is actually sent to platform
type centralMetricEvent struct {
	metricInfo
	Status      string              `json:"status,omitempty"`
	Count       int64               `json:"count"`
	Response    *ResponseMetrics    `json:"response,omitempty"`
	Observation *ObservationDetails `json:"observation"`
	EventID     string              `json:"-"`
	StartTime   time.Time           `json:"-"`
}

// GetStartTime - Returns the start time for subscription metric
func (a *centralMetricEvent) GetStartTime() time.Time {
	return a.StartTime
}

// GetType - Returns APIMetric
func (a *centralMetricEvent) GetType() string {
	return "APIMetric"
}

// GetType - Returns APIMetric
func (a *centralMetricEvent) GetEventID() string {
	return a.EventID
}

func (a *centralMetricEvent) GetLogFields() logrus.Fields {
	fields := logrus.Fields{
		"id":             a.EventID,
		"count":          a.Count,
		"status":         a.StatusCode,
		"minResponse":    a.Response.Min,
		"maxResponse":    a.Response.Max,
		"avgResponse":    a.Response.Avg,
		"startTimestamp": a.Observation.Start,
		"endTimestamp":   a.Observation.End,
	}
	if a.Subscription != nil {
		fields = a.Subscription.GetLogFields(fields)
	}
	if a.App != nil {
		fields = a.App.GetLogFields(fields)
	}
	if a.Product != nil {
		fields = a.Product.GetLogFields(fields)
	}
	if a.API != nil {
		fields = a.API.GetLogFields(fields)
	}
	if a.AssetResource != nil {
		fields = a.AssetResource.GetLogFields(fields)
	}
	if a.ProductPlan != nil {
		fields = a.ProductPlan.GetLogFields(fields)
	}
	if a.Quota != nil {
		fields = a.Quota.GetLogFields(fields)
	}
	if a.Unit != nil {
		fields = a.Unit.GetLogFields(fields)
	}
	return fields
}

// getKey - returns the cache key for the metric
func (a *centralMetricEvent) getKey() string {
	subID := unknown
	if a.Subscription != nil {
		subID = a.Subscription.ID
	}
	appID := unknown
	if a.App != nil {
		subID = a.App.ID
	}
	apiID := unknown
	if a.API != nil {
		subID = a.API.ID
	}
	uniqueKey := unknown
	if a.StatusCode == "" {
		uniqueKey = a.StatusCode
	} else if a.Unit != nil {
		uniqueKey = a.Unit.ID
	}

	return metricKeyPrefix + strings.Join([]string{subID, appID, apiID, uniqueKey}, ".")
}

func (a *centralMetricEvent) createdCachedMetric(histogram metrics.Histogram) cachedMetric {
	return cachedMetric{
		metricInfo: a.metricInfo,
		StartTime:  a.StartTime,
		Count:      histogram.Count(),
		Values:     histogram.Sample().Values(),
	}
}
