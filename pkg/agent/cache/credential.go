package cache

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

// Credential cache related methods
func (c *cacheManager) GetCredentialCacheKeys() []string {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	return c.credentialMap.GetKeys()
}

func (c *cacheManager) AddCredential(ri *v1.ResourceInstance) {
	if ri == nil {
		return
	}

	ar := &mv1.Credential{}
	if ar.FromInstance(ri) != nil {
		return
	}

	c.credentialMap.Set(ar.Metadata.ID, ri)
}

func (c *cacheManager) GetCredential(id string) *v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	credential, _ := c.credentialMap.Get(id)
	if credential != nil {
		if ri, ok := credential.(*v1.ResourceInstance); ok {
			return ri
		}
	}
	return nil
}

func (c *cacheManager) DeleteCredential(id string) error {
	return c.credentialMap.Delete(id)
}
