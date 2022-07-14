package cache

import (
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/util"
)

// AccessRequest cache related methods
func (c *cacheManager) GetAccessRequestCacheKeys() []string {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	return c.accessRequestMap.GetKeys()
}

func (c *cacheManager) AddAccessRequest(ri *v1.ResourceInstance) {
	if ri == nil {
		return
	}

	ar := &management.AccessRequest{}
	if ar.FromInstance(ri) != nil {
		return
	}

	appName := ar.Spec.ManagedApplication
	instID := ""
	for _, ref := range ar.Metadata.References {
		if ref.Name == ar.Spec.ApiServiceInstance {
			instID = ref.ID
			break
		}
	}

	instance, _ := c.GetAPIServiceInstanceByID(instID)
	apiID := ""
	apiStage := ""
	if instance != nil {
		apiID, _ = util.GetAgentDetailsValue(instance, defs.AttrExternalAPIID)
		apiStage, _ = util.GetAgentDetailsValue(instance, defs.AttrExternalAPIStage)
	}

	c.accessRequestMap.SetWithSecondaryKey(ar.Metadata.ID, appName+":"+apiID+":"+apiStage, ri)
}

func (c *cacheManager) GetAccessRequestByAppAndAPI(appName, remoteAPIID, remoteAPIStage string) *v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	accessRequest, _ := c.accessRequestMap.GetBySecondaryKey(appName + ":" + remoteAPIID + ":" + remoteAPIStage)
	if accessRequest != nil {
		if ri, ok := accessRequest.(*v1.ResourceInstance); ok {
			return ri
		}
	}
	return nil
}

func (c *cacheManager) GetAccessRequest(id string) *v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	accessRequest, _ := c.accessRequestMap.Get(id)
	if accessRequest != nil {
		if ri, ok := accessRequest.(*v1.ResourceInstance); ok {
			return ri
		}
	}
	return nil
}

func (c *cacheManager) DeleteAccessRequest(id string) error {
	return c.accessRequestMap.Delete(id)
}
