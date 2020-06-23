package v1

import (
	"context"
	"net/http"

	apiv1 "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
	"git.ecd.axway.int/apigov/service-mesh-agent/pkg/apicauth"
	ot "github.com/opentracing/opentracing-go"
)

// Options
type Options func(*ClientBase)

type authenticator interface {
	Authenticate(req *http.Request) error
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
	tokenGetter *apicauth.PlatformTokenGetter
}

// ClientBase for grouping a client, auth method and url together
type ClientBase struct {
	tracer ot.Tracer
	client *http.Client
	url    string
	auth   authenticator
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

type Base interface {
	ForKind(apiv1.GroupVersionKind) (Unscoped, error)
}

type Unscoped interface {
	Scoped
	WithScope(name string) Scoped
}

type Scoped interface {
	Create(context.Context, *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error)
	Delete(context.Context, *apiv1.ResourceInstance) error
	Get(context.Context, string) (*apiv1.ResourceInstance, error)
	List(context.Context, ...ListOptions) ([]*apiv1.ResourceInstance, error)
	Update(context.Context, *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error)
}

type ListOptions func(*listOptions)

type listOptions struct {
	query QueryNode
}

type EventHandler interface {
	Handle(*apiv1.Event)
}

type EventHandlerFunc func(*apiv1.Event)

func (ehf EventHandlerFunc) Handle(ev *apiv1.Event) {
	ehf(ev)
}

// QueryNode represents a query
type QueryNode interface {
	Accept(Visitor)
}
