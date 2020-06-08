package v1

import (
	"net/http"
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
	user       string
	pass       string
	tenantId   string
	instanceId string
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
}
