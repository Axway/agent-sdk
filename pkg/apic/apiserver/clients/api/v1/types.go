package v1

import (
	"net/http"

	"git.ecd.axway.int/apigov/service-mesh-agent/pkg/apicauth"
)

// Options -
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

// ListOptions are options for the list operation
type ListOptions func(*listOptions)

type listOptions struct {
	query QueryNode //
}

// QueryNode represents a query
type QueryNode interface {
	Accept(Visitor)
}
