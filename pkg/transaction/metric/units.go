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
	Duration int64            `json:"duration,omitempty"`
	Response *ResponseMetrics `json:"response,omitempty"`
	Status   string           `json:"status,omitempty"`
}

type Units struct {
	Transactions *Transactions         `json:"transactions,omitempty"`
	Units        map[string]*UnitCount `json:"-"`
}

func (u Units) MarshalJSON() ([]byte, error) {
	// Add the fields from the struct to a new map
	result := map[string]interface{}{
		TransactionUnit.String(): u.Transactions,
	}

	// Add the non-transaction units to the map
	for k, cu := range u.Units {
		result[k] = cu
	}

	// return the marshaled map
	return json.Marshal(result)
}

type UnitType int32

const (
	TransactionUnit UnitType = iota
	// CostUSDUnit - reported cost is expressed as an integer number of milli-USD (USD * 1000).
	CostUSDUnit
	LLMRequests
	LLMInputTokens
	LLMOutputTokens
	LLMCachedInputTokens
	LLMTotalTokens
)

func (u UnitType) String() string {
	return map[UnitType]string{
		TransactionUnit:      "transactions",
		CostUSDUnit:          "cost-usd",
		LLMRequests:          "llm-requests",
		LLMInputTokens:       "llm-input-tokens",
		LLMOutputTokens:      "llm-output-tokens",
		LLMCachedInputTokens: "llm-cached-input-tokens",
		LLMTotalTokens:       "llm-total-tokens",
	}[u]
}

func StringToUnitType(in string) UnitType {
	return map[string]UnitType{
		"transactions":            TransactionUnit,
		"cost-usd":                CostUSDUnit,
		"llm-requests":            LLMRequests,
		"llm-input-tokens":        LLMInputTokens,
		"llm-output-tokens":       LLMOutputTokens,
		"llm-cached-input-tokens": LLMCachedInputTokens,
		"llm-total-tokens":        LLMTotalTokens,
	}[in]
}
