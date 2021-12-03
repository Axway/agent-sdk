package agent

import (
	"net/http"
	"sync"
	"testing"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/stretchr/testify/assert"
)

func setupCache(externalAPIID, externalAPIName string) (*v1.ResourceInstance, *v1.ResourceInstance) {
	svc := &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{
				ID: "svc-" + externalAPIID,
			},
			Attributes: map[string]string{
				apic.AttrExternalAPIID:         externalAPIID,
				apic.AttrExternalAPIPrimaryKey: "primary-" + externalAPIID,
				apic.AttrExternalAPIName:       externalAPIName,
			},
		},
	}
	instance := &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{
				ID: "instance-" + externalAPIID,
			},
			Attributes: map[string]string{
				apic.AttrExternalAPIID:   externalAPIID,
				apic.AttrExternalAPIName: externalAPIName,
			},
		},
	}

	agent.instanceMap = cache.New()
	agent.apiMap = cache.New()
	agent.instanceMap.Set(instance.Metadata.ID, instance)
	agent.apiMap.SetWithSecondaryKey("primaryKey-"+externalAPIID, externalAPIID, svc)
	return svc, instance
}

func setupAPICClient(mockResponse []api.MockResponse) {
	client, httpClient := apic.GetTestServiceClient()
	httpClient.SetResponses(mockResponse)
	agent.apicClient = client
}

func setupAPIValidator(apiValidation bool) {
	agent.apiValidator = func(apiID, stageName string) bool {
		return apiValidation
	}
}

func TestValidatorAPIExistsOnDataplane(t *testing.T) {
	// Setup
	instanceValidator := newInstanceValidator(&sync.Mutex{}, true)
	setupCache("12345", "test")
	setupAPIValidator(true)
	instanceValidator.Execute()
	i, err := agent.instanceMap.Get("instance-12345")
	assert.Nil(t, err)
	assert.NotNil(t, i)

	s, err := agent.apiMap.Get("primaryKey-12345")
	assert.Nil(t, err)
	assert.NotNil(t, s)
}

func TestValidatorAPIDoesExistsDeleteService(t *testing.T) {
	// Setup
	instanceValidator := newInstanceValidator(&sync.Mutex{}, true)
	setupCache("12345", "test")
	setupAPICClient([]api.MockResponse{
		{
			FileName: "../apic/testdata/apiservice-list.json", // for call to get the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "../apic/testdata/apiservice-list.json", // for call to get the consumer instances
			RespCode: http.StatusOK,
		},
		{
			RespCode: http.StatusNoContent, // delete service
		},
	})
	setupAPIValidator(false)
	instanceValidator.Execute()
	i, err := agent.instanceMap.Get("instance-12345")
	assert.NotNil(t, err)
	assert.Nil(t, i)

	s, err := agent.apiMap.Get("primaryKey-12345")
	assert.NotNil(t, err)
	assert.Nil(t, s)
}

func TestValidatorAPIDoesExistsDeleteInstance(t *testing.T) {
	// Setup
	instanceValidator := newInstanceValidator(&sync.Mutex{}, true)
	setupCache("12345", "test")
	setupAPICClient([]api.MockResponse{
		{
			RespCode: http.StatusNoContent, // for call to get the consumer instances
		},
		{
			RespCode: http.StatusNoContent, // delete instance
		},
	})
	setupAPIValidator(false)

	instanceValidator.Execute()
	i, err := agent.instanceMap.Get("instance-12345")
	assert.NotNil(t, err)
	assert.Nil(t, i)

	s, err := agent.apiMap.Get("primaryKey-12345")
	assert.Nil(t, err)
	assert.NotNil(t, s)
}
