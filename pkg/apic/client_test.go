package apic

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	apicClient "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/api"
	corecfg "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/config"
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
	c, cfg := createServiceClient()
	cfg.Environment = "Environment"
	mockClient := mockHTTPClient{
		respCount: 0,
		responses: []mockResponse{
			{
				fileName: "./testdata/apic-environment.json",
				respCode: http.StatusOK,
			},
		},
	}
	c.apiClient = &mockClient
	c.tokenRequester = &mockTokenGetter{}

	// Test DiscoveryAgent, PublishToCatalog
	cfg.AgentType = corecfg.DiscoveryAgent
	cfg.Mode = corecfg.PublishToCatalog
	err := c.checkCatalogHealth()
	assert.Nil(t, err, "An unexpected error was returned from the health check with discovery agent in publishToCatalog mode")

	// Test DiscoveryAgent, PublishToEnvironment
	mockClient.respCount = 0
	mockClient.responses[0].fileName = "./testdata/apiserver-environment.json"
	cfg.Mode = corecfg.PublishToEnvironment
	err = c.checkAPIServerHealth()
	assert.Nil(t, err, "An unexpected error was returned from the health check with discovery agent in publishToEnvironment mode")

	// Test TraceabilityAgent, publishToCatalog
	cfg.AgentType = corecfg.TraceabilityAgent
	cfg.Mode = corecfg.PublishToCatalog
	mockClient.respCount = 0
	mockClient.responses = []mockResponse{
		{
			fileName: "./testdata/apiserver-environment-notfound.json",
			respCode: http.StatusNotFound,
		},
		{
			fileName: "./testdata/apic-environment.json",
			respCode: http.StatusOK,
		},
	}
	err = c.checkAPIServerHealth()
	assert.Nil(t, err, "An unexpected error was returned from the health check with traceability agent in publishToCatalog mode")
	assert.Equal(t, "e4e084b66fcf325a016fcf54677b0001", cfg.GetEnvironmentID(), "The EnvironmentID was not set correctly, Traceability and publishToCatalog mode")

	// Test TraceabilityAgent, publishToEnvironment
	cfg.AgentType = corecfg.TraceabilityAgent
	cfg.Mode = corecfg.PublishToEnvironment
	mockClient.respCount = 0
	mockClient.responses = []mockResponse{
		{
			fileName: "./testdata/apiserver-environment.json",
			respCode: http.StatusOK,
		},
	}
	err = c.checkAPIServerHealth()
	assert.Nil(t, err, "An unexpected error was returned from the health check with traceability agent in publishToEnvironment mode")
	assert.Equal(t, "e4e085bf70638a1d0170639297610000", cfg.GetEnvironmentID(), "The EnvironmentID was not set correctly, Traceability and publishToEnvironment mode")
}
