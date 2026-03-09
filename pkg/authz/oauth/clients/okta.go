package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
)

// OktaAuthHeaderPrefix is the Authorization header scheme for Okta API tokens.
// Okta expects this to be formatted as: "SSWS <token>".
const OktaAuthHeaderPrefix = "SSWS"

// OktaAPI wraps Okta Management API calls.
type OktaAPI struct {
	BaseURL  string
	APIToken string
	Client   coreapi.Client
}

func New(apiClient coreapi.Client, baseURL, apiToken string) *OktaAPI {
	if apiClient == nil {
		apiClient = coreapi.NewClient(nil, "")
	}
	return &OktaAPI{
		BaseURL:  baseURL,
		APIToken: apiToken,
		Client:   apiClient,
	}
}

func (o *OktaAPI) doRequest(ctx context.Context, method, endpoint string, body interface{}) (*coreapi.Response, error) {
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	request := coreapi.Request{
		Method: method,
		URL:    endpoint,
		Headers: map[string]string{
			"Authorization": OktaAuthHeaderPrefix + " " + o.APIToken,
			"Content-Type":  "application/json",
		},
		Body: reqBody,
	}

	_ = ctx // coreapi.Client doesn't accept ctx today
	return o.Client.Send(request)
}

func (o *OktaAPI) unexpectedStatusError(method, endpoint string, resp *coreapi.Response) error {
	return fmt.Errorf("okta api %s %s returned %d: %s", method, endpoint, resp.Code, string(resp.Body))
}

func isStatus(code int, allowed ...int) bool {
	for _, c := range allowed {
		if code == c {
			return true
		}
	}
	return false
}

func (o *OktaAPI) FindGroupByName(ctx context.Context, groupName string) (string, error) {
	endpoint := fmt.Sprintf("%s/api/v1/groups?q=%s", o.BaseURL, url.QueryEscape(groupName))
	resp, err := o.doRequest(ctx, coreapi.GET, endpoint, nil)
	if err != nil {
		return "", err
	}
	if !isStatus(resp.Code, http.StatusOK) {
		return "", o.unexpectedStatusError(coreapi.GET, endpoint, resp)
	}

	var groups []struct {
		ID      string `json:"id"`
		Profile struct {
			Name string `json:"name"`
		} `json:"profile"`
	}
	if err := json.Unmarshal(resp.Body, &groups); err != nil {
		return "", err
	}
	for _, g := range groups {
		if g.Profile.Name == groupName {
			return g.ID, nil
		}
	}
	return "", nil
}

func (o *OktaAPI) CreateGroup(ctx context.Context, name, description string) (string, error) {
	endpoint := fmt.Sprintf("%s/api/v1/groups", o.BaseURL)
	body := map[string]interface{}{
		"profile": map[string]interface{}{
			"name":        name,
			"description": description,
		},
	}
	resp, err := o.doRequest(ctx, coreapi.POST, endpoint, body)
	if err != nil {
		return "", err
	}
	if resp.Code == http.StatusConflict {
		return o.FindGroupByName(ctx, name)
	}
	if !isStatus(resp.Code, http.StatusOK, http.StatusCreated) {
		return "", o.unexpectedStatusError(coreapi.POST, endpoint, resp)
	}
	var group struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resp.Body, &group); err != nil {
		return "", err
	}
	return group.ID, nil
}

func (o *OktaAPI) AssignGroupToApp(ctx context.Context, appID, groupID string) error {
	endpoint := fmt.Sprintf("%s/api/v1/apps/%s/groups", o.BaseURL, appID)
	body := map[string]interface{}{"id": groupID}
	resp, err := o.doRequest(ctx, coreapi.POST, endpoint, body)
	if err != nil {
		return err
	}
	if resp.Code == http.StatusConflict {
		return nil
	}
	if !isStatus(resp.Code, http.StatusOK, http.StatusCreated, http.StatusNoContent) {
		return o.unexpectedStatusError(coreapi.POST, endpoint, resp)
	}
	return nil
}

func (o *OktaAPI) UnassignGroupFromApp(ctx context.Context, appID, groupID string) error {
	endpoint := fmt.Sprintf("%s/api/v1/apps/%s/groups/%s", o.BaseURL, appID, groupID)
	resp, err := o.doRequest(ctx, coreapi.DELETE, endpoint, nil)
	if err != nil {
		return err
	}
	if resp.Code == http.StatusNotFound {
		return nil
	}
	if !isStatus(resp.Code, http.StatusOK, http.StatusNoContent) {
		return o.unexpectedStatusError(coreapi.DELETE, endpoint, resp)
	}
	return nil
}

