package apic

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	apicClient "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
	corecfg "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/healthcheck"
)

type mockResponse struct {
	fileName  string
	respCode  int
	errString string
}

type mockHTTPClient struct {
	Client
	respCount int
	responses []mockResponse
}

func setupMocks(c *ServiceClient) *mockHTTPClient {
	mockClient := mockHTTPClient{
		respCount: 0,
		responses: []mockResponse{
			{
				respCode: http.StatusOK,
			},
		},
	}
	c.apiClient = &mockClient
	c.tokenRequester = MockTokenGetter
	return &mockClient
}

// Send - send the http request and returns the API Response
func (c *mockHTTPClient) Send(request apicClient.Request) (*apicClient.Response, error) {
	responseFile, _ := os.Open(c.responses[c.respCount].fileName) // APIC Environments
	dat, _ := ioutil.ReadAll(responseFile)

	response := apicClient.Response{
		Code:    c.responses[c.respCount].respCode,
		Body:    dat,
		Headers: map[string][]string{},
	}

	var err error
	if c.responses[c.respCount].errString != "" {
		err = fmt.Errorf(c.responses[c.respCount].errString)
	}
	c.respCount++
	return &response, err
}

func TestCheckAPIServerHealth(t *testing.T) {
	c, cfg := createServiceClient(nil)
	mockClient := setupMocks(c)
	cfg.Environment = "Environment"
	cfg.Mode = corecfg.PublishToEnvironment

	// mockClient := mockHTTPClient{
	// 	respCount: 0,
	// 	responses: []mockResponse{
	// 		{
	// 			fileName: "./testdata/apic-environment.json",
	// 			respCode: http.StatusOK,
	// 		},
	// 		{
	// 			fileName: "./testdata/apic-team-notfound.json",
	// 			respCode: http.StatusOK,
	// 		},
	// 	},
	// }
	mockClient.responses = []mockResponse{
		{
			fileName: "./testdata/apic-environment.json",
			respCode: http.StatusOK,
		},
		{
			fileName: "./testdata/apic-team-notfound.json",
			respCode: http.StatusOK,
		},
	}

	// c.apiClient = &mockClient
	// c.tokenRequester = MockTokenGetter

	// Test DiscoveryAgent, PublishToEnvironment and with team not found specified
	err := c.checkAPIServerHealth()
	assert.NotNil(t, err, "Expecting error to be returned from the health check with discovery agent in publishToEnvironment mode for invalid team name")

	// Test Team found
	mockClient.respCount = 0
	mockClient.responses = []mockResponse{
		{
			fileName: "./testdata/apiserver-environment.json",
			respCode: http.StatusOK,
		},
		{
			fileName: "./testdata/apic-team.json",
			respCode: http.StatusOK,
		},
	}
	c.cfg.SetEnvironmentID("")
	err = c.checkAPIServerHealth()
	assert.Nil(t, err, "An unexpected error was returned from the health check with discovery agent in publishToEnvironment mode")

	// Test TraceabilityAgent, publishToEnvironment
	cfg.AgentType = corecfg.TraceabilityAgent
	cfg.Mode = corecfg.PublishToEnvironment
	mockClient.respCount = 0
	err = c.checkAPIServerHealth()
	assert.Nil(t, err, "An unexpected error was returned from the health check with traceability agent in publishToEnvironment mode")
	assert.Equal(t, "e4e085bf70638a1d0170639297610000", cfg.GetEnvironmentID(), "The EnvironmentID was not set correctly, Traceability and publishToEnvironment mode")

	// pass in 2 urls to test 2nd path to getting environment
	responses := []mockResponse{
		{fileName: "./testdata/apiserver-environment.json", respCode: http.StatusBadRequest},
		{fileName: "./testdata/apic-environment.json", respCode: http.StatusOK},
		{fileName: "./testdata/apic-team.json", respCode: http.StatusOK},
	}
	mockClient.respCount = 0
	mockClient.responses = responses
	c.cfg.SetEnvironmentID("")
	err = c.checkAPIServerHealth()
	assert.Nil(t, err, "An unexpected error was returned from the health check with discovery agent in publishToEnvironment mode")
	assert.Equal(t, "e4e084b66fcf325a016fcf54677b0001", cfg.GetEnvironmentID(), "The EnvironmentID was not set correctly, Traceability and publishToEnvironment mode")
}

func TestNewClientWithTLSConfig(t *testing.T) {
	tlsCfg := corecfg.NewTLSConfig()
	client, cfg := createServiceClient(tlsCfg)
	assert.NotNil(t, client)
	assert.NotNil(t, cfg)
}

func arrContains(arr []string, s string) bool {
	for _, n := range arr {
		if n == s {
			return true
		}
	}
	return false
}

func TestMapTagsToArray(t *testing.T) {
	client, cfg := createServiceClient(nil)

	tag4Value := "value4"
	tags := map[string]interface{}{"tag1": "value1", "tag2": "", "tag3": "value3", "tag4": &tag4Value}
	result := client.mapToTagsArray(tags)
	assert.Equal(t, 4, len(result))
	assert.True(t, arrContains(result, "tag1_value1"))
	assert.True(t, arrContains(result, "tag2"))
	assert.False(t, arrContains(result, "bar"))

	cfg.TagsToPublish = "bar"
	result = client.mapToTagsArray(tags)
	assert.Equal(t, 5, len(result))
	assert.True(t, arrContains(result, "tag1_value1"))
	assert.True(t, arrContains(result, "tag2"))
	assert.True(t, arrContains(result, "bar"))

}

func TestGetUserEmailAddress(t *testing.T) {
	client, cfg := createServiceClient(nil)
	mockClient := setupMocks(client)

	cfg.PlatformURL = "http://foo.bar:4080"
	cfg.Environment = "Environment"

	// Test DiscoveryAgent, PublishToEnvironment
	mockClient.respCount = 0
	mockClient.responses[0].fileName = "./testdata/userinfo.json"

	addr, err := client.GetUserEmailAddress("b0433b7f-ac38-4d29-8a64-cf645c99b99f")
	assert.Nil(t, err)
	assert.Equal(t, "joe@axway.com", addr)

	// test a failure
	mockClient.respCount = 0
	mockClient.responses[0].fileName = "./testdata/userinfoerror.json"
	mockClient.responses[0].respCode = http.StatusNotFound
	mockClient.responses[0].errString = "Resource Not Found"
	addr, err = client.GetUserEmailAddress("b0433b7f-ac38-4d29-8a64-cf645c99b99g")
	assert.NotNil(t, err)
	assert.Equal(t, "", addr)
}

func TestHealthCheck(t *testing.T) {
	client, cfg := createServiceClient(nil)

	// failure
	status := client.healthcheck("Client Test")
	assert.Equal(t, status.Result, healthcheck.FAIL)
	assert.True(t, strings.Contains(status.Details, "error getting authentication token"))

	mockClient := setupMocks(client)

	// failure
	status = client.healthcheck("Client Test")
	assert.Equal(t, status.Result, healthcheck.FAIL)
	assert.True(t, strings.Contains(status.Details, "unexpected end"))

	// success
	responses := []mockResponse{
		{fileName: "./testdata/apiserver-environment.json", respCode: http.StatusOK},
		{fileName: "./testdata/apic-team.json", respCode: http.StatusOK},
	}
	mockClient.respCount = 0
	mockClient.responses = responses
	status = client.healthcheck("Client Test")
	assert.Equal(t, status.Result, healthcheck.OK)
	assert.Equal(t, "e4e085bf70638a1d0170639297610000", cfg.GetEnvironmentID())
}
