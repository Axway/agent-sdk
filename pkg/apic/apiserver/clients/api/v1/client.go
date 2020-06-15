package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	apiv1 "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
	"git.ecd.axway.int/apigov/service-mesh-agent/pkg/apicauth"
)

// HTTPClient allows you to replace the default client for different use cases
func HTTPClient(client *http.Client) Options {
	return func(c *ClientBase) {
		c.client = client
	}
}

func (ba *basicAuth) Authenticate(req *http.Request) error {
	req.SetBasicAuth(ba.user, ba.pass)
	req.Header.Set("X-Axway-Tenant-Id", ba.tenantID)
	req.Header.Set("X-Axway-Instance-Id", ba.instanceID)
	return nil
}

func (j *jwtAuth) Authenticate(req *http.Request) error {
	t, err := j.tokenGetter.GetToken()
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t))
	req.Header.Set("X-Axway-Tenant-Id", j.tenantID)
	return nil
}

// BasicAuth auth with user/pass
func BasicAuth(user, password, tenantID, instanceID string) Options {
	return func(c *ClientBase) {
		c.auth = &basicAuth{
			user:       user,
			pass:       password,
			tenantID:   tenantID,
			instanceID: instanceID,
		}
	}
}

// JWTAuth auth with token
func JWTAuth(tenantID, privKey, pubKey, password, url, aud, clientID string, timeout time.Duration) Options {
	return func(c *ClientBase) {
		tokenGetter := apicauth.NewPlatformTokenGetter(privKey, pubKey, password, url, aud, clientID, timeout)
		c.auth = &jwtAuth{
			tenantID:    tenantID,
			tokenGetter: tokenGetter,
		}
	}
}

// NewClient creates a new HTTP client
func NewClient(baseURL string, options ...Options) *ClientBase {
	c := &ClientBase{
		client: &http.Client{},
		url:    baseURL,
		auth:   noopAuth{},
	}

	for _, o := range options {
		o(c)
	}

	return c
}

// ForKind registers a client with a given group/version
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
		version:       gvk.APIVersion,
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
	// unscoped
	url := fmt.Sprintf(unscopedURLFormat, c.ClientBase.url, c.group, c.version, c.resource)

	// scoped
	if c.scopeResource != "" {
		url = fmt.Sprintf(scopedURLFormat, c.ClientBase.url, c.group, c.version, c.scopeResource, c.scope, c.resource)
	}

	return url
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

// WithScope creates a request within the given scope. ex: env/$name/services
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

// WithQuery applies a query on the list operation
func WithQuery(n QueryNode) func(*listOptions) {
	return func(lo *listOptions) {
		lo.query = n
	}
}

// List returns a list of resources
func (c *Client) List(options ...ListOptions) ([]*apiv1.ResourceInstance, error) {
	req, err := http.NewRequest("GET", c.url(), nil)
	if err != nil {
		return nil, err
	}

	err = c.auth.Authenticate(req)
	if err != nil {
		return nil, err
	}
	opts := listOptions{}

	for _, o := range options {
		o(&opts)
	}

	if opts.query != nil {
		rv := newRSQLVisitor()
		rv.Visit(opts.query)
		q := req.URL.Query()
		q.Add("query", rv.String())
		req.URL.RawQuery = q.Encode()
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

// Get returns a single resource
func (c *Client) Get(name string) (*apiv1.ResourceInstance, error) {
	req, err := http.NewRequest("GET", c.urlForResource(&apiv1.ResourceMeta{Name: name}), nil)
	if err != nil {
		return nil, err
	}

	err = c.auth.Authenticate(req)
	if err != nil {
		return nil, err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Failed to get resource for %s: %s", name, res.Status)
	}
	dec := json.NewDecoder(res.Body)
	obj := &apiv1.ResourceInstance{}
	err = dec.Decode(&obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

// Delete deletes a single resource
func (c *Client) Delete(ri *apiv1.ResourceInstance) error {
	req, err := http.NewRequest("DELETE", c.urlForResource(&ri.ResourceMeta), nil)
	if err != nil {
		return err
	}

	err = c.auth.Authenticate(req)
	if err != nil {
		return err
	}

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

// Create creates a single resource
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
	err = c.auth.Authenticate(req)
	if err != nil {
		return nil, err
	}
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

// Update updates a single resource
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

	err = c.auth.Authenticate(req)
	if err != nil {
		return nil, err
	}

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
