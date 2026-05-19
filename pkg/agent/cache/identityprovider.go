package cache

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

func (c *cacheManager) AddIdentityProvider(ri *v1.ResourceInstance) {
	if ri == nil {
		return
	}

	idp := &management.IdentityProvider{}
	if idp.FromInstance(ri) != nil {
		return
	}

	defer c.setCacheUpdated(true)
	c.logger.
		WithField("id", idp.Metadata.ID).
		WithField("name", idp.Name).
		Trace("add identity provider")

	c.idpMap.Set(idp.Metadata.ID, ri)
	c.idpMap.SetSecondaryKey(idp.Metadata.ID, idp.Name)
}

func (c *cacheManager) DeleteIdentityProvider(id string) error {
	defer c.setCacheUpdated(true)
	return c.idpMap.Delete(id)
}

func (c *cacheManager) GetIdentityProviderByID(id string) *v1.ResourceInstance {
	ri, _ := c.idpMap.Get(id)
	if ri != nil {
		idp, ok := ri.(*v1.ResourceInstance)
		if ok {
			return idp
		}
	}
	return nil
}

func (c *cacheManager) GetIdentityProviderByName(name string) *v1.ResourceInstance {
	ri, _ := c.idpMap.GetBySecondaryKey(name)
	if ri != nil {
		idp, ok := ri.(*v1.ResourceInstance)
		if ok {
			return idp
		}
	}
	return nil
}

func (c *cacheManager) AddIdentityProviderMetadata(ri *v1.ResourceInstance) {
	if ri == nil {
		return
	}

	idpMetadata := &management.IdentityProviderMetadata{}
	if idpMetadata.FromInstance(ri) != nil {
		return
	}

	defer c.setCacheUpdated(true)
	c.logger.
		WithField("id", idpMetadata.Metadata.ID).
		WithField("name", idpMetadata.Name).
		WithField("idp", idpMetadata.Metadata.Scope.Name).
		Trace("add identity provider metadata")

	c.idpMetadataMap.Set(idpMetadata.Metadata.ID, ri)
	c.idpMetadataMap.SetSecondaryKey(idpMetadata.Metadata.ID, idpMetadata.Spec.TokenEndpoint)
}

func (c *cacheManager) DeleteIdentityProviderMetadata(id string) error {
	defer c.setCacheUpdated(true)
	return c.idpMetadataMap.Delete(id)
}

func (c *cacheManager) GetIdentityProviderMetadataByTokenUrl(tokenURL string) *v1.ResourceInstance {
	ri, _ := c.idpMetadataMap.GetBySecondaryKey(tokenURL)
	if ri != nil {
		idp, ok := ri.(*v1.ResourceInstance)
		if ok {
			return idp
		}
	}
	return nil
}
