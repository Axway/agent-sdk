package cache

import (
	"fmt"

	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	"github.com/Axway/agent-sdk/pkg/util"
)

func formatAppForeignKey(appName string) string { return fmt.Sprintf("ManagedApplication:%v", appName) }

func arSecondaryKey(appName, apiID, apiStage, apiVersion string) string {
	return fmt.Sprintf("%s:%s:%s:%s", appName, apiID, apiStage, apiVersion)
}

// AccessRequest cache related methods
func (c *cacheManager) GetAccessRequestCacheKeys() []string {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	return c.accessRequestMap.GetKeys()
}

// IsAccessRequestCacheEnabled returns true if AccessRequests are being cached.
func (c *cacheManager) IsAccessRequestCacheEnabled() bool {
	return c.isAccessRequestCacheEnabled
}

// SetAccessRequestCacheEnabled allows the agent using the SDK to turn AccessRequest caching on or
// off. Traceability agents have it enabled by default; discovery agents must opt in, if they need it.
func (c *cacheManager) SetAccessRequestCacheEnabled(enabled bool) {
	c.isAccessRequestCacheEnabled = enabled
}

// resolveAccessRequestAPIAttributes looks up the APIServiceInstance referenced by ar and returns
// the external API ID, stage, and version reported by the gateway. Empty strings are returned for
// any attribute that can't be resolved.
func (c *cacheManager) resolveAccessRequestAPIAttributes(ar *management.AccessRequest) (apiID, apiStage, apiVersion string) {
	instRef := ar.GetReferenceByGVK(management.APIServiceInstanceGVK())
	instance, _ := c.GetAPIServiceInstanceByID(instRef.ID)
	if instance == nil && ar.Spec.ApiServiceInstance != "" {
		instance, _ = c.GetAPIServiceInstanceByName(ar.Spec.ApiServiceInstance)
	}
	if instance == nil {
		return "", "", ""
	}

	apiID, _ = util.GetAgentDetailsValue(instance, defs.AttrExternalAPIID)
	apiStage, _ = util.GetAgentDetailsValue(instance, defs.AttrExternalAPIStage)
	apiVersion, _ = util.GetAgentDetailsValue(instance, defs.AttrExternalAPIVersion)
	return apiID, apiStage, apiVersion
}

func (c *cacheManager) AddAccessRequest(ri *v1.ResourceInstance) {
	if !c.isAccessRequestCacheEnabled || ri == nil {
		return
	}

	ri = withComputedHashes(ri)
	ar := &management.AccessRequest{}
	if ar.FromInstance(ri) != nil {
		return
	}

	appName := ar.Spec.ManagedApplication
	instID := ar.GetReferenceByGVK(management.APIServiceInstanceGVK()).ID
	apiID, apiStage, apiVersion := c.resolveAccessRequestAPIAttributes(ar)

	secKey := arSecondaryKey(appName, apiID, apiStage, apiVersion)
	formattedAppForeignKey := formatAppForeignKey(appName)

	c.logger.
		WithField("instID", instID).
		WithField("apiID", apiID).
		WithField("apiStage", apiStage).
		WithField("apiVersion", apiVersion).
		WithField("formattedAppForeignKey", formattedAppForeignKey).
		WithField("metadataID", ar.Metadata.ID).
		WithField("secKey", secKey).
		Trace("add access request and set secondary key")

	c.accessRequestMap.SetWithSecondaryKey(ar.Metadata.ID, secKey, ri)
	c.accessRequestMap.SetForeignKey(ar.Metadata.ID, formattedAppForeignKey)

}

func (c *cacheManager) GetAccessRequestByAppAndAPI(appName, remoteAPIID, remoteAPIStage string) *v1.ResourceInstance {
	return c.GetAccessRequestByAppAndAPIStageVersion(appName, remoteAPIID, remoteAPIStage, "")
}

func (c *cacheManager) GetAccessRequestByAppAndAPIStageVersion(appName, remoteAPIID, remoteAPIStage, remoteAPIVersion string) *v1.ResourceInstance {
	c.logger.
		WithField("appName", appName).
		WithField("remoteAPIID", remoteAPIID).
		WithField("remoteAPIStage", remoteAPIStage).
		WithField("remoteAPIVersion", remoteAPIVersion).
		Trace("get access request by app, API stage, and version")

	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	secKey := arSecondaryKey(appName, remoteAPIID, remoteAPIStage, remoteAPIVersion)
	c.logger.WithField("secKey", secKey).Trace("computed access request secondary key for lookup")

	accessRequest, _ := c.accessRequestMap.GetBySecondaryKey(secKey)
	if accessRequest != nil {
		if ri, ok := accessRequest.(*v1.ResourceInstance); ok {
			c.logger.WithField("secKey", secKey).Trace("access request found for secondary key")
			return ri
		}
	}
	c.logger.WithField("secKey", secKey).Trace("no access request found for secondary key")
	return nil
}

func (c *cacheManager) GetAccessRequestsByApp(appName string) []*v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	items, _ := c.accessRequestMap.GetItemsByForeignKey(formatAppForeignKey(appName))

	accessRequests := []*v1.ResourceInstance{}
	for _, item := range items {
		if item != nil {
			if ri, ok := item.GetObject().(*v1.ResourceInstance); ok {
				accessRequests = append(accessRequests, ri)
			}
		}
	}

	return accessRequests
}

// GetAccessRequestsByAPI returns every cached AccessRequest for the given API ID, across
// all managed applications, stages, and versions.
func (c *cacheManager) GetAccessRequestsByAPI(apiID string) []*v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	accessRequests := []*v1.ResourceInstance{}
	for _, key := range c.accessRequestMap.GetKeys() {
		item, _ := c.accessRequestMap.Get(key)
		ri, ok := item.(*v1.ResourceInstance)
		if !ok || ri == nil {
			continue
		}

		ar := &management.AccessRequest{}
		if ar.FromInstance(ri) != nil {
			continue
		}

		id, _, _ := c.resolveAccessRequestAPIAttributes(ar)
		if id == apiID {
			accessRequests = append(accessRequests, withComputedHashes(ri))
		}
	}

	return accessRequests
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

func (c *cacheManager) ListAccessRequests() []*v1.ResourceInstance {
	list := make([]*v1.ResourceInstance, 0)
	for _, key := range c.accessRequestMap.GetKeys() {
		item, _ := c.accessRequestMap.Get(key)
		if v, ok := item.(*v1.ResourceInstance); ok && v != nil {
			list = append(list, v)
		}
	}
	return list
}
