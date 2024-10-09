package metric

import (
	"time"

	"github.com/Axway/agent-sdk/pkg/transaction/models"
	"github.com/sirupsen/logrus"
)

// use a variable for this to fake it for tests
var now = time.Now

const (
	schemaPath              = "/api/v1/report.schema.json"
	metricEvent             = "api.transaction.status.metric"
	messageKey              = "message"
	metricKey               = "metric"
	metricFlow              = "api-central-metric"
	metricRetries           = "metricRetry"
	retries                 = "retries"
	lighthouseTransactions  = "Transactions"
	lighthouseVolume        = "Volume"
	transactionCountMetric  = "transaction.count"
	transactionVolumeMetric = "transaction.volume"
	unknown                 = "unknown"
)

type transactionContext struct {
	APIDetails models.APIDetails
	AppDetails models.AppDetails
	UniqueKey  string // status code or custom metric name
}

// Detail - holds the details for computing metrics
// for API and consumer subscriptions
type Detail struct {
	APIDetails models.APIDetails
	AppDetails models.AppDetails
	StatusCode string
	Duration   int64
	Bytes      int64
}

type MetricDetail struct {
	APIDetails  models.APIDetails
	AppDetails  models.AppDetails
	StatusCode  string
	Count       int64
	Response    ResponseMetrics
	Observation ObservationDetails
}

type CustomMetricDetail struct {
	APIDetails  models.APIDetails
	AppDetails  models.AppDetails
	UnitDetails models.Unit
	Count       int64
	Observation ObservationDetails
}

// ResponseMetrics - Holds metrics API response
type ResponseMetrics struct {
	Max int64   `json:"max"`
	Min int64   `json:"min"`
	Avg float64 `json:"avg"`
}

// ObservationDetails - Holds start and end timestamp for interval
type ObservationDetails struct {
	Start int64 `json:"start,omitempty"`
	End   int64 `json:"end,omitempty"`
}

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

// cachedMetric - struct to hold metric specific that gets cached and used for agent recovery
type cachedMetric struct {
	Subscription  *models.Subscription  `json:"subscription,omitempty"`
	App           *models.AppDetails    `json:"app,omitempty"`
	Product       *models.Product       `json:"product,omitempty"`
	API           *models.APIDetails    `json:"api,omitempty"`
	AssetResource *models.AssetResource `json:"assetResource,omitempty"`
	ProductPlan   *models.ProductPlan   `json:"productPlan,omitempty"`
	Quota         *models.Quota         `json:"quota,omitempty"`
	Unit          *models.Unit          `json:"unit,omitempty"`
	StatusCode    string                `json:"statusCode,omitempty"`
	Count         int64                 `json:"count"`
	Values        []int64               `json:"values,omitempty"`
	StartTime     time.Time             `json:"startTime"`
}

// V4EventDistribution - represents V4 distribution
type V4EventDistribution struct {
	Environment string `json:"environment"`
	Version     string `json:"version"`
}

// V4Session - represents V4 session
type V4Session struct {
	ID string `json:"id"`
}

// V4Data - Interface for representing the metric data
type V4Data interface {
	GetStartTime() time.Time
	GetType() string
	GetEventID() string
	GetLogFields() logrus.Fields
}

// V4Event - represents V7 event
type V4Event struct {
	ID           string               `json:"id"`
	Timestamp    int64                `json:"timestamp"`
	Event        string               `json:"event"`
	App          string               `json:"app,omitempty"` // ORG GUID
	Version      string               `json:"version"`
	Distribution *V4EventDistribution `json:"distribution"`
	Data         V4Data               `json:"data"`
	Session      *V4Session           `json:"session,omitempty"`
}

func (v V4Event) getLogFields() logrus.Fields {
	return v.Data.GetLogFields()
}

// UsageReport -Lighthouse Usage report
type UsageReport struct {
	Product string                 `json:"product"`
	Usage   map[string]int64       `json:"usage"`
	Meta    map[string]interface{} `json:"meta"`
}

// UsageEvent -Lighthouse Usage Event
type UsageEvent struct {
	OrgGUID     string                 `json:"-"`
	EnvID       string                 `json:"envId"`
	Timestamp   ISO8601Time            `json:"timestamp"`
	Granularity int                    `json:"granularity"`
	SchemaID    string                 `json:"schemaId"`
	Report      map[string]UsageReport `json:"report"`
	Meta        map[string]interface{} `json:"meta"`
}

type UsageResponse struct {
	Success     bool   `json:"success"`
	Description string `json:"description"`
	StatusCode  int    `json:"code"`
}

// Data - struct for data to report as API Metrics
type Data struct {
	APIDetails models.APIDetails
	StatusCode string
	Duration   int64
	UsageBytes int64
	AppDetails models.AppDetails
	TeamName   string
}

// AppUsage - struct to hold metric specific for app usage
type AppUsage struct {
	App   models.AppDetails `json:"app"`
	Count int64             `json:"count"`
}

// ISO8601 - time format
const (
	ISO8601 = "2006-01-02T15:04:00Z07:00"
)

// ISO8601Time - time
type ISO8601Time time.Time

// UnmarshalJSON - unmarshal json for time
func (t *ISO8601Time) UnmarshalJSON(bytes []byte) error {
	tt, err := time.Parse(`"`+ISO8601+`"`, string(bytes))
	if err != nil {
		return err
	}
	*t = ISO8601Time(tt)
	return nil
}

// MarshalJSON -
func (t ISO8601Time) MarshalJSON() ([]byte, error) {
	tt := time.Time(t)

	b := make([]byte, 0, len(ISO8601)+2)
	b = append(b, '"')
	b = tt.AppendFormat(b, ISO8601)
	b = append(b, '"')
	return b, nil
}
