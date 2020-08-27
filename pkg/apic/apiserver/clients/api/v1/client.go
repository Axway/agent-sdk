package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	apiv1 "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
	"git.ecd.axway.org/apigov/service-mesh-agent/pkg/apicauth"
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

func (ba *basicAuth) impersonate(req *http.Request, toImpersonate string) error {
	req.Header.Set("X-Axway-User-Id", toImpersonate)

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
		ba := &basicAuth{
			user:       user,
			pass:       password,
			tenantID:   tenantID,
			instanceID: instanceID,
		}

		c.auth = ba

		c.impersonator = ba
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

type Logger interface {
	Log(kv ...interface{}) error
}

type noOpLogger struct{}

func (noOpLogger) Log(kv ...string) error { return nil }

func WithLogger(log Logger) Options {
	return func(cb *ClientBase) {
		cb.client = loggingDoerWrapper{log, cb.client}
	}
}

// NewClient creates a new HTTP client
func NewClient(baseURL string, options ...Options) *ClientBase {
	c := &ClientBase{
		client:       &http.Client{},
		url:          baseURL,
		auth:         noopAuth{},
		impersonator: noImpersonator{},
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

type ClientCtx struct {
	Client
}

func (c *ClientCtx) WithScope(scope string) ScopedCtx {
	return c
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

func (c *Client) url(rm apiv1.ResourceMeta) string {
	if c.scopeResource != "" {
		scope := c.scope
		if c.scope == "" {
			scope = rm.Metadata.Scope.Name
		}

		return fmt.Sprintf(scopedURLFormat, c.ClientBase.url, c.group, c.version, c.scopeResource, scope, c.resource)
	}

	return fmt.Sprintf(unscopedURLFormat, c.ClientBase.url, c.group, c.version, c.resource)
}

func (c *Client) urlForResource(rm apiv1.ResourceMeta) string {
	return c.url(rm) + "/" + rm.Name
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
func WithQuery(n QueryNode) func(*listOptions) {
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
	req, err := http.NewRequestWithContext(ctx, "GET", c.url(apiv1.ResourceMeta{}), nil)
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

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, handleError(res)
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
	return c.GetCtx(context.Background(), name)
}

// GetCtx2 returns a single resource. If client is unscoped then name can be "<scopeName>/<name>".
// If client is scoped then name can be "<name>" or "<scopeName>/<name>" but <scopeName> is ignored.
func (c *Client) GetCtx2(ctx context.Context, toGet *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.urlForResource(toGet.ResourceMeta), nil)
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

// GetCtx returns a single resource. If client is unscoped then name can be "<scopeName>/<name>".
// If client is scoped then name can be "<name>" or "<scopeName>/<name>" but <scopeName> is ignored.
func (c *Client) GetCtx(ctx context.Context, name string) (*apiv1.ResourceInstance, error) {
	split := strings.SplitN(name, `/`, 2)

	url := ""

	if len(split) == 2 {
		url = c.urlForResource(apiv1.ResourceMeta{Name: split[1], Metadata: apiv1.Metadata{Scope: apiv1.MetadataScope{Name: split[0]}}})
	} else {
		url = c.urlForResource(apiv1.ResourceMeta{Name: name})
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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

func (c *Client) Delete(ri *apiv1.ResourceInstance) error {
	return c.DeleteCtx(context.Background(), ri)
}

// DeleteCtx deletes a single resource
func (c *Client) DeleteCtx(ctx context.Context, ri *apiv1.ResourceInstance) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", c.urlForResource(ri.ResourceMeta), nil)
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
	if err != nil {
		return err
	}

	return nil
}

func CUserID(userID string) CreateOption {
	return func(co *createOptions) {
		co.impersonateUserID = userID
	}
}

// Create creates a single resource
func (c *Client) Create(ri *apiv1.ResourceInstance, opts ...CreateOption) (*apiv1.ResourceInstance, error) {
	return c.CreateCtx(context.Background(), ri, opts...)
}

// CreateCtx creates a single resource
func (c *Client) CreateCtx(ctx context.Context, ri *apiv1.ResourceInstance, opts ...CreateOption) (*apiv1.ResourceInstance, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)

	co := createOptions{}

	for _, opt := range opts {
		opt(&co)
	}

	err := enc.Encode(ri)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.url(ri.ResourceMeta), buf)
	if err != nil {
		return nil, err
	}
	err = c.auth.Authenticate(req)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	if co.impersonateUserID != "" {
		err = c.impersonator.impersonate(req, co.impersonateUserID)
		if err != nil {
			return nil, err
		}
	}

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

func UUserID(userID string) UpdateOption {
	return func(co *updateOptions) {
		co.impersonateUserID = userID
	}
}

type MergeFunc func(fetched apiv1.Instance, new apiv1.Instance) (apiv1.Instance, error)

// Merge option first fetches the resource and then
// applies the merge function and uses the result for the actual update
// fetched will be the old resource
// new will be the resource passed to the Update call
// If the resource doesn't exist it will fetched will be set to null
// If the merge function returns an error, the update operation will be cancelled
func Merge(merge MergeFunc) UpdateOption {
	return func(co *updateOptions) {
		co.mergeFunc = merge
	}
}

// Update updates a single resource. If the merge option is passed it will first fetch the resource and then apply the merge function.
func (c *Client) Update(ri *apiv1.ResourceInstance, opts ...UpdateOption) (*apiv1.ResourceInstance, error) {
	return c.UpdateCtx(context.Background(), ri, opts...)
}

// UpdateCtx updates a single resource
func (c *Client) UpdateCtx(ctx context.Context, ri *apiv1.ResourceInstance, opts ...UpdateOption) (*apiv1.ResourceInstance, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)

	uo := updateOptions{}

	for _, opt := range opts {
		opt(&uo)
	}

	if uo.mergeFunc != nil {
		old, err := c.GetCtx2(ctx, ri)
		if err != nil {
			switch err.(type) {
			case NotFoundError:
				old = nil
			default:
				return nil, err
			}
		}

		i, err := uo.mergeFunc(old, ri)
		if err != nil {
			return nil, err
		}
		newRi, err := i.AsInstance()
		if err != nil {
			return nil, err
		}

		if old == nil {
			return c.CreateCtx(ctx, newRi, CUserID(uo.impersonateUserID))
		}

		ri = newRi
	}

	err := enc.Encode(ri)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", c.urlForResource(ri.ResourceMeta), buf)
	if err != nil {
		return nil, err
	}

	err = c.auth.Authenticate(req)
	if err != nil {
		return nil, err
	}

	if uo.impersonateUserID != "" {
		err = c.impersonator.impersonate(req, uo.impersonateUserID)
		if err != nil {
			return nil, err
		}
	}

	req.Header.Add("Content-Type", "application/json")

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
	err = dec.Decode(obj)
	if err != nil {
		return nil, err
	}

	return obj, err
}
