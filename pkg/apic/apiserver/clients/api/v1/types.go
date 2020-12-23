package v1

import (
	"context"
	"fmt"
	"net/http"

	"net/http/httputil"

	apiv1 "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/auth"
	ot "github.com/opentracing/opentracing-go"
)

// Options
type Options func(*ClientBase)

type authenticator interface {
	Authenticate(req *http.Request) error
}

type impersonator interface {
	impersonate(req *http.Request, toImpersonate string) error
}

type noImpersonator struct{}

func (noImpersonator) impersonate(_ *http.Request, _ string) error {
	return fmt.Errorf("user impersonation not allowed")
}

type noopAuth struct{}

// Authenticate -
func (noopAuth) Authenticate(*http.Request) error {
	return nil
}

type basicAuth struct {
	instanceID string
	pass       string
	tenantID   string
	user       string
}

type jwtAuth struct {
	tenantID    string
	tokenGetter auth.PlatformTokenGetter
}

type requestDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type loggingDoerWrapper struct {
	Logger
	wrapped requestDoer
}

func (ldw loggingDoerWrapper) Do(req *http.Request) (res *http.Response, err error) {
	bReq, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		ldw.Log("error", "failed to log request: "+err.Error())
		return
	}

	res, err = ldw.wrapped.Do(req)
	if err != nil {
		ldw.Log("req", string(bReq), "error", err)
		return
	}

	bRes, err := httputil.DumpResponse(res, true)
	if err != nil {
		ldw.Log("error", "failed to log request: "+err.Error())
		return
	}

	ldw.Log("request", string(bReq), "response", string(bRes))
	return
}

// ClientBase for grouping a client, auth method and url together
type ClientBase struct {
	tracer       ot.Tracer
	client       requestDoer
	url          string
	auth         authenticator
	impersonator impersonator
	log          Logger
}

// Client for a resource with the given version, group & scope
type Client struct {
	*ClientBase
	version       string
	group         string
	resource      string
	scopeResource string
	scope         string
	query         string
}

// Base -
type Base interface {
	ForKind(apiv1.GroupVersionKind) (Unscoped, error)
}

// BaseCtx -
type BaseCtx interface {
	ForKindCtx(apiv1.GroupVersionKind) (UnscopedCtx, error)
}

// UnscopedCtx -
type UnscopedCtx interface {
	ScopedCtx
	WithScope(name string) ScopedCtx
}

// Unscoped -
type Unscoped interface {
	Scoped
	WithScope(name string) Scoped
}

// ScopedCtx -
type ScopedCtx interface {
	CreateCtx(context.Context, *apiv1.ResourceInstance, ...CreateOption) (*apiv1.ResourceInstance, error)
	DeleteCtx(context.Context, *apiv1.ResourceInstance) error
	GetCtx(context.Context, string) (*apiv1.ResourceInstance, error)
	ListCtx(context.Context, ...ListOptions) ([]*apiv1.ResourceInstance, error)
	UpdateCtx(context.Context, *apiv1.ResourceInstance, ...UpdateOption) (*apiv1.ResourceInstance, error)
}

type Scoped interface {
	Create(*apiv1.ResourceInstance, ...CreateOption) (*apiv1.ResourceInstance, error)
	Delete(*apiv1.ResourceInstance) error
	Get(string) (*apiv1.ResourceInstance, error)
	List(...ListOptions) ([]*apiv1.ResourceInstance, error)
	Update(*apiv1.ResourceInstance, ...UpdateOption) (*apiv1.ResourceInstance, error)
}

type UpdateOption func(*updateOptions)

type updateOptions struct {
	impersonateUserID string
	mergeFunc         MergeFunc
}

type CreateOption func(*createOptions)

type createOptions struct {
	impersonateUserID string
}

// ListOptions -
type ListOptions func(*listOptions)

type listOptions struct {
	query QueryNode
}

// EventHandler -
type EventHandler interface {
	Handle(*apiv1.Event)
}

// EventHandlerFunc -
type EventHandlerFunc func(*apiv1.Event)

// Handle -
func (ehf EventHandlerFunc) Handle(ev *apiv1.Event) {
	ehf(ev)
}

// QueryNode represents a query
type QueryNode interface {
	Accept(Visitor)
}
