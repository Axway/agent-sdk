package stream

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Axway/agent-sdk/pkg/api"

	"github.com/Axway/agent-sdk/pkg/apic/auth"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
)

// ResourceClient interface for creating and retrieving a ResourceInstance
type ResourceClient interface {
	Get(selfLink string) (*apiv1.ResourceInstance, error)
	// create(r *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error)
}

// resourceClient client for getting a ResourceInstance
type resourceClient struct {
	auth     auth.TokenGetter
	client   api.Client
	tenantID string
	url      string
}

// NewResourceClient creates a new resourceClient
func NewResourceClient(url, tenantID string, client api.Client, getToken auth.TokenGetter) ResourceClient {
	return &resourceClient{
		auth:     getToken,
		client:   client,
		tenantID: tenantID,
		url:      url,
	}
}

// Get retrieves a resourceClient
func (c *resourceClient) Get(selfLink string) (*apiv1.ResourceInstance, error) {
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

// func (c *resourceClient) create(r *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
// 	token, err := c.auth.GetToken()
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	req := api.Request{
// 		Method:  http.MethodPost,
// 		URL:     fmt.Sprintf("%s/%s/%s/%s/%s", c.url, r.Group, r.APIVersion, r.Kind, r.Name),
// 		Headers: make(map[string]string),
// 	}
//
// 	req.Headers["Authorization"] = "Bearer " + token
// 	req.Headers["X-Axway-Tenant-Id"] = c.tenantID
// 	req.Headers["Content-Type"] = "application/json"
// 	res, err := c.client.Send(req)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	err = r.UnmarshalJSON(res.Body)
//
// 	return r, err
// }
