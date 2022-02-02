package apic

import (
	"net/http"
	"strings"
	"testing"
	"time"

	cache2 "github.com/Axway/agent-sdk/pkg/agent/cache"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/healthcheck"
)

func TestNewClient(t *testing.T) {
	cfg := corecfg.NewCentralConfig(corecfg.DiscoveryAgent)
	client := New(cfg, MockTokenGetter, cache2.NewAgentCacheManager(cfg, false))
	assert.NotNil(t, client)
}

func TestGetEnvironment(t *testing.T) {
	svcClient, mockHTTPClient := GetTestServiceClient()
	cfg := GetTestServiceClientCentralConfiguration(svcClient)
	cfg.Environment = "Environment"
	cfg.Mode = corecfg.PublishToEnvironment
	mockHTTPClient.SetResponse("./testdata/apiserver-environment.json", http.StatusOK)

	env, err := svcClient.GetEnvironment()
	assert.NotNil(t, env)
	assert.Nil(t, err, "An unexpected error was returned from the health check with traceability agent in publishToEnvironment mode")
	assert.NotNil(t, env)
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
	svcClient, _ := GetTestServiceClient()

	tag4Value := "value4"
	tags := map[string]interface{}{"tag1": "value1", "tag2": "", "tag3": "value3", "tag4": &tag4Value}
	result := svcClient.mapToTagsArray(tags)
	assert.Equal(t, 4, len(result))
	assert.True(t, arrContains(result, "tag1_value1"))
	assert.True(t, arrContains(result, "tag2"))
	assert.False(t, arrContains(result, "bar"))

	cfg := GetTestServiceClientCentralConfiguration(svcClient)
	cfg.TagsToPublish = "bar"
	result = svcClient.mapToTagsArray(tags)
	assert.Equal(t, 5, len(result))
	assert.True(t, arrContains(result, "tag1_value1"))
	assert.True(t, arrContains(result, "tag2"))
	assert.True(t, arrContains(result, "bar"))
}

func TestGetUserEmailAddress(t *testing.T) {
	svcClient, mockHTTPClient := GetTestServiceClient()

	cfg := GetTestServiceClientCentralConfiguration(svcClient)
	cfg.Environment = "Environment"
	cfg.PlatformURL = "http://foo.bar:4080"

	// Test DiscoveryAgent, PublishToEnvironment
	mockHTTPClient.SetResponses([]api.MockResponse{
		{
			FileName: "./testdata/userinfo.json",
			RespCode: http.StatusOK,
		},
	})

	addr, err := svcClient.GetUserEmailAddress("b0433b7f-ac38-4d29-8a64-cf645c99b99f")
	assert.Nil(t, err)
	assert.Equal(t, "joe@axway.com", addr)

	// test a failure
	mockHTTPClient.SetResponses([]api.MockResponse{
		{FileName: "./testdata/userinfoerror.json",
			RespCode:  http.StatusNotFound,
			ErrString: "Resource Not Found",
		},
	})

	addr, err = svcClient.GetUserEmailAddress("b0433b7f-ac38-4d29-8a64-cf645c99b99g")
	assert.NotNil(t, err)
	assert.Equal(t, "", addr)
}

func TestGetUserName(t *testing.T) {
	svcClient, mockHTTPClient := GetTestServiceClient()

	cfg := GetTestServiceClientCentralConfiguration(svcClient)
	cfg.Environment = "Environment"
	cfg.PlatformURL = "http://foo.bar:4080"

	// Test DiscoveryAgent, PublishToEnvironment
	mockHTTPClient.SetResponses([]api.MockResponse{
		{
			FileName: "./testdata/userinfo.json",
			RespCode: http.StatusOK,
		},
	})

	userName, err := svcClient.GetUserName("b0433b7f-ac38-4d29-8a64-cf645c99b99f")
	assert.Nil(t, err)
	assert.Equal(t, "Dale Feldick", userName)

	// test a failure
	mockHTTPClient.SetResponses([]api.MockResponse{
		{FileName: "./testdata/userinfoerror.json",
			RespCode:  http.StatusNotFound,
			ErrString: "Resource Not Found",
		},
	})

	userName, err = svcClient.GetUserName("b0433b7f-ac38-4d29-8a64-cf645c99b99g")
	assert.NotNil(t, err)
	assert.Equal(t, "", userName)
}

func TestHealthCheck(t *testing.T) {
	svcClient, mockHTTPClient := GetTestServiceClient()
	requester := svcClient.tokenRequester

	// swap out mock for a real tokenRequester
	svcClient.tokenRequester = auth.NewPlatformTokenGetter("", "", "", "", "", "", 1*time.Second)

	// failure
	status := svcClient.Healthcheck("Client Test")
	assert.Equal(t, status.Result, healthcheck.FAIL)
	assert.True(t, strings.Contains(status.Details, "error getting authentication token"))

	svcClient.tokenRequester = requester

	// failure
	status = svcClient.Healthcheck("Client Test")
	assert.Equal(t, status.Result, healthcheck.FAIL)
	assert.True(t, strings.Contains(status.Details, "unexpected end"))

	// success
	responses := []api.MockResponse{
		{FileName: "./testdata/apiserver-environment.json", RespCode: http.StatusOK},
	}
	mockHTTPClient.SetResponses(responses)
	status = svcClient.Healthcheck("Client Test")
	assert.Equal(t, status.Result, healthcheck.OK)
}
