package metric

// Metrics - struct to hold metrics for transaction
type Metrics struct {
	Count            int64   `json:"count"`
	MaxResponseTime  int64   `json:"maxResponseTime"`
	MinResponseTime  int64   `json:"minResponseTime"`
	MeanResponseTime float64 `json:"meanResponseTime"`
}

// StatusMetric - struct to hold metric specific for status code based transactions
type StatusMetric struct {
	StatusCode string `json:"statusCode"`
	APIMetric
}

// APIMetric - struct to hold metric specific for api based transactions
type APIMetric struct {
	APIName       string `json:"apiName"`
	APIID         string `json:"apiID"`
	ObservedStart int64  `json:"observedStart,omitempty"`
	ObservedEnd   int64  `json:"observedEnd,omitempty"`
	Metrics
}

// V4EventDistribution - represents V7 distribution
type V4EventDistribution struct {
	Environment string `json:"environment"`
	Version     string `json:"version"`
}

// V4Event - represents V7 event
type V4Event struct {
	ID           string              `json:"id"`
	Timestamp    int64               `json:"timestamp"`
	Event        string              `json:"event"`
	App          string              `json:"app"` // ORG GUID
	Version      string              `json:"version"`
	Distribution V4EventDistribution `json:"distribution"`
	Data         interface{}         `json:"data"`
}
