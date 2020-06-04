package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	apiv1 "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
)

func (ba *basicAuth) Authenticate(req *http.Request) {
	req.SetBasicAuth(ba.user, ba.pass)
	req.Header.Set("X-Axway-Tenant-Id", ba.tenantId)
	req.Header.Set("X-Axway-Instance-Id", ba.instanceId)
}

func BasicAuth(user, password, tenantId, instanceId string) Options {
	return func(c *ClientBase) {
		c.auth = &basicAuth{
			user:       user,
			pass:       password,
			tenantId:   tenantId,
			instanceId: instanceId,
		}
	}
}

func HTTPClient(client *http.Client) Options {
	return func(c *ClientBase) {
		c.client = client
	}
}

func NewClient(baseUrl string, options ...Options) *ClientBase {
	c := &ClientBase{
		client: &http.Client{},
		url:    baseUrl,
		auth:   noopAuth{},
	}

	for _, o := range options {
		o(c)
	}

	return c
}

func (cb *ClientBase) ForKind(gvk apiv1.GroupVersionKind) (*Client, error) {
	resource, ok := apiv1.GetResource(gvk.GroupKind)
	if !ok {
		return nil, fmt.Errorf("no resource for gvk: %s", gvk)
	}

	sk, ok := apiv1.GetScope(gvk.GroupKind)
	if !ok {
		return nil, fmt.Errorf("no scope for gvk: %s", gvk)
	}

	scopeResource := ""

	if sk != "" {
		sGV := apiv1.GroupKind{Group: gvk.Group, Kind: sk}
		scopeResource, ok = apiv1.GetResource(sGV)
		if !ok {
			return nil, fmt.Errorf("no resource for scope gv: %s", sGV)
		}
	}

	return &Client{
		ClientBase:    cb,
		version:       gvk.ApiVersion,
		group:         gvk.Group,
		resource:      resource,
		scopeResource: scopeResource,
	}, nil
}

const (
	// baseURL/group/version/scopeResource/scope/resource
	scopedURLFormat   = "%s/%s/%s/%s/%s/%s"
	unscopedURLFormat = "%s/%s/%s/%s"
)

func (c *Client) url() string {
	if c.scopeResource != "" {
		return fmt.Sprintf(scopedURLFormat, c.ClientBase.url, c.group, c.version, c.scopeResource, c.scope, c.resource)
	}

	return fmt.Sprintf(unscopedURLFormat, c.ClientBase.url, c.group, c.version, c.resource)
}

func (c *Client) urlForResource(rm *apiv1.ResourceMeta) string {
	if c.scopeResource != "" {
		scope := c.scope
		if c.scope == "" {
			scope = rm.Metadata.Scope.Name
		}

		return fmt.Sprintf(scopedURLFormat+"/%s", c.ClientBase.url, c.group, c.version, c.scopeResource, scope, c.resource, rm.Name)
	}

	return fmt.Sprintf(unscopedURLFormat+"/%s", c.ClientBase.url, c.group, c.version, c.resource, rm.Name)
}

func (c *Client) WithScope(scope string) *Client {
	return &Client{
		ClientBase:    c.ClientBase,
		version:       c.version,
		group:         c.group,
		resource:      c.resource,
		scopeResource: c.scopeResource,
		scope:         scope,
	}
}

func (c *Client) List() ([]*apiv1.ResourceInstance, error) {
	req, err := http.NewRequest("GET", c.url(), nil)
	if err != nil {
		return nil, err
	}

	c.auth.Authenticate(req)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Failed to get resource list: %s", res.Status)
	}
	dec := json.NewDecoder(res.Body)
	objs := []*apiv1.ResourceInstance{}
	err = dec.Decode(&objs)
	if err != nil {
		return nil, err
	}

	return objs, nil
}

func (c *Client) Get(name string) (*apiv1.ResourceInstance, error) {
	req, err := http.NewRequest("GET", c.urlForResource(&apiv1.ResourceMeta{Name: name}), nil)
	if err != nil {
		return nil, err
	}

	c.auth.Authenticate(req)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Failed to get resource list: %s", res.Status)
	}
	dec := json.NewDecoder(res.Body)
	obj := &apiv1.ResourceInstance{}
	err = dec.Decode(&obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (c *Client) Delete(ri *apiv1.ResourceInstance) error {
	req, err := http.NewRequest("DELETE", c.urlForResource(&ri.ResourceMeta), nil)
	if err != nil {
		return err
	}

	c.auth.Authenticate(req)

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 202 && res.StatusCode != 204 {
		return fmt.Errorf("Failed to delete resource: %s", res.Status)
	}
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Create(ri *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)

	err := enc.Encode(ri)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.url(), buf)
	if err != nil {
		return nil, err
	}
	c.auth.Authenticate(req)
	req.Header.Add("Content-Type", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 201 {
		return nil, fmt.Errorf("Failed to create resource: %s", res.Status)
	}

	dec := json.NewDecoder(res.Body)
	obj := &apiv1.ResourceInstance{}
	err = dec.Decode(obj)
	if err != nil {
		return nil, err
	}

	return obj, err
}

func (c *Client) Update(ri *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)

	err := enc.Encode(ri)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", c.urlForResource(&ri.ResourceMeta), buf)
	if err != nil {
		return nil, err
	}
	c.auth.Authenticate(req)
	req.Header.Add("Content-Type", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Failed to Update resource: %s", res.Status)
	}

	dec := json.NewDecoder(res.Body)
	obj := &apiv1.ResourceInstance{}
	err = dec.Decode(obj)
	if err != nil {
		return nil, err
	}

	return obj, err
}
