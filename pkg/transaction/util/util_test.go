package util

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
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

func TestStripSummaryEventPrefix(t *testing.T) {
	tests := map[string]struct {
		apiID    string
		expected string
	}{
		"real ID prefix stripped": {
			apiID:    SummaryEventProxyIDPrefix + "dwight",
			expected: "dwight",
		},
		"name-fallback prefix stripped": {
			apiID:    SummaryEventAPINamePrefix + "schrute",
			expected: "schrute",
		},
		"no prefix, returned as-is": {
			apiID:    "dwight",
			expected: "dwight",
		},
		"empty string": {
			apiID:    "",
			expected: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, StripSummaryEventPrefix(tc.apiID))
		})
	}
}

func TestStripApplicationIDPrefix(t *testing.T) {
	tests := map[string]struct {
		appID    string
		expected string
	}{
		"app-ID prefix stripped": {
			appID:    SummaryEventApplicationIDPrefix + "dwight",
			expected: "dwight",
		},
		"no prefix, returned as-is": {
			appID:    "dwight",
			expected: "dwight",
		},
		"empty string": {
			appID:    "",
			expected: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, StripApplicationIDPrefix(tc.appID))
		})
	}
}

func marketplaceAppRI(marketplaceName string, owner *v1.Owner) *v1.ResourceInstance {
	app := management.NewManagedApplication("mp-app", "env1")
	app.Marketplace = management.ManagedApplicationMarketplace{
		Name:     marketplaceName,
		Resource: management.ManagedApplicationMarketplaceResource{Owner: owner},
	}
	ri, _ := app.AsInstance()
	return ri
}

func TestGetMarketplaceDetails(t *testing.T) {
	const (
		testMarketplaceGUID  = "marketplace-guid-1"
		testConsumerOrgID    = "consumer-org-1"
		testConsumerTeamGUID = "consumer-team-1"
	)

	// "marketplace" as a string instead of an object triggers a real parse failure.
	malformedInstance := &v1.ResourceInstance{}
	err := json.Unmarshal([]byte(`{"marketplace":"not-an-object"}`), malformedInstance)
	assert.NoError(t, err)

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
		"malformed instance returns unknown guid since marketplace context could not be determined": {
			ri:       malformedInstance,
			expected: &models.MarketplaceReference{GUID: unknown, ConsumerOrgID: none},
		},
		"resolved marketplace name and consumer org resolve to real values": {
			ri: marketplaceAppRI(testMarketplaceGUID, &v1.Owner{
				ID:           testConsumerTeamGUID,
				Organization: v1.Organization{ID: testConsumerOrgID},
			}),
			expected: &models.MarketplaceReference{
				GUID:           testMarketplaceGUID,
				ConsumerOrgID:  testConsumerOrgID,
				ConsumerTeamID: testConsumerTeamGUID,
			},
		},
		"marketplace name resolved but owner absent leaves consumer org as none": {
			ri:       marketplaceAppRI(testMarketplaceGUID, nil),
			expected: &models.MarketplaceReference{GUID: testMarketplaceGUID, ConsumerOrgID: none},
		},
		"owner present without an organization id leaves consumer org as none": {
			ri: marketplaceAppRI(testMarketplaceGUID, &v1.Owner{ID: testConsumerTeamGUID}),
			expected: &models.MarketplaceReference{
				GUID:           testMarketplaceGUID,
				ConsumerOrgID:  none,
				ConsumerTeamID: testConsumerTeamGUID,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetMarketplaceDetails(tt.ri))
		})
	}
}