func (o *OktaAPI) CreatePolicy(ctx context.Context, authServerID string, policy map[string]interface{}) (string, error) {
	endpoint := fmt.Sprintf("%s/api/v1/authorizationServers/%s/policies", o.BaseURL, authServerID)
	resp, err := o.doRequest(ctx, coreapi.POST, endpoint, policy)
	if err != nil {
		return "", err
	}
	if resp.Code == http.StatusConflict {
		return "", nil
	}
	if !isStatus(resp.Code, http.StatusOK, http.StatusCreated) {
		return "", o.unexpectedStatusError(coreapi.POST, endpoint, resp)
	}
	var pol struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resp.Body, &pol); err != nil {
		return "", err
	}
	return pol.ID, nil
}

func (o *OktaAPI) CreateRule(ctx context.Context, authServerID, policyID string, rule map[string]interface{}) (string, error) {
	endpoint := fmt.Sprintf("%s/api/v1/authorizationServers/%s/policies/%s/rules", o.BaseURL, authServerID, policyID)
	resp, err := o.doRequest(ctx, coreapi.POST, endpoint, rule)
	if err != nil {
		return "", err
	}
	if resp.Code == http.StatusConflict {
		return "", nil
	}
	if !isStatus(resp.Code, http.StatusOK, http.StatusCreated) {
		return "", o.unexpectedStatusError(coreapi.POST, endpoint, resp)
	}
	var r struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resp.Body, &r); err != nil {
		return "", err
	}
	return r.ID, nil
}

func (o *OktaAPI) DeleteRule(ctx context.Context, authServerID, policyID, ruleID string) error {
	endpoint := fmt.Sprintf("%s/api/v1/authorizationServers/%s/policies/%s/rules/%s", o.BaseURL, authServerID, policyID, ruleID)
	resp, err := o.doRequest(ctx, coreapi.DELETE, endpoint, nil)
	if err != nil {
		return err
	}
	if resp.Code == http.StatusNotFound {
		return nil
	}
	if !isStatus(resp.Code, http.StatusOK, http.StatusNoContent) {
		return o.unexpectedStatusError(coreapi.DELETE, endpoint, resp)
	}
	return nil
}

func (o *OktaAPI) DeletePolicy(ctx context.Context, authServerID, policyID string) error {
	endpoint := fmt.Sprintf("%s/api/v1/authorizationServers/%s/policies/%s", o.BaseURL, authServerID, policyID)
	resp, err := o.doRequest(ctx, coreapi.DELETE, endpoint, nil)
	if err != nil {
		return err
	}
	if resp.Code == http.StatusNotFound {
		return nil
	}
	if !isStatus(resp.Code, http.StatusOK, http.StatusNoContent) {
		return o.unexpectedStatusError(coreapi.DELETE, endpoint, resp)
	}
	return nil
}

func (o *OktaAPI) CreateScope(ctx context.Context, authServerID string, scope map[string]interface{}) (string, error) {
	endpoint := fmt.Sprintf("%s/api/v1/authorizationServers/%s/scopes", o.BaseURL, authServerID)
	resp, err := o.doRequest(ctx, coreapi.POST, endpoint, scope)
	if err != nil {
		return "", err
	}
	if resp.Code == http.StatusConflict {
		return "", nil
	}
	if !isStatus(resp.Code, http.StatusOK, http.StatusCreated) {
		return "", o.unexpectedStatusError(coreapi.POST, endpoint, resp)
	}
	var s struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resp.Body, &s); err != nil {
		return "", err
	}
	return s.ID, nil
}

func (o *OktaAPI) DeleteScope(ctx context.Context, authServerID, scopeID string) error {
	endpoint := fmt.Sprintf("%s/api/v1/authorizationServers/%s/scopes/%s", o.BaseURL, authServerID, scopeID)
	resp, err := o.doRequest(ctx, coreapi.DELETE, endpoint, nil)
	if err != nil {
		return err
	}
	if resp.Code == http.StatusNotFound {
		return nil
	}
	if !isStatus(resp.Code, http.StatusOK, http.StatusNoContent) {
		return o.unexpectedStatusError(coreapi.DELETE, endpoint, resp)
	}
	return nil
}
