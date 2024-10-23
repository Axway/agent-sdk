package metric

import (
	"time"

	"github.com/Axway/agent-sdk/pkg/transaction/models"
	"github.com/sirupsen/logrus"
)

// APIMetric - struct to hold metric aggregated for subscription,application,api,statuscode
type APIMetric struct {
	Subscription  models.Subscription  `json:"subscription,omitempty"`
	App           models.AppDetails    `json:"application,omitempty"`
	Product       models.Product       `json:"product,omitempty"`
	API           models.APIDetails    `json:"api"`
	AssetResource models.AssetResource `json:"assetResource,omitempty"`
	ProductPlan   models.ProductPlan   `json:"productPlan,omitempty"`
	Quota         models.Quota         `json:"quota,omitempty"`
	Unit          models.Unit          `json:"unit,omitempty"`
	StatusCode    string               `json:"statusCode,omitempty"`
	Status        string               `json:"status,omitempty"`
	Count         int64                `json:"count"`
	Response      ResponseMetrics      `json:"response,omitempty"`
	Observation   ObservationDetails   `json:"observation"`
	EventID       string               `json:"-"`
	StartTime     time.Time            `json:"-"`
}

// GetStartTime - Returns the start time for subscription metric
func (a *APIMetric) GetStartTime() time.Time {
	return a.StartTime
}

// GetType - Returns APIMetric
func (a *APIMetric) GetType() string {
	return "APIMetric"
}

// GetType - Returns APIMetric
func (a *APIMetric) GetEventID() string {
	return a.EventID
}

func (a *APIMetric) GetLogFields() logrus.Fields {
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
	fields = a.Subscription.GetLogFields(fields)
	fields = a.App.GetLogFields(fields)
	fields = a.Product.GetLogFields(fields)
	fields = a.API.GetLogFields(fields)
	fields = a.AssetResource.GetLogFields(fields)
	fields = a.ProductPlan.GetLogFields(fields)
	fields = a.Quota.GetLogFields(fields)
	return fields
}
