package cache

import (
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"

	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/util"
)

// AccessRequest cache related methods
func (c *cacheManager) GetAccessRequestCacheKeys() []string {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	return c.managedApplicationMap.GetKeys()
}

func (c *cacheManager) AddAccessRequest(ar *mv1.AccessRequest) {
	if ar == nil {
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

	c.accessRequestMap.SetWithSecondaryKey(ar.Metadata.ID, appName+":"+apiID+":"+apiStage, ar)
}

func (c *cacheManager) GetAccessRequestByAppAndAPI(appName, remoteAPIID, remoteAPIStage string) *mv1.AccessRequest {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	accessRequest, _ := c.accessRequestMap.GetBySecondaryKey(appName + ":" + remoteAPIID + ":" + remoteAPIStage)
	if accessRequest != nil {
		ri, ok := accessRequest.(*mv1.AccessRequest)
		if ok {
			return ri
		}
	}
	return nil
}

func (c *cacheManager) GetAccessRequest(id string) *mv1.AccessRequest {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	accessRequest, _ := c.accessRequestMap.Get(id)
	if accessRequest != nil {
		ri, ok := accessRequest.(*mv1.AccessRequest)
		if ok {
			return ri
		}
	}
	return nil
}

func (c *cacheManager) DeleteAccessRequest(id string) error {
	return c.accessRequestMap.Delete(id)
}
