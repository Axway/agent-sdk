package apic

import (
	"net/http"
	"strings"
	"testing"
	"time"

	cache2 "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/api"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/stretchr/testify/assert"
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
	result := mapToTagsArray(tags, svcClient.cfg.GetTagsToPublish())
	assert.Equal(t, 4, len(result))
	assert.True(t, arrContains(result, "tag1_value1"))
	assert.True(t, arrContains(result, "tag2"))
	assert.False(t, arrContains(result, "bar"))

	cfg := GetTestServiceClientCentralConfiguration(svcClient)
	cfg.TagsToPublish = "bar"
	result = mapToTagsArray(tags, cfg.GetTagsToPublish())
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
	svcClient.tokenRequester = auth.NewPlatformTokenGetter("", "", "", "", "", "", "", 1*time.Second)

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

func TestCreateSubResource(t *testing.T) {
	svcClient, mockHTTPClient := GetTestServiceClient()
	cfg := GetTestServiceClientCentralConfiguration(svcClient)
	cfg.Environment = "mockenv"
	cfg.PlatformURL = "http://foo.bar:4080"

	// There should be one request for each sub resource of the ResourceInstance
	mockHTTPClient.SetResponses([]api.MockResponse{
		{
			FileName: "./testdata/agent-details-sr.json",
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/agent-details-sr.json",
			RespCode: http.StatusOK,
		},
	})

	ri := &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Name:             "test-resource",
			GroupVersionKind: mv1.APIServiceGVK(),
			SubResources: map[string]interface{}{
				defs.XAgentDetails: map[string]interface{}{
					"externalAPIID":   "12345",
					"externalAPIName": "daleapi",
					"createdBy":       "",
				},
				"abc": map[string]interface{}{
					"123": "132",
				},
			},
		},
	}

	err := svcClient.CreateSubResource(ri.ResourceMeta, ri.SubResources)
	assert.Nil(t, err)
}

func TestUpdateSpecORCreateResourceInstance(t *testing.T) {
	tests := []struct {
		name           string
		gvk            v1.GroupVersionKind
		oldHash        string
		newHash        string
		apiResponses   []api.MockResponse
		expectedTagVal string
		expectErr      bool
	}{
		{
			name:    "should error with bad response from api call",
			gvk:     mv1.AccessRequestDefinitionGVK(),
			oldHash: "1234",
			newHash: "1235",
			apiResponses: []api.MockResponse{
				{
					RespCode: http.StatusUnauthorized,
				},
			},
			expectedTagVal: "existing",
			expectErr:      true,
		},
		{
			name:           "should not update ARD as hash is unchanged",
			gvk:            mv1.AccessRequestDefinitionGVK(),
			oldHash:        "1234",
			newHash:        "1234",
			apiResponses:   []api.MockResponse{},
			expectedTagVal: "existing",
			expectErr:      false,
		},
		{
			name:           "should not update CRD as hash is unchanged",
			gvk:            mv1.CredentialRequestDefinitionGVK(),
			oldHash:        "1234",
			newHash:        "1234",
			apiResponses:   []api.MockResponse{},
			expectedTagVal: "existing",
			expectErr:      false,
		},
		{
			name:    "should update ARD as hash has changed",
			gvk:     mv1.AccessRequestDefinitionGVK(),
			oldHash: "1234",
			newHash: "5234",
			apiResponses: []api.MockResponse{
				{
					FileName: "./testdata/apiservice.json",
					RespCode: http.StatusOK,
				},
				{
					FileName: "./testdata/apiservice.json",
					RespCode: http.StatusOK,
				},
			},
			expectedTagVal: "prod",
			expectErr:      false,
		},
		{
			name:    "should update CRD as hash has changed",
			gvk:     mv1.CredentialRequestDefinitionGVK(),
			oldHash: "1234",
			newHash: "5234",
			apiResponses: []api.MockResponse{
				{
					FileName: "./testdata/apiservice.json",
					RespCode: http.StatusOK,
				},
				{
					FileName: "./testdata/apiservice.json",
					RespCode: http.StatusOK,
				},
			},
			expectedTagVal: "prod",
			expectErr:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcClient, mockHTTPClient := GetTestServiceClient()
			cfg := GetTestServiceClientCentralConfiguration(svcClient)
			cfg.Environment = "mockenv"
			cfg.PlatformURL = "http://foo.bar:4080"

			// There should be one request for each sub resource of the ResourceInstance
			mockHTTPClient.SetResponses(tt.apiResponses)

			res := v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:             tt.name,
					GroupVersionKind: tt.gvk,
					SubResources: map[string]interface{}{
						defs.XAgentDetails: map[string]interface{}{
							defs.AttrSpecHash: tt.oldHash,
						},
					},
					Tags: []string{"existing"},
				},
				Spec: map[string]interface{}{},
			}

			// setup the cachedResources
			switch tt.gvk.Kind {
			case mv1.AccessRequestDefinitionGVK().Kind:
				svcClient.caches.AddAccessRequestDefinition(&res)
			case mv1.CredentialRequestDefinitionGVK().Kind:
				svcClient.caches.AddCredentialRequestDefinition(&res)
			}

			newRes := res
			newRes.Tags = []string{}
			newRes.SubResources = map[string]interface{}{defs.XAgentDetails: map[string]interface{}{defs.AttrSpecHash: tt.newHash}}

			ri, err := svcClient.updateSpecORCreateResourceInstance(&newRes)
			if tt.expectErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.expectedTagVal, ri.Tags[0])
			}

		})
	}
}
