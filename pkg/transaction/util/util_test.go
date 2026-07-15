package util

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
)

func TestGetTransactionEventStatus(t *testing.T) {
	assert.Equal(t, "Fail", GetTransactionEventStatus(401))
	assert.Equal(t, "Fail", GetTransactionEventStatus(500))
	assert.Equal(t, "Pass", GetTransactionEventStatus(201))
}

func TestGetTransactionSummaryStatus(t *testing.T) {
	assert.Equal(t, "Success", GetTransactionSummaryStatus(201))
	assert.Equal(t, "Failure", GetTransactionSummaryStatus(404))
	assert.Equal(t, "Exception", GetTransactionSummaryStatus(501))
	assert.Equal(t, "Unknown", GetTransactionSummaryStatus(555))
}

func TestMarshalHeadersAsJSONString(t *testing.T) {
	m := map[string]string{}
	assert.Equal(t, "{}", MarshalHeadersAsJSONString(m))

	m = map[string]string{
		"prop1": "val1",
		"prop2": "val2",
	}
	assert.Equal(t, "{\"prop1\":\"val1\",\"prop2\":\"val2\"}", MarshalHeadersAsJSONString(m))

	m = map[string]string{
		"prop1": "val1",
		"prop2": "",
	}
	assert.Equal(t, "{\"prop1\":\"val1\",\"prop2\":\"\"}", MarshalHeadersAsJSONString(m))

	m = map[string]string{
		"prop1": "aaa\"bbb\"ccc",
	}
	assert.Equal(t, "{\"prop1\":\"aaa\\\"bbb\\\"ccc\"}", MarshalHeadersAsJSONString(m))
}

func TestFormatProxyID(t *testing.T) {
	s := FormatProxyID("foobar")
	assert.Equal(t, SummaryEventProxyIDPrefix+"foobar", s)
}
func TestFormatApplicationID(t *testing.T) {
	s := FormatApplicationID("barfoo")
	assert.Equal(t, SummaryEventApplicationIDPrefix+"barfoo", s)
}

func TestResolveIDWithPrefix(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		inputName   string
		expected    string
		description string
	}{
		{
			name:        "ID with content after prefix",
			id:          "remoteApiId_dwight",
			inputName:   "schrute",
			expected:    "remoteApiId_dwight",
			description: "Should return original ID when it has content after prefix",
		},
		{
			name:        "ID is just prefix, use name",
			id:          "remoteApiId_",
			inputName:   "schrute",
			expected:    "remoteApiName_schrute",
			description: "Should use name with name prefix when ID is just the prefix",
		},
		{
			name:        "ID is empty, use name",
			id:          "",
			inputName:   "schrute",
			expected:    "remoteApiName_schrute",
			description: "Should use name with name prefix when ID is empty",
		},
		{
			name:        "Both ID and name are empty",
			id:          "",
			inputName:   "",
			expected:    "remoteApiId_unknown",
			description: "Should use unknown with prefix when both are empty",
		},
		{
			name:        "ID without prefix",
			id:          "dwight",
			inputName:   "schrute",
			expected:    "dwight",
			description: "Should return original ID when it doesn't start with prefix",
		},
		{
			name:        "Different prefix",
			id:          "differentPrefix_dwight",
			inputName:   "schrute",
			expected:    "differentPrefix_dwight",
			description: "Should return original ID when it has a different prefix",
		},
		{
			name:        "prefixed ID content equals name is not a real ID",
			id:          "remoteApiId_schrute",
			inputName:   "schrute",
			expected:    "remoteApiName_schrute",
			description: "Should use name prefix when the prefixed ID content is identical to the name, since a resolvable ID would not match the display name",
		},
		{
			name:        "prefixed ID content differs from name only by case is still preserved",
			id:          "remoteApiId_Schrute",
			inputName:   "schrute",
			expected:    "remoteApiId_Schrute",
			description: "Should preserve original ID since the comparison is exact and case differences count as distinct content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveIDWithPrefix(tt.id, tt.inputName)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestGetMarketplaceDetails(t *testing.T) {
	tests := map[string]struct {
		ri       *v1.ResourceInstance
		expected *models.MarketplaceReference
	}{
		"nil managed application returns none placeholders": {
			ri:       nil,
			expected: &models.MarketplaceReference{GUID: none, ConsumerOrgID: none},
		},
		"resolved instance with no marketplace data returns none placeholders": {
			ri:       &v1.ResourceInstance{},
			expected: &models.MarketplaceReference{GUID: none, ConsumerOrgID: none},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetMarketplaceDetails(tt.ri))
		})
	}
}
