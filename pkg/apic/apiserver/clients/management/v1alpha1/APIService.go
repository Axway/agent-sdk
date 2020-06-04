// TODO complete all methods and convert it to a template

package v1alpha1

import (
	v1 "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/clients/api/v1"
	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

type APIServiceClient struct {
	client *v1.Client
}

func NewAPIServiceClient(cb *v1.ClientBase) (*APIServiceClient, error) {
	client, err := cb.ForKind(v1alpha1.APIServiceGVK())
	if err != nil {
		return nil, err
	}

	return &APIServiceClient{client}, nil
}

func (c *APIServiceClient) WithScope(scope string) *APIServiceClient {
	return &APIServiceClient{
		c.client.WithScope(scope),
	}
}

func (c *APIServiceClient) List() ([]*v1alpha1.APIService, error) {
	ris, err := c.client.List()
	if err != nil {
		return nil, err
	}

	result := make([]*v1alpha1.APIService, len(ris))

	for i := range ris {
		result[i] = &v1alpha1.APIService{}
		err := result[i].FromInstance(ris[i])
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (c *APIServiceClient) Create(res *v1alpha1.APIService) (*v1alpha1.APIService, error) {
	ri, err := res.AsInstance()

	if err != nil {
		return nil, err
	}

	cri, err := c.client.Create(ri)
	if err != nil {
		return nil, err
	}

	created := &v1alpha1.APIService{}

	err = created.FromInstance(cri)
	if err != nil {
		return nil, err
	}

	return res, err
}

func (c *APIServiceClient) Delete(res *v1alpha1.APIService) error {
	ri, err := res.AsInstance()

	if err != nil {
		return err
	}

	return c.client.Delete(ri)
}
