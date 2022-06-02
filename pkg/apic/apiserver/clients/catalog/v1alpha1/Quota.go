/*
 * This file is automatically generated
 */

package v1alpha1

import (
	"fmt"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/api/v1"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	m "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
)

type QuotaMergeFunc func(*m.Quota, *m.Quota) (*m.Quota, error)

// QuotaMerge builds a merge option for an update operation
func QuotaMerge(f QuotaMergeFunc) v1.UpdateOption {
	return v1.Merge(func(prev, new apiv1.Interface) (apiv1.Interface, error) {
		p, n := &m.Quota{}, &m.Quota{}

		switch t := prev.(type) {
		case *m.Quota:
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
		case *m.Quota:
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

// QuotaClient - rest client for Quota resources that have a defined resource scope
type QuotaClient struct {
	client v1.Scoped
}

// UnscopedQuotaClient - rest client for Quota resources that do not have a defined scope
type UnscopedQuotaClient struct {
	client v1.Unscoped
}

// NewQuotaClient - creates a client that is not scoped to any resource
func NewQuotaClient(c v1.Base) (*UnscopedQuotaClient, error) {

	client, err := c.ForKind(m.QuotaGVK())
	if err != nil {
		return nil, err
	}

	return &UnscopedQuotaClient{client}, nil

}

// WithScope - sets the resource scope for the client
func (c *UnscopedQuotaClient) WithScope(scope string) *QuotaClient {
	return &QuotaClient{
		c.client.WithScope(scope),
	}
}

// Get - gets a resource by name
func (c *UnscopedQuotaClient) Get(name string) (*m.Quota, error) {
	ri, err := c.client.Get(name)
	if err != nil {
		return nil, err
	}

	service := &m.Quota{}
	service.FromInstance(ri)

	return service, nil
}

// Update - updates a resource
func (c *UnscopedQuotaClient) Update(res *m.Quota, opts ...v1.UpdateOption) (*m.Quota, error) {
	ri, err := res.AsInstance()
	if err != nil {
		return nil, err
	}
	resource, err := c.client.Update(ri, opts...)
	if err != nil {
		return nil, err
	}

	updated := &m.Quota{}

	// Updates the resource in place
	err = updated.FromInstance(resource)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// List - gets a list of resources
func (c *QuotaClient) List(options ...v1.ListOptions) ([]*m.Quota, error) {
	riList, err := c.client.List(options...)
	if err != nil {
		return nil, err
	}

	result := make([]*m.Quota, len(riList))

	for i := range riList {
		result[i] = &m.Quota{}
		err := result[i].FromInstance(riList[i])
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// Get - gets a resource by name
func (c *QuotaClient) Get(name string) (*m.Quota, error) {
	ri, err := c.client.Get(name)
	if err != nil {
		return nil, err
	}

	service := &m.Quota{}
	service.FromInstance(ri)

	return service, nil
}

// Delete - deletes a resource
func (c *QuotaClient) Delete(res *m.Quota) error {
	ri, err := res.AsInstance()

	if err != nil {
		return err
	}

	return c.client.Delete(ri)
}

// Create - creates a resource
func (c *QuotaClient) Create(res *m.Quota, opts ...v1.CreateOption) (*m.Quota, error) {
	ri, err := res.AsInstance()

	if err != nil {
		return nil, err
	}

	cri, err := c.client.Create(ri, opts...)
	if err != nil {
		return nil, err
	}

	created := &m.Quota{}

	err = created.FromInstance(cri)
	if err != nil {
		return nil, err
	}

	return created, err
}

// Update - updates a resource
func (c *QuotaClient) Update(res *m.Quota, opts ...v1.UpdateOption) (*m.Quota, error) {
	ri, err := res.AsInstance()
	if err != nil {
		return nil, err
	}
	resource, err := c.client.Update(ri, opts...)
	if err != nil {
		return nil, err
	}

	updated := &m.Quota{}

	// Updates the resource in place
	err = updated.FromInstance(resource)
	if err != nil {
		return nil, err
	}

	return updated, nil
}