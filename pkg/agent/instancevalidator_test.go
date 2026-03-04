package agent

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func setupCache(externalAPIID, externalAPIName string) (*v1.ResourceInstance, *v1.ResourceInstance) {
	svc := &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Name: "service",
			Metadata: v1.Metadata{
				ID: "svc-" + externalAPIID,
			},
			SubResources: map[string]interface{}{
				definitions.XAgentDetails: map[string]interface{}{
					definitions.AttrExternalAPIID:         externalAPIID,
					definitions.AttrExternalAPIPrimaryKey: "primary-" + externalAPIID,
					definitions.AttrExternalAPIName:       externalAPIName,
				},
			},
		},
	}
	instance := &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Name: "instance",
			Metadata: v1.Metadata{
				ID: "instance-" + externalAPIID,
			},
			SubResources: map[string]interface{}{
				definitions.XAgentDetails: map[string]interface{}{
					definitions.AttrExternalAPIID:         externalAPIID,
					definitions.AttrExternalAPIPrimaryKey: "primary-" + externalAPIID,
					definitions.AttrExternalAPIName:       externalAPIName,
				},
			},
		},
	}

	agent.cacheManager = agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	agent.cacheManager.AddAPIService(svc)
	agent.cacheManager.AddAPIServiceInstance(instance)
	return svc, instance
}

func setupAPICClient(mockResponse []api.MockResponse) {
	client, httpClient := apic.GetTestServiceClient()
	httpClient.SetResponses(mockResponse)
	agent.apicClient = client
}

func setupAPIValidator(apiValidation bool) {
	setAPIValidator(func(apiID, stageName string) bool {
		return apiValidation
	})
}

func TestValidatorAPIExistsOnDataplane(t *testing.T) {
	// Setup
	instanceValidator := newInstanceValidator()
	setupCache("12345", "test")
	setupAPIValidator(true)
	instanceValidator.Execute()
	i, err := agent.cacheManager.GetAPIServiceInstanceByID("instance-12345")
	assert.Nil(t, err)
	assert.NotNil(t, i)

	s := agent.cacheManager.GetAPIServiceWithPrimaryKey("primary-12345")
	assert.NotNil(t, s)
}
