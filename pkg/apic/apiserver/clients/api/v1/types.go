package v1

import (
	"net/http"
)

type Options func(*ClientBase)

type authenticator interface {
	Authenticate(req *http.Request)
}

type noopAuth struct{}

func (noopAuth) Authenticate(*http.Request) {}

type basicAuth struct {
	user       string
	pass       string
	tenantId   string
	instanceId string
}

type ClientBase struct {
	client *http.Client
	url    string
	auth   authenticator
}

type Client struct {
	*ClientBase
	version       string
	group         string
	resource      string
	scopeResource string
	scope         string
}
