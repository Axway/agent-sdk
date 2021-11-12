package stream

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Axway/agent-sdk/pkg/apic/auth"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
)

type doer interface {
	Do(*http.Request) (*http.Response, error)
}

// RiGetter interface for retrieving a ResourceInstance
type RiGetter interface {
	Get(selfLink string) (apiv1.Interface, error)
}

// ResourceInstanceClient client for getting a ResourceInstance
type ResourceInstanceClient struct {
	url      string
	client   doer
	auth     auth.TokenGetter
	tenantID string
}

// NewResourceInstanceClient creates a new ResourceInstanceClient
func NewResourceInstanceClient(url string, client doer, getToken auth.TokenGetter, tenantID string) *ResourceInstanceClient {
	return &ResourceInstanceClient{
		auth:     getToken,
		url:      url,
		client:   client,
		tenantID: tenantID,
	}
}

// Get retrieves a ResourceInstanceClient
func (c *ResourceInstanceClient) Get(selfLink string) (apiv1.Interface, error) {
	token, err := c.auth.GetToken()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, c.url+selfLink, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-axway-tenant-id", c.tenantID)
	req.Header.Set("authorization", token)
	req.Header.Set("user-agent", "something")

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("expected a 200 response but received %d", res.StatusCode)
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	ri := &apiv1.ResourceInstance{}
	err = json.Unmarshal(data, ri)

	return ri, err
}
