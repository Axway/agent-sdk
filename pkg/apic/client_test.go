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
			mockResponse{
				fileName: "./testdata/apic-environment.json",
				respCode: http.StatusOK,
			},
		},
	}
	c.apiClient = &mockClient
	c.tokenRequester = &mockTokenGetter{}

	// Test DiscoveryAgent, disconnected
	cfg.AgentType = corecfg.DiscoveryAgent
	cfg.Mode = corecfg.Disconnected
	err := c.checkAPIServerHealth()
	assert.Nil(t, err, "An unexpected error was returned from the health check with discovery agent in disconnected mode")

	// Test DiscoveryAgent, connected
	mockClient.respCount = 0
	mockClient.responses[0].fileName = "./testdata/apiserver-environment.json"
	cfg.Mode = corecfg.Connected
	err = c.checkAPIServerHealth()
	assert.Nil(t, err, "An unexpected error was returned from the health check with discovery agent in connected mode")

	// Test TraceabilityAgent, disconnected
	cfg.AgentType = corecfg.TraceabilityAgent
	cfg.Mode = corecfg.Disconnected
	mockClient.respCount = 0
	mockClient.responses = []mockResponse{
		mockResponse{
			fileName: "./testdata/apiserver-environment-notfound.json",
			respCode: http.StatusNotFound,
		},
		mockResponse{
			fileName: "./testdata/apic-environment.json",
			respCode: http.StatusOK,
		},
	}
	err = c.checkAPIServerHealth()
	assert.Nil(t, err, "An unexpected error was returned from the health check with traceability agent in disconnected mode")
	assert.Equal(t, "e4e084b66fcf325a016fcf54677b0001", cfg.GetEnvironmentID(), "The EnvironmentID was not set correctly, Traceability and disconnected mode")

	// Test TraceabilityAgent, connected
	cfg.AgentType = corecfg.TraceabilityAgent
	cfg.Mode = corecfg.Connected
	mockClient.respCount = 0
	mockClient.responses = []mockResponse{
		mockResponse{
			fileName: "./testdata/apiserver-environment.json",
			respCode: http.StatusOK,
		},
	}
	err = c.checkAPIServerHealth()
	assert.Nil(t, err, "An unexpected error was returned from the health check with traceability agent in connected mode")
	assert.Equal(t, "e4e085bf70638a1d0170639297610000", cfg.GetEnvironmentID(), "The EnvironmentID was not set correctly, Traceability and connected mode")
}
