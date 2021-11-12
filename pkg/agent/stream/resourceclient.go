package stream

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Axway/agent-sdk/pkg/api"

	"github.com/Axway/agent-sdk/pkg/apic/auth"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
)

// ResourceGetter interface for retrieving a ResourceInstance
type ResourceGetter interface {
	Get(selfLink string) (*apiv1.ResourceInstance, error)
}

// ResourceClient client for getting a ResourceInstance
type ResourceClient struct {
	url      string
	client   api.Client
	auth     auth.TokenGetter
	tenantID string
}

// NewResourceClient creates a new ResourceClient
func NewResourceClient(url string, client api.Client, getToken auth.TokenGetter, tenantID string) *ResourceClient {
	return &ResourceClient{
		auth:     getToken,
		url:      url,
		client:   client,
		tenantID: tenantID,
	}
}

// Get retrieves a ResourceClient
func (c *ResourceClient) Get(selfLink string) (*apiv1.ResourceInstance, error) {
	token, err := c.auth.GetToken()
	if err != nil {
		return nil, err
	}

	req := api.Request{
		Method:  http.MethodGet,
		URL:     c.url + selfLink,
		Headers: make(map[string]string),
	}

	req.Headers["authorization"] = token
	req.Headers["x-axway-tenant-id"] = c.tenantID

	res, err := c.client.Send(req)
	if err != nil {
		return nil, err
	}

	if res.Code != 200 {
		return nil, fmt.Errorf("expected a 200 response but received %d", res.Code)
	}

	ri := &apiv1.ResourceInstance{}
	err = json.Unmarshal(res.Body, ri)

	return ri, err
}
