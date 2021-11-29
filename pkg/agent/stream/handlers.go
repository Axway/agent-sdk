package stream

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/apic"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const (
	apiService         = "APIService"
	apiServiceInstance = "APIServiceInstance"
	category           = "Category"
)

type apiSvcHandler struct {
	apis cache.Cache
}

// NewAPISvcHandler creates a Handler for API Services.
func NewAPISvcHandler(cache cache.Cache) Handler {
	return &apiSvcHandler{
		apis: cache,
	}
}

func (h apiSvcHandler) handle(action proto.Event_Type, resource *apiv1.ResourceInstance) error {
	if resource.Kind != apiService {
		return nil
	}

	id, ok := resource.Attributes[apic.AttrExternalAPIID]
	if !ok {
		return fmt.Errorf("%s not found on resource api service %s", apic.AttrExternalAPIID, resource.Name)
	}

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		externalAPIName := resource.Attributes[apic.AttrExternalAPIName]
		primaryKey, ok := resource.Attributes[apic.AttrExternalAPIPrimaryKey]
		if !ok {
			return h.apis.SetWithSecondaryKey(id, externalAPIName, resource)
		}

		return h.apis.SetWithSecondaryKey(primaryKey, externalAPIName, resource)
	}

	if action == proto.Event_DELETED {
		return h.apis.Delete(id)
	}

	return nil
}

type instanceHandler struct {
	instances cache.Cache
}

// NewInstanceHandler creates a Handler for API Service Instances.
func NewInstanceHandler(cache cache.Cache) Handler {
	return &instanceHandler{
		instances: cache,
	}
}

func (h instanceHandler) handle(action proto.Event_Type, resource *apiv1.ResourceInstance) error {
	if resource.Kind != apiServiceInstance {
		return nil
	}

	key := resource.Metadata.ID
	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		return h.instances.Set(key, resource)
	}

	if action == proto.Event_DELETED {
		return h.instances.Delete(key)
	}

	return nil
}

type categoryHandler struct {
	categories cache.Cache
}

// NewCategoryHandler creates a Handler for Categories.
func NewCategoryHandler(cache cache.Cache) Handler {
	return &categoryHandler{
		categories: cache,
	}
}

func (c categoryHandler) handle(action proto.Event_Type, resource *apiv1.ResourceInstance) error {
	if resource.Kind != category {
		return nil
	}

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		return c.categories.SetWithSecondaryKey(resource.Name, resource.Title, resource)
	}

	if action == proto.Event_DELETED {
		return c.categories.Delete(resource.Name)
	}

	return nil
}
