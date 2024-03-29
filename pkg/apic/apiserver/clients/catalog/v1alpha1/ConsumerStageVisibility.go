/*
 * This file is automatically generated
 */

package catalog

import (
	"fmt"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/api/v1"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	m "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
)

type ConsumerStageVisibilityMergeFunc func(*m.ConsumerStageVisibility, *m.ConsumerStageVisibility) (*m.ConsumerStageVisibility, error)

// ConsumerStageVisibilityMerge builds a merge option for an update operation
func ConsumerStageVisibilityMerge(f ConsumerStageVisibilityMergeFunc) v1.UpdateOption {
	return v1.Merge(func(prev, new apiv1.Interface) (apiv1.Interface, error) {
		p, n := &m.ConsumerStageVisibility{}, &m.ConsumerStageVisibility{}

		switch t := prev.(type) {
		case *m.ConsumerStageVisibility:
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
		case *m.ConsumerStageVisibility:
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

// ConsumerStageVisibilityClient - rest client for ConsumerStageVisibility resources that have a defined resource scope
type ConsumerStageVisibilityClient struct {
	client v1.Scoped
}

// UnscopedConsumerStageVisibilityClient - rest client for ConsumerStageVisibility resources that do not have a defined scope
type UnscopedConsumerStageVisibilityClient struct {
	client v1.Unscoped
}

// NewConsumerStageVisibilityClient - creates a client that is not scoped to any resource
func NewConsumerStageVisibilityClient(c v1.Base) (*UnscopedConsumerStageVisibilityClient, error) {

	client, err := c.ForKind(m.ConsumerStageVisibilityGVK())
	if err != nil {
		return nil, err
	}

	return &UnscopedConsumerStageVisibilityClient{client}, nil

}

// WithScope - sets the resource scope for the client
func (c *UnscopedConsumerStageVisibilityClient) WithScope(scope string) *ConsumerStageVisibilityClient {
	return &ConsumerStageVisibilityClient{
		c.client.WithScope(scope),
	}
}

// Get - gets a resource by name
func (c *UnscopedConsumerStageVisibilityClient) Get(name string) (*m.ConsumerStageVisibility, error) {
	ri, err := c.client.Get(name)
	if err != nil {
		return nil, err
	}

	service := &m.ConsumerStageVisibility{}
	service.FromInstance(ri)

	return service, nil
}

// Update - updates a resource
func (c *UnscopedConsumerStageVisibilityClient) Update(res *m.ConsumerStageVisibility, opts ...v1.UpdateOption) (*m.ConsumerStageVisibility, error) {
	ri, err := res.AsInstance()
	if err != nil {
		return nil, err
	}
	resource, err := c.client.Update(ri, opts...)
	if err != nil {
		return nil, err
	}

	updated := &m.ConsumerStageVisibility{}

	// Updates the resource in place
	err = updated.FromInstance(resource)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// List - gets a list of resources
func (c *ConsumerStageVisibilityClient) List(options ...v1.ListOptions) ([]*m.ConsumerStageVisibility, error) {
	riList, err := c.client.List(options...)
	if err != nil {
		return nil, err
	}

	result := make([]*m.ConsumerStageVisibility, len(riList))

	for i := range riList {
		result[i] = &m.ConsumerStageVisibility{}
		err := result[i].FromInstance(riList[i])
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// Get - gets a resource by name
func (c *ConsumerStageVisibilityClient) Get(name string) (*m.ConsumerStageVisibility, error) {
	ri, err := c.client.Get(name)
	if err != nil {
		return nil, err
	}

	service := &m.ConsumerStageVisibility{}
	service.FromInstance(ri)

	return service, nil
}

// Delete - deletes a resource
func (c *ConsumerStageVisibilityClient) Delete(res *m.ConsumerStageVisibility) error {
	ri, err := res.AsInstance()

	if err != nil {
		return err
	}

	return c.client.Delete(ri)
}

// Create - creates a resource
func (c *ConsumerStageVisibilityClient) Create(res *m.ConsumerStageVisibility, opts ...v1.CreateOption) (*m.ConsumerStageVisibility, error) {
	ri, err := res.AsInstance()

	if err != nil {
		return nil, err
	}

	cri, err := c.client.Create(ri, opts...)
	if err != nil {
		return nil, err
	}

	created := &m.ConsumerStageVisibility{}

	err = created.FromInstance(cri)
	if err != nil {
		return nil, err
	}

	return created, err
}

// Update - updates a resource
func (c *ConsumerStageVisibilityClient) Update(res *m.ConsumerStageVisibility, opts ...v1.UpdateOption) (*m.ConsumerStageVisibility, error) {
	ri, err := res.AsInstance()
	if err != nil {
		return nil, err
	}
	resource, err := c.client.Update(ri, opts...)
	if err != nil {
		return nil, err
	}

	updated := &m.ConsumerStageVisibility{}

	// Updates the resource in place
	err = updated.FromInstance(resource)
	if err != nil {
		return nil, err
	}

	return updated, nil
}
