package stream

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Axway/agent-sdk/pkg/api"

	"github.com/Axway/agent-sdk/pkg/apic/auth"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
)

// resourceGetter interface for retrieving a ResourceInstance
type resourceGetter interface {
	get(selfLink string) (*apiv1.ResourceInstance, error)
}

// resourceClient client for getting a ResourceInstance
type resourceClient struct {
	auth     auth.TokenGetter
	client   api.Client
	tenantID string
	url      string
}

// newResourceClient creates a new resourceClient
func newResourceClient(url, tenantID string, client api.Client, getToken auth.TokenGetter) *resourceClient {
	return &resourceClient{
		auth:     getToken,
		client:   client,
		tenantID: tenantID,
		url:      url,
	}
}

// get retrieves a resourceClient
func (c *resourceClient) get(selfLink string) (*apiv1.ResourceInstance, error) {
	token, err := c.auth.GetToken()
	if err != nil {
		return nil, err
	}

	req := api.Request{
		Method:  http.MethodGet,
		URL:     c.url + selfLink,
		Headers: make(map[string]string),
	}

	req.Headers["Authorization"] = "Bearer " + token
	req.Headers["X-Axway-Tenant-Id"] = c.tenantID
	req.Headers["Content-Type"] = "application/json"
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
