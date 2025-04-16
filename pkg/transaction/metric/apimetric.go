package metric

import (
	"time"

	"github.com/Axway/agent-sdk/pkg/transaction/models"
	"github.com/sirupsen/logrus"
)

// APIMetric - struct to hold metric aggregated for subscription,application,api,statuscode
type APIMetric struct {
	Subscription  models.Subscription       `json:"subscription,omitempty"`
	App           models.AppDetails         `json:"application,omitempty"`
	Product       models.Product            `json:"product,omitempty"`
	API           models.APIDetails         `json:"api"`
	AssetResource models.AssetResource      `json:"assetResource,omitempty"`
	ProductPlan   models.ProductPlan        `json:"productPlan,omitempty"`
	Quota         models.Quota              `json:"quota,omitempty"`
	StatusCode    string                    `json:"statusCode,omitempty"`
	Status        string                    `json:"status,omitempty"`
	Count         int64                     `json:"count"`
	Response      ResponseMetrics           `json:"response,omitempty"`
	Observation   models.ObservationDetails `json:"observation"`
	EventID       string                    `json:"-"`
	StartTime     time.Time                 `json:"-"`
	Unit          *models.Unit              `json:"unit,omitempty"`
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

func (a *APIMetric) GetSubscriptionID() string {
	return a.Subscription.ID
}

func (a *APIMetric) GetAppInfo() (string, string) {
	return a.App.ID, a.App.ConsumerOrgID
}

func (a *APIMetric) GetProductInfo() (string, string) {
	return a.Product.ID, a.Product.VersionID
}

func (a *APIMetric) GetAPIInfo() (string, string) {
	return a.API.ID, a.API.Name
}

func (a *APIMetric) GetAssetResourceID() string {
	return a.AssetResource.ID
}

func (a *APIMetric) GetProductPlanID() string {
	return a.ProductPlan.ID
}

func (a *APIMetric) GetStatus() string {
	return a.Status
}

func (a *APIMetric) GetCount() int64 {
	return a.Count
}

func (a *APIMetric) GetResponseMetrics() *ResponseMetrics {
	return &a.Response
}

func (a *APIMetric) GetQuotaID() string {
	return a.Quota.ID
}

func (a *APIMetric) GetObservation() *models.ObservationDetails {
	return &a.Observation
}
