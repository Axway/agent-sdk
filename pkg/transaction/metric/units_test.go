package metric

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/transaction/models"
)

func TestUnitTypeString(t *testing.T) {
	cases := map[UnitType]string{
		TransactionUnit:      "transactions",
		CostUSDUnit:          "cost-usd",
		LLMRequests:          "llm-requests",
		LLMInputTokens:       "llm-inputtokens",
		LLMOutputTokens:      "llm-outputtokens",
		LLMCachedInputTokens: "llm-cachedinputtokens",
		LLMTotalTokens:       "llm-totaltokens",
	}
	for unit, want := range cases {
		assert.Equal(t, want, unit.String())
	}
}

func TestUnitTypeRoundTrip(t *testing.T) {
	units := []UnitType{
		TransactionUnit, CostUSDUnit, LLMRequests, LLMInputTokens,
		LLMOutputTokens, LLMCachedInputTokens, LLMTotalTokens,
	}
	for _, u := range units {
		assert.Equal(t, u, StringToUnitType(u.String()), "round trip failed for %s", u.String())
	}
}

func TestUnitsMarshalJSON(t *testing.T) {
	t.Run("transactions and custom units are both emitted", func(t *testing.T) {
		u := Units{
			Transactions: &Transactions{
				UnitCount: UnitCount{Count: 3},
				Status:    "Success",
			},
			CustomUnits: map[string]*UnitCount{
				"llm-inputtokens": {Count: 120},
				"cost-usd":        {Count: 10},
			},
		}

		b, err := json.Marshal(u)
		assert.NoError(t, err)

		var out map[string]json.RawMessage
		assert.NoError(t, json.Unmarshal(b, &out))

		assert.Contains(t, out, "transactions")
		assert.Contains(t, out, "llm-inputtokens")
		assert.Contains(t, out, "cost-usd")

		var txn Transactions
		assert.NoError(t, json.Unmarshal(out["transactions"], &txn))
		assert.Equal(t, int64(3), txn.Count)
		assert.Equal(t, "Success", txn.Status)

		var tokens UnitCount
		assert.NoError(t, json.Unmarshal(out["llm-inputtokens"], &tokens))
		assert.Equal(t, int64(120), tokens.Count)
	})

	t.Run("custom units only still emits a null transactions key", func(t *testing.T) {
		u := Units{
			CustomUnits: map[string]*UnitCount{"llm-requests": {Count: 1}},
		}

		b, err := json.Marshal(u)
		assert.NoError(t, err)

		var out map[string]json.RawMessage
		assert.NoError(t, json.Unmarshal(b, &out))

		assert.Contains(t, out, "transactions")
		assert.Equal(t, "null", string(out["transactions"]))
		assert.Contains(t, out, "llm-requests")
	})
}

func TestUnitCountMarshalsQuota(t *testing.T) {
	uc := UnitCount{Count: 5, Quota: &models.ResourceReference{ID: "quota-1"}}
	b, err := json.Marshal(uc)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"count":5,"quota":{"id":"quota-1"}}`, string(b))
}
