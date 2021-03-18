package metric

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
	Name string `json:"name"`
	ID   string `json:"id"`
}

// APIMetric - struct to hold metric specific for status code based API transactions
type APIMetric struct {
	API         APIDetails         `json:"api"`
	StatusCode  string             `json:"statusCode"`
	Count       int64              `json:"count"`
	Response    ResponseMetrics    `json:"response"`
	Observation ObservationDetails `json:"observation"`
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
