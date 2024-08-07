/*
 * This file is automatically generated
 */

package management

import (
	"fmt"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/api/v1"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	m "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

type IdentityProviderSecretMergeFunc func(*m.IdentityProviderSecret, *m.IdentityProviderSecret) (*m.IdentityProviderSecret, error)

// IdentityProviderSecretMerge builds a merge option for an update operation
func IdentityProviderSecretMerge(f IdentityProviderSecretMergeFunc) v1.UpdateOption {
	return v1.Merge(func(prev, new apiv1.Interface) (apiv1.Interface, error) {
		p, n := &m.IdentityProviderSecret{}, &m.IdentityProviderSecret{}

		switch t := prev.(type) {
		case *m.IdentityProviderSecret:
			p = t
		case *apiv1.ResourceInstance:
			err := p.FromInstance(t)
			if err != nil {
				return nil, fmt.Errorf("merge: failed to unserialise prev resource: %w", err)
			}
		default:
			return nil, fmt.Errorf("merge: failed to unserialise prev resource, unxexpected resource type: %T", t)
		}

		switch t := new.(type) {
		case *m.IdentityProviderSecret:
			n = t
		case *apiv1.ResourceInstance:
			err := n.FromInstance(t)
			if err != nil {
				return nil, fmt.Errorf("merge: failed to unserialize new resource: %w", err)
			}
		default:
			return nil, fmt.Errorf("merge: failed to unserialise new resource, unxexpected resource type: %T", t)
		}

		return f(p, n)
	})
}

// IdentityProviderSecretClient - rest client for IdentityProviderSecret resources that have a defined resource scope
type IdentityProviderSecretClient struct {
	client v1.Scoped
}

// UnscopedIdentityProviderSecretClient - rest client for IdentityProviderSecret resources that do not have a defined scope
type UnscopedIdentityProviderSecretClient struct {
	client v1.Unscoped
}

// NewIdentityProviderSecretClient - creates a client that is not scoped to any resource
func NewIdentityProviderSecretClient(c v1.Base) (*UnscopedIdentityProviderSecretClient, error) {

	client, err := c.ForKind(m.IdentityProviderSecretGVK())
	if err != nil {
		return nil, err
	}

	return &UnscopedIdentityProviderSecretClient{client}, nil

}

// WithScope - sets the resource scope for the client
func (c *UnscopedIdentityProviderSecretClient) WithScope(scope string) *IdentityProviderSecretClient {
	return &IdentityProviderSecretClient{
		c.client.WithScope(scope),
	}
}

// Get - gets a resource by name
func (c *UnscopedIdentityProviderSecretClient) Get(name string) (*m.IdentityProviderSecret, error) {
	ri, err := c.client.Get(name)
	if err != nil {
		return nil, err
	}

	service := &m.IdentityProviderSecret{}
	service.FromInstance(ri)

	return service, nil
}

// Update - updates a resource
func (c *UnscopedIdentityProviderSecretClient) Update(res *m.IdentityProviderSecret, opts ...v1.UpdateOption) (*m.IdentityProviderSecret, error) {
	ri, err := res.AsInstance()
	if err != nil {
		return nil, err
	}
	resource, err := c.client.Update(ri, opts...)
	if err != nil {
		return nil, err
	}

	updated := &m.IdentityProviderSecret{}

	// Updates the resource in place
	err = updated.FromInstance(resource)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// List - gets a list of resources
func (c *IdentityProviderSecretClient) List(options ...v1.ListOptions) ([]*m.IdentityProviderSecret, error) {
	riList, err := c.client.List(options...)
	if err != nil {
		return nil, err
	}

	result := make([]*m.IdentityProviderSecret, len(riList))

	for i := range riList {
		result[i] = &m.IdentityProviderSecret{}
		err := result[i].FromInstance(riList[i])
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// Get - gets a resource by name
func (c *IdentityProviderSecretClient) Get(name string) (*m.IdentityProviderSecret, error) {
	ri, err := c.client.Get(name)
	if err != nil {
		return nil, err
	}

	service := &m.IdentityProviderSecret{}
	service.FromInstance(ri)

	return service, nil
}

// Delete - deletes a resource
func (c *IdentityProviderSecretClient) Delete(res *m.IdentityProviderSecret) error {
	ri, err := res.AsInstance()

	if err != nil {
		return err
	}

	return c.client.Delete(ri)
}

// Create - creates a resource
func (c *IdentityProviderSecretClient) Create(res *m.IdentityProviderSecret, opts ...v1.CreateOption) (*m.IdentityProviderSecret, error) {
	ri, err := res.AsInstance()

	if err != nil {
		return nil, err
	}

	cri, err := c.client.Create(ri, opts...)
	if err != nil {
		return nil, err
	}

	created := &m.IdentityProviderSecret{}

	err = created.FromInstance(cri)
	if err != nil {
		return nil, err
	}

	return created, err
}

// Update - updates a resource
func (c *IdentityProviderSecretClient) Update(res *m.IdentityProviderSecret, opts ...v1.UpdateOption) (*m.IdentityProviderSecret, error) {
	ri, err := res.AsInstance()
	if err != nil {
		return nil, err
	}
	resource, err := c.client.Update(ri, opts...)
	if err != nil {
		return nil, err
	}

	updated := &m.IdentityProviderSecret{}

	// Updates the resource in place
	err = updated.FromInstance(resource)
	if err != nil {
		return nil, err
	}

	return updated, nil
}
