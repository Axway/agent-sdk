package metric

import "time"

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
)

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

// APIDetails - Holds the api details
type APIDetails struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Revision int    `json:"revision"`
}

// APIMetric - struct to hold metric specific for status code based API transactions
type APIMetric struct {
	API         APIDetails         `json:"api"`
	StatusCode  string             `json:"statusCode"`
	Status      string             `json:"status"`
	Count       int64              `json:"count"`
	Response    ResponseMetrics    `json:"response"`
	Observation ObservationDetails `json:"observation"`
	StartTime   time.Time          `json:"-"`
}

// cachedMetric - struct to hold metric specific that gets cached and used for agent recovery
type cachedMetric struct {
	API        APIDetails `json:"api"`
	StatusCode string     `json:"statusCode"`
	Count      int64      `json:"count"`
	Values     []int64    `json:"values"`
	StartTime  time.Time  `json:"startTime"`
}

// V4EventDistribution - represents V7 distribution
type V4EventDistribution struct {
	Environment string `json:"environment"`
	Version     string `json:"version"`
}

// V4Event - represents V7 event
type V4Event struct {
	ID           string               `json:"id"`
	Timestamp    int64                `json:"timestamp"`
	Event        string               `json:"event"`
	App          string               `json:"app"` // ORG GUID
	Version      string               `json:"version"`
	Distribution *V4EventDistribution `json:"distribution"`
	Data         *APIMetric           `json:"data"`
}

// LighthouseUsageReport -Lighthouse Usage report
type LighthouseUsageReport struct {
	Product string                 `json:"product"`
	Usage   map[string]int64       `json:"usage"`
	Meta    map[string]interface{} `json:"meta"`
}

// LighthouseUsageEvent -Lighthouse Usage Event
type LighthouseUsageEvent struct {
	OrgGUID     string                           `json:"-"`
	EnvID       string                           `json:"envId"`
	Timestamp   ISO8601Time                      `json:"timestamp"`
	Granularity int                              `json:"granularity"`
	SchemaID    string                           `json:"schemaId"`
	Report      map[string]LighthouseUsageReport `json:"report"`
	Meta        map[string]interface{}           `json:"meta"`
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
