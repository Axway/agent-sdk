package apic

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	cache2 "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/api"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
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
	responsesInOrder := []api.MockResponse{}
	numberOfResponses := 20
	for i := 0; i < numberOfResponses; i++ {
		responsesInOrder = append(responsesInOrder, api.MockResponse{
			FileName: "./testdata/agent-details-sr.json",
			RespCode: http.StatusOK,
		},
		)
	}
	mockHTTPClient.SetResponses(responsesInOrder)

	ri := &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			Name:             "test-resource",
			GroupVersionKind: management.APIServiceGVK(),
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

	// with no changes to the subResource, there should be no updates
	ri.CreateHashes()
	bts, err := json.Marshal(ri)
	assert.Nil(t, err)
	err = json.Unmarshal(bts, &ri)
	assert.Nil(t, err)
	err = svcClient.CreateSubResource(ri.ResourceMeta, ri.SubResources)
	assert.Nil(t, err)
	assert.Equal(t, mockHTTPClient.RespCount, 2)

	bts, err = json.Marshal(ri)
	assert.Nil(t, err)
	err = json.Unmarshal(bts, &ri)
	assert.Nil(t, err)
	// with a subResource update, we expect 2 extra updates
	ri.SetSubResource("sub1", "val")
	err = svcClient.CreateSubResource(ri.ResourceMeta, ri.SubResources)
	assert.Nil(t, err)
	assert.Equal(t, mockHTTPClient.RespCount, 4)

	bts, err = json.Marshal(ri)
	assert.Nil(t, err)
	err = json.Unmarshal(bts, &ri)
	assert.Nil(t, err)
	err = svcClient.CreateSubResource(ri.ResourceMeta, map[string]interface{}{
		definitions.XAgentDetails: map[string]interface{}{
			"externalAPIID":   "12345",
			"externalAPIName": "daleapi",
			"createdBy":       "",
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, 4, mockHTTPClient.RespCount)
}

func TestUpdateSpecORCreateResourceInstance(t *testing.T) {
	tests := []struct {
		name            string
		gvk             apiv1.GroupVersionKind
		oldHash         string
		newHash         string
		apiResponses    []api.MockResponse
		expectedAttrVal string
		expectErr       bool
	}{
		{
			name:    "should error with bad response from api call",
			gvk:     management.AccessRequestDefinitionGVK(),
			oldHash: "1234",
			newHash: "1235",
			apiResponses: []api.MockResponse{
				{
					RespCode: http.StatusUnauthorized,
				},
			},
			expectedAttrVal: "existing",
			expectErr:       true,
		},
		{
			name:            "should not update ARD as hash is unchanged",
			gvk:             management.AccessRequestDefinitionGVK(),
			oldHash:         "1234",
			newHash:         "1234",
			apiResponses:    []api.MockResponse{},
			expectedAttrVal: "existing",
			expectErr:       false,
		},
		{
			name:            "should not update CRD as hash is unchanged",
			gvk:             management.CredentialRequestDefinitionGVK(),
			oldHash:         "1234",
			newHash:         "1234",
			apiResponses:    []api.MockResponse{},
			expectedAttrVal: "existing",
			expectErr:       false,
		},
		{
			name:    "should update ARD as hash has changed",
			gvk:     management.AccessRequestDefinitionGVK(),
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
			expectErr: false,
		},
		{
			name:    "should update CRD as hash has changed",
			gvk:     management.CredentialRequestDefinitionGVK(),
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
			expectErr: false,
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

			res := apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{
					Name:             tt.name,
					GroupVersionKind: tt.gvk,
					SubResources: map[string]interface{}{
						defs.XAgentDetails: map[string]interface{}{
							defs.AttrSpecHash: tt.oldHash,
						},
					},
					Attributes: map[string]string{"existing": "existing"},
				},
				Spec: map[string]interface{}{},
			}

			// setup the cachedResources
			switch tt.gvk.Kind {
			case management.AccessRequestDefinitionGVK().Kind:
				svcClient.caches.AddAccessRequestDefinition(&res)
			case management.CredentialRequestDefinitionGVK().Kind:
				svcClient.caches.AddCredentialRequestDefinition(&res)
			}

			newRes := res
			newRes.Attributes = map[string]string{}
			newRes.SubResources = map[string]interface{}{defs.XAgentDetails: map[string]interface{}{defs.AttrSpecHash: tt.newHash}}

			ri, err := svcClient.updateSpecORCreateResourceInstance(&newRes)
			if tt.expectErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.expectedAttrVal, ri.Attributes["existing"])
			}

		})
	}
}

func TestPatchRequest(t *testing.T) {
	createInstance := func(name string) *v1.ResourceInstance {
		inst := createAPIServiceInstance(name, name, "", "", false)
		ri, _ := inst.AsInstance()
		return ri
	}

	tests := []struct {
		name             string
		res              *v1.ResourceInstance
		patchSubResource string
		patches          []map[string]interface{}
		apiResponses     []api.MockResponse
		expectErr        bool
	}{
		{
			name:         "no self link",
			res:          &v1.ResourceInstance{},
			apiResponses: []api.MockResponse{},
			expectErr:    true,
		},
		{
			name:      "no patches",
			res:       createInstance("test"),
			expectErr: false,
		},
		{
			name: "send patch",
			res:  createInstance("test"),
			apiResponses: []api.MockResponse{
				{
					FileName: "./testdata/serviceinstance.json",
					RespCode: http.StatusOK,
				},
			},
			patchSubResource: "source",
			patches: []map[string]interface{}{
				{
					PatchOperation: PatchOpAdd,
					PatchPath:      "/source/compliance",
					PatchValue:     map[string]interface{}{},
				},
			},
			expectErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcClient, mockHTTPClient := GetTestServiceClient()
			cfg := GetTestServiceClientCentralConfiguration(svcClient)
			cfg.Environment = "mockenv"
			cfg.PlatformURL = "http://foo.bar:4080"

			mockHTTPClient.SetResponses(tt.apiResponses)

			ri, err := svcClient.PatchSubResource(tt.res, tt.patchSubResource, tt.patches)
			if tt.expectErr {
				if tt.patches != nil {
					assert.NotNil(t, err)
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, ri)
				if len(tt.patches) > 0 {
					assert.NotEmpty(t, mockHTTPClient.Requests)
					p := make([]map[string]interface{}, 0)
					json.Unmarshal(mockHTTPClient.Requests[0].Body, &p)
					assert.NotEmpty(t, p)
					assert.Equal(t, len(tt.patches)+1, len(p))
					buildObjectTreePatchFound := false
					for _, patch := range p {
						operation := patch[PatchOperation]
						if operation == PatchOpBuildObjectTree {
							buildObjectTreePatchFound = true
							assert.Equal(t, "/"+tt.patchSubResource, patch[PatchPath])
						} else {
							path := patch[PatchPath]
							assert.Equal(t, path, tt.patches[0][PatchPath])
						}
					}
					assert.True(t, buildObjectTreePatchFound)
				}
			}
		})
	}
}
