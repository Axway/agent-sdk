package metric

import (
	"encoding/json"

	"github.com/Axway/agent-sdk/pkg/transaction/models"
)

type UnitCount struct {
	Count int64                     `json:"count"`
	Quota *models.ResourceReference `json:"quota,omitempty"`
}

type Transactions struct {
	UnitCount
	Response *ResponseMetrics `json:"response,omitempty"`
	Status   string           `json:"status,omitempty"`
}

type Units struct {
	Transactions *Transactions         `json:"transactions,omitempty"`
	CustomUnits  map[string]*UnitCount `json:"-"`
}

func (u Units) MarshalJSON() ([]byte, error) {
	// Add the fields from the struct to a new map
	result := map[string]interface{}{
		"transactions": u.Transactions,
	}

	// Add the custom units to the map
	for k, cu := range u.CustomUnits {
		result[k] = cu
	}

	// return the marshaled map
	return json.Marshal(result)
}
