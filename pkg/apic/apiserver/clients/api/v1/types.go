package v1

import (
	"net/http"

	apiv1 "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
)

// Options -
type Options func(*ClientBase)

type authenticator interface {
	Authenticate(req *http.Request)
}

type noopAuth struct{}

// Authenticate -
func (noopAuth) Authenticate(*http.Request) {}

type basicAuth struct {
	instanceId string
	pass       string
	tenantId   string
	user       string
}

type jwtAuth struct {
	instanceId string
	tenantId   string
	token      string
}

// ClientBase for grouping a client, auth method and url together
type ClientBase struct {
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
	Create(*apiv1.ResourceInstance) (*apiv1.ResourceInstance, error)
	Delete(*apiv1.ResourceInstance) error
	Get(string) (*apiv1.ResourceInstance, error)
	List(...ListOptions) ([]*apiv1.ResourceInstance, error)
	Update(*apiv1.ResourceInstance) (*apiv1.ResourceInstance, error)
}

func WithQuery(n QueryNode) func(*listOptions) {
	return func(lo *listOptions) {
		lo.query = n
	}
}

type ListOptions func(*listOptions)

// ListOptions
type listOptions struct {
	query QueryNode //
}

// GetOptions - placeholder for future options
type letOptions struct {
}

// CreateOptions - placeholder for future options
type createOptions struct {
}

// DeleteOptions - placeholder for future options
type deleteOptions struct {
}

// UpdateOptions - placeholder for future options
type updateOptions struct {
}
