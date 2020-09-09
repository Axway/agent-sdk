package apic

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
	corecfg "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/healthcheck"
)

func TestCheckAPIServerHealth(t *testing.T) {
	svcClient, mockHTTPClient := GetTestServiceClient()
	// mockClient := setupMocks(c)
	// cfg.Environment = "Environment"
	// cfg.Mode = corecfg.PublishToEnvironment
	mockHTTPClient.SetResponses([]api.MockResponse{
		{
			FileName: "./testdata/apic-environment.json", // this for call to getEnvironment
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/apic-team-notfound.json", // this for call to getTeamByName
			RespCode: http.StatusOK,
		},
	})

	// Test DiscoveryAgent, PublishToEnvironment and with team not found specified
	err := svcClient.checkAPIServerHealth()
	assert.NotNil(t, err, "Expecting error to be returned from the health check with discovery agent in publishToEnvironment mode for invalid team name")

	// Test Team found
	mockHTTPClient.SetResponses([]api.MockResponse{
		{
			FileName: "./testdata/apiserver-environment.json", // this for call to getEnvironment
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/apic-team.json", // this for call to getTeamByName
			RespCode: http.StatusOK,
		},
	})
	svcClient.cfg.SetEnvironmentID("")
	err = svcClient.checkAPIServerHealth()
	assert.Nil(t, err, "An unexpected error was returned from the health check with discovery agent in publishToEnvironment mode")

	// Test TraceabilityAgent, publishToEnvironment
	centralConfig := svcClient.cfg.(*corecfg.CentralConfiguration)
	centralConfig.AgentType = corecfg.TraceabilityAgent
	centralConfig.Mode = corecfg.PublishToEnvironment
	err = svcClient.checkAPIServerHealth()
	assert.Nil(t, err, "An unexpected error was returned from the health check with traceability agent in publishToEnvironment mode")
	assert.Equal(t, "e4e085bf70638a1d0170639297610000", centralConfig.GetEnvironmentID(), "The EnvironmentID was not set correctly, Traceability and publishToEnvironment mode")
}

// TODO func TestNewClientWithTLSConfig(t *testing.T) {
// 	tlsCfg := corecfg.NewTLSConfig()
// 	client, mockHTTPClient := GetTestServiceClient(tlsCfg)

// 	assert.NotNil(t, client)
// 	assert.NotNil(t, mockHTTPClient)
// }

func arrContains(arr []string, s string) bool {
	for _, n := range arr {
		if n == s {
			return true
		}
	}
	return false
}

func TestMapTagsToArray(t *testing.T) {
	svcClient, _ := GetTestServiceClient()

	tag4Value := "value4"
	tags := map[string]interface{}{"tag1": "value1", "tag2": "", "tag3": "value3", "tag4": &tag4Value}
	result := svcClient.mapToTagsArray(tags)
	assert.Equal(t, 4, len(result))
	assert.True(t, arrContains(result, "tag1_value1"))
	assert.True(t, arrContains(result, "tag2"))
	assert.False(t, arrContains(result, "bar"))

	// cfg.TagsToPublish = "bar"
	result = svcClient.mapToTagsArray(tags)
	assert.Equal(t, 5, len(result))
	assert.True(t, arrContains(result, "tag1_value1"))
	assert.True(t, arrContains(result, "tag2"))
	assert.True(t, arrContains(result, "bar"))

}

func TestGetUserEmailAddress(t *testing.T) {
	client, mockHTTPClient := GetTestServiceClient()

	// cfg.PlatformURL = "http://foo.bar:4080"
	// cfg.Environment = "Environment"

	// Test DiscoveryAgent, PublishToEnvironment
	mockHTTPClient.SetResponses([]api.MockResponse{
		{FileName: "./testdata/userinfo.json"},
	})

	addr, err := client.GetUserEmailAddress("b0433b7f-ac38-4d29-8a64-cf645c99b99f")
	assert.Nil(t, err)
	assert.Equal(t, "joe@axway.com", addr)

	// test a failure
	mockHTTPClient.SetResponses([]api.MockResponse{
		{FileName: "./testdata/userinfoerror.json",
			RespCode:  http.StatusNotFound,
			ErrString: "Resource Not Found",
		},
	})

	addr, err = client.GetUserEmailAddress("b0433b7f-ac38-4d29-8a64-cf645c99b99g")
	assert.NotNil(t, err)
	assert.Equal(t, "", addr)
}

func TestHealthCheck(t *testing.T) {
	client, mockHTTPClient := GetTestServiceClient()

	// failure
	status := client.healthcheck("Client Test")
	assert.Equal(t, status.Result, healthcheck.FAIL)
	assert.True(t, strings.Contains(status.Details, "error getting authentication token"))

	// mockClient := setupMocks(client)

	// failure
	status = client.healthcheck("Client Test")
	assert.Equal(t, status.Result, healthcheck.FAIL)
	assert.True(t, strings.Contains(status.Details, "unexpected end"))

	// success
	responses := []api.MockResponse{
		{FileName: "./testdata/apiserver-environment.json", RespCode: http.StatusOK},
		{FileName: "./testdata/apic-team.json", RespCode: http.StatusOK},
	}
	mockHTTPClient.SetResponses(responses)
	status = client.healthcheck("Client Test")
	assert.Equal(t, status.Result, healthcheck.OK)
	// assert.Equal(t, "e4e085bf70638a1d0170639297610000", cfg.GetEnvironmentID())
}
