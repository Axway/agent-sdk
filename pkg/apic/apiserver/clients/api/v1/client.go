package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	apiv1 "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/auth"
	"github.com/tomnomnom/linkheader"

	ot "github.com/opentracing/opentracing-go"
)

// HTTPClient allows you to replace the default client for different use cases
func HTTPClient(client *http.Client) Options {
	return func(c *ClientBase) {
		c.client = client
	}
}

// Authenticate Basic authentication
func (ba *basicAuth) Authenticate(req *http.Request) error {
	req.SetBasicAuth(ba.user, ba.pass)
	req.Header.Set("X-Axway-Tenant-Id", ba.tenantID)
	req.Header.Set("X-Axway-Instance-Id", ba.instanceID)
	return nil
}

// Authenticate JWT Authentication
func (j *jwtAuth) Authenticate(req *http.Request) error {
	t, err := j.tokenGetter.GetToken()
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t))
	req.Header.Set("X-Axway-Tenant-Id", j.tenantID)
	return nil
}

type modifier interface {
	Modify()
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
		tokenGetter := auth.NewPlatformTokenGetter(privKey, pubKey, password, url, aud, clientID, timeout)
		c.auth = &jwtAuth{
			tenantID:    tenantID,
			tokenGetter: tokenGetter,
		}
	}
}

// NewClient creates a new HTTP client
func NewClient(baseURL string, options ...Options) *ClientBase {
	c := &ClientBase{
		tracer: ot.NoopTracer{},
		client: &http.Client{},
		url:    baseURL,
		auth:   noopAuth{},
	}

	for _, o := range options {
		o(c)
	}

	return c
}

func (cb *ClientBase) forKindInternal(gvk apiv1.GroupVersionKind) (*Client, error) {
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

// ForKindCtx registers a client with a given group/version
func (cb *ClientBase) ForKindCtx(gvk apiv1.GroupVersionKind) (UnscopedCtx, error) {
	c, err := cb.forKindInternal(gvk)
	return &ClientCtx{*c}, err
}

// ForKind registers a client with a given group/version
func (cb *ClientBase) ForKind(gvk apiv1.GroupVersionKind) (Unscoped, error) {
	return cb.forKindInternal(gvk)
}

const (
	// baseURL/group/version/scopeResource/scope/resource
	scopedURLFormat   = "%s/%s/%s/%s/%s/%s"
	unscopedURLFormat = "%s/%s/%s/%s"
)

// ClientCtx -
type ClientCtx struct {
	Client
}

// WithScope -
func (c *ClientCtx) WithScope(scope string) ScopedCtx {
	return c
}

func (c *Client) url() string {
	// unscoped
	url := fmt.Sprintf(unscopedURLFormat, c.ClientBase.url, c.group, c.version, c.resource)

	// scoped
	if c.scopeResource != "" {
		url = fmt.Sprintf(scopedURLFormat, c.ClientBase.url, c.group, c.version, c.scopeResource, c.scope, c.resource)
	}

	return url
}

// handleError handles an api-server error response. caller should close body.
func handleError(res *http.Response) error {
	var errors Errors
	errRes := apiv1.ErrorResponse{}
	err := json.NewDecoder(res.Body).Decode(&errRes)
	if err != nil {
		errors = []apiv1.Error{{
			Status: 0,
			Detail: err.Error(),
		}}
	} else {
		errors = errRes.Errors
	}

	switch res.StatusCode {
	case 400:
		return BadRequestError{errors}
	case 401:
		return UnauthorizedError{errors}
	case 403:
		return ForbiddenError{errors}
	case 404:
		return NotFoundError{errors}
	case 409:
		return ConflictError{errors}
	case 500:
		return InternalServerError{errors}
	default:
		return UnexpectedError{res.StatusCode, errors}
	}
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
func (c *Client) WithScope(scope string) Scoped {
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
func WithQuery(n QueryNode) ListOptions {
	return func(lo *listOptions) {
		lo.query = n
	}
}

// List -
func (c *Client) List(options ...ListOptions) ([]*apiv1.ResourceInstance, error) {
	return c.ListCtx(context.Background(), options...)
}

// ListCtx returns a list of resources
func (c *Client) ListCtx(ctx context.Context, options ...ListOptions) ([]*apiv1.ResourceInstance, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(), nil)
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
	return c.listAll(req)
}

func (c *Client) doOneRequest(req *http.Request) ([]*apiv1.ResourceInstance, linkheader.Links, error) {
	res, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, nil, handleError(res)
	}
	dec := json.NewDecoder(res.Body)
	var objs []*apiv1.ResourceInstance
	err = dec.Decode(&objs)
	if err != nil {
		return nil, nil, err
	}
	links := linkheader.Parse(res.Header.Get("Link"))
	return objs, links.FilterByRel("next"), nil
}

// fetch all items based on the Link headers
func (c *Client) listAll(req *http.Request) ([]*apiv1.ResourceInstance, error) {
	var objs []*apiv1.ResourceInstance
	for {
		res, links, err := c.doOneRequest(req)
		if err != nil {
			return nil, err
		}
		objs = append(objs, res...)
		if links == nil || len(links) == 0 {
			break
		}
		link := links[0]
		parsedLink, err := url.Parse(link.URL)
		if err != nil {
			return nil, err
		}
		req.URL.RawQuery = parsedLink.RawQuery
	}
	return objs, nil
}

// Get -
func (c *Client) Get(name string) (*apiv1.ResourceInstance, error) {
	return c.GetCtx(context.Background(), name)
}

// GetCtx returns a single resource
func (c *Client) GetCtx(ctx context.Context, name string) (*apiv1.ResourceInstance, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.urlForResource(&apiv1.ResourceMeta{Name: name}), nil)
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
		return nil, handleError(res)
	}
	dec := json.NewDecoder(res.Body)
	obj := &apiv1.ResourceInstance{}
	err = dec.Decode(&obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

// Delete -
func (c *Client) Delete(ri *apiv1.ResourceInstance) error {
	return c.DeleteCtx(context.Background(), ri)
}

// DeleteCtx deletes a single resource
func (c *Client) DeleteCtx(ctx context.Context, ri *apiv1.ResourceInstance) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.urlForResource(&ri.ResourceMeta), nil)
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
		return handleError(res)
	}

	return nil
}

// Create -
func (c *Client) Create(ri *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
	return c.CreateCtx(context.Background(), ri)
}

// CreateCtx creates a single resource
func (c *Client) CreateCtx(ctx context.Context, ri *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)

	err := enc.Encode(ri)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(), buf)
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
		return nil, handleError(res)
	}

	dec := json.NewDecoder(res.Body)
	obj := &apiv1.ResourceInstance{}
	err = dec.Decode(obj)
	if err != nil {
		return nil, err
	}

	return obj, err
}

// Update -
func (c *Client) Update(ri *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
	return c.UpdateCtx(context.Background(), ri)
}

// UpdateCtx updates a single resource
func (c *Client) UpdateCtx(ctx context.Context, ri *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)

	err := enc.Encode(ri)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.urlForResource(&ri.ResourceMeta), buf)
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
	switch res.StatusCode {
	case 200:
	case 404:
	default:
	}
	if res.StatusCode != 200 {
		return nil, handleError(res)
	}

	dec := json.NewDecoder(res.Body)
	obj := &apiv1.ResourceInstance{}
	err = dec.Decode(obj)
	if err != nil {
		return nil, err
	}

	return obj, err
}
