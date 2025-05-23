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

type ComplianceAgentMergeFunc func(*m.ComplianceAgent, *m.ComplianceAgent) (*m.ComplianceAgent, error)

// ComplianceAgentMerge builds a merge option for an update operation
func ComplianceAgentMerge(f ComplianceAgentMergeFunc) v1.UpdateOption {
	return v1.Merge(func(prev, new apiv1.Interface) (apiv1.Interface, error) {
		p, n := &m.ComplianceAgent{}, &m.ComplianceAgent{}

		switch t := prev.(type) {
		case *m.ComplianceAgent:
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
		case *m.ComplianceAgent:
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

// ComplianceAgentClient - rest client for ComplianceAgent resources that have a defined resource scope
type ComplianceAgentClient struct {
	client v1.Scoped
}

// UnscopedComplianceAgentClient - rest client for ComplianceAgent resources that do not have a defined scope
type UnscopedComplianceAgentClient struct {
	client v1.Unscoped
}

// NewComplianceAgentClient - creates a client that is not scoped to any resource
func NewComplianceAgentClient(c v1.Base) (*UnscopedComplianceAgentClient, error) {

	client, err := c.ForKind(m.ComplianceAgentGVK())
	if err != nil {
		return nil, err
	}

	return &UnscopedComplianceAgentClient{client}, nil

}

// WithScope - sets the resource scope for the client
func (c *UnscopedComplianceAgentClient) WithScope(scope string) *ComplianceAgentClient {
	return &ComplianceAgentClient{
		c.client.WithScope(scope),
	}
}

// Get - gets a resource by name
func (c *UnscopedComplianceAgentClient) Get(name string) (*m.ComplianceAgent, error) {
	ri, err := c.client.Get(name)
	if err != nil {
		return nil, err
	}

	service := &m.ComplianceAgent{}
	service.FromInstance(ri)

	return service, nil
}

// Update - updates a resource
func (c *UnscopedComplianceAgentClient) Update(res *m.ComplianceAgent, opts ...v1.UpdateOption) (*m.ComplianceAgent, error) {
	ri, err := res.AsInstance()
	if err != nil {
		return nil, err
	}
	resource, err := c.client.Update(ri, opts...)
	if err != nil {
		return nil, err
	}

	updated := &m.ComplianceAgent{}

	// Updates the resource in place
	err = updated.FromInstance(resource)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// List - gets a list of resources
func (c *ComplianceAgentClient) List(options ...v1.ListOptions) ([]*m.ComplianceAgent, error) {
	riList, err := c.client.List(options...)
	if err != nil {
		return nil, err
	}

	result := make([]*m.ComplianceAgent, len(riList))

	for i := range riList {
		result[i] = &m.ComplianceAgent{}
		err := result[i].FromInstance(riList[i])
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// Get - gets a resource by name
func (c *ComplianceAgentClient) Get(name string) (*m.ComplianceAgent, error) {
	ri, err := c.client.Get(name)
	if err != nil {
		return nil, err
	}

	service := &m.ComplianceAgent{}
	service.FromInstance(ri)

	return service, nil
}

// Delete - deletes a resource
func (c *ComplianceAgentClient) Delete(res *m.ComplianceAgent) error {
	ri, err := res.AsInstance()

	if err != nil {
		return err
	}

	return c.client.Delete(ri)
}

// Create - creates a resource
func (c *ComplianceAgentClient) Create(res *m.ComplianceAgent, opts ...v1.CreateOption) (*m.ComplianceAgent, error) {
	ri, err := res.AsInstance()

	if err != nil {
		return nil, err
	}

	cri, err := c.client.Create(ri, opts...)
	if err != nil {
		return nil, err
	}

	created := &m.ComplianceAgent{}

	err = created.FromInstance(cri)
	if err != nil {
		return nil, err
	}

	return created, err
}

// Update - updates a resource
func (c *ComplianceAgentClient) Update(res *m.ComplianceAgent, opts ...v1.UpdateOption) (*m.ComplianceAgent, error) {
	ri, err := res.AsInstance()
	if err != nil {
		return nil, err
	}
	resource, err := c.client.Update(ri, opts...)
	if err != nil {
		return nil, err
	}

	updated := &m.ComplianceAgent{}

	// Updates the resource in place
	err = updated.FromInstance(resource)
	if err != nil {
		return nil, err
	}

	return updated, nil
}
