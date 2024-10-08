/*
 * This file is automatically generated
 */

package definitions

import (
	"fmt"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/api/v1"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	m "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/definitions/v1alpha1"
)

type ComponentMergeFunc func(*m.Component, *m.Component) (*m.Component, error)

// ComponentMerge builds a merge option for an update operation
func ComponentMerge(f ComponentMergeFunc) v1.UpdateOption {
	return v1.Merge(func(prev, new apiv1.Interface) (apiv1.Interface, error) {
		p, n := &m.Component{}, &m.Component{}

		switch t := prev.(type) {
		case *m.Component:
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
		case *m.Component:
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

// ComponentClient - rest client for Component resources that have a defined resource scope
type ComponentClient struct {
	client v1.Scoped
}

// NewComponentClient - creates a client scoped to a particular resource
func NewComponentClient(c v1.Base) (*ComponentClient, error) {

	client, err := c.ForKind(m.ComponentGVK())
	if err != nil {
		return nil, err
	}

	return &ComponentClient{client}, nil

}

// List - gets a list of resources
func (c *ComponentClient) List(options ...v1.ListOptions) ([]*m.Component, error) {
	riList, err := c.client.List(options...)
	if err != nil {
		return nil, err
	}

	result := make([]*m.Component, len(riList))

	for i := range riList {
		result[i] = &m.Component{}
		err := result[i].FromInstance(riList[i])
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// Get - gets a resource by name
func (c *ComponentClient) Get(name string) (*m.Component, error) {
	ri, err := c.client.Get(name)
	if err != nil {
		return nil, err
	}

	service := &m.Component{}
	service.FromInstance(ri)

	return service, nil
}

// Delete - deletes a resource
func (c *ComponentClient) Delete(res *m.Component) error {
	ri, err := res.AsInstance()

	if err != nil {
		return err
	}

	return c.client.Delete(ri)
}

// Create - creates a resource
func (c *ComponentClient) Create(res *m.Component, opts ...v1.CreateOption) (*m.Component, error) {
	ri, err := res.AsInstance()

	if err != nil {
		return nil, err
	}

	cri, err := c.client.Create(ri, opts...)
	if err != nil {
		return nil, err
	}

	created := &m.Component{}

	err = created.FromInstance(cri)
	if err != nil {
		return nil, err
	}

	return created, err
}

// Update - updates a resource
func (c *ComponentClient) Update(res *m.Component, opts ...v1.UpdateOption) (*m.Component, error) {
	ri, err := res.AsInstance()
	if err != nil {
		return nil, err
	}
	resource, err := c.client.Update(ri, opts...)
	if err != nil {
		return nil, err
	}

	updated := &m.Component{}

	// Updates the resource in place
	err = updated.FromInstance(resource)
	if err != nil {
		return nil, err
	}

	return updated, nil
}
