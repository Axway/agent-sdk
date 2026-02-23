package oktaapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// OktaAPI wraps Okta Management API calls
type OktaAPI struct {
	BaseURL  string
	APIToken string
	Client   *http.Client
}

func New(baseURL, apiToken string) *OktaAPI {
	return &OktaAPI{
		BaseURL:  baseURL,
		APIToken: apiToken,
		Client:   &http.Client{},
	}
}

func (o *OktaAPI) doRequest(ctx context.Context, method, url string, body interface{}) (*http.Response, error) {
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "SSWS "+o.APIToken)
	req.Header.Set("Content-Type", "application/json")
	return o.Client.Do(req)
}

// FindGroupByName returns group ID for a given group name
func (o *OktaAPI) FindGroupByName(ctx context.Context, groupName string) (string, error) {
	url := fmt.Sprintf("%s/api/v1/groups?q=%s", o.BaseURL, groupName)
	resp, err := o.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var groups []struct {
		Id      string `json:"id"`
		Profile struct {
			Name string `json:"name"`
		} `json:"profile"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		return "", err
	}
	for _, g := range groups {
		if g.Profile.Name == groupName {
			return g.Id, nil
		}
	}
	return "", nil // not found
}

// CreateGroup creates a new group and returns its ID
func (o *OktaAPI) CreateGroup(ctx context.Context, name, description string) (string, error) {
	url := fmt.Sprintf("%s/api/v1/groups", o.BaseURL)
	body := map[string]interface{}{
		"profile": map[string]interface{}{
			"name":        name,
			"description": description,
		},
	}
	resp, err := o.doRequest(ctx, "POST", url, body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return o.FindGroupByName(ctx, name)
	}
	var group struct {
		Id string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&group); err != nil {
		return "", err
	}
	return group.Id, nil
}

// AssignGroupToApp assigns a group to an app
func (o *OktaAPI) AssignGroupToApp(ctx context.Context, appId, groupId string) error {
	url := fmt.Sprintf("%s/api/v1/apps/%s/groups", o.BaseURL, appId)
	body := map[string]interface{}{"id": groupId}
	resp, err := o.doRequest(ctx, "POST", url, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return nil // already assigned
	}
	return nil
}

// UnassignGroupFromApp removes a group from an app
func (o *OktaAPI) UnassignGroupFromApp(ctx context.Context, appId, groupId string) error {
	url := fmt.Sprintf("%s/api/v1/apps/%s/groups/%s", o.BaseURL, appId, groupId)
	resp, err := o.doRequest(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// CreatePolicy creates a policy and returns its ID
func (o *OktaAPI) CreatePolicy(ctx context.Context, authServerId string, policy map[string]interface{}) (string, error) {
	url := fmt.Sprintf("%s/api/v1/authorizationServers/%s/policies", o.BaseURL, authServerId)
	resp, err := o.doRequest(ctx, "POST", url, policy)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return "", nil // treat as success
	}
	var pol struct {
		Id string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pol); err != nil {
		return "", err
	}
	return pol.Id, nil
}

// CreateRule creates a rule under a policy and returns its ID
func (o *OktaAPI) CreateRule(ctx context.Context, authServerId, policyId string, rule map[string]interface{}) (string, error) {
	url := fmt.Sprintf("%s/api/v1/authorizationServers/%s/policies/%s/rules", o.BaseURL, authServerId, policyId)
	resp, err := o.doRequest(ctx, "POST", url, rule)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return "", nil // treat as success
	}
	var r struct {
		Id string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}
	return r.Id, nil
}

// DeleteRule deletes a rule
func (o *OktaAPI) DeleteRule(ctx context.Context, authServerId, policyId, ruleId string) error {
	url := fmt.Sprintf("%s/api/v1/authorizationServers/%s/policies/%s/rules/%s", o.BaseURL, authServerId, policyId, ruleId)
	resp, err := o.doRequest(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// DeletePolicy deletes a policy
func (o *OktaAPI) DeletePolicy(ctx context.Context, authServerId, policyId string) error {
	url := fmt.Sprintf("%s/api/v1/authorizationServers/%s/policies/%s", o.BaseURL, authServerId, policyId)
	resp, err := o.doRequest(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// CreateScope creates a scope and returns its ID
func (o *OktaAPI) CreateScope(ctx context.Context, authServerId string, scope map[string]interface{}) (string, error) {
	url := fmt.Sprintf("%s/api/v1/authorizationServers/%s/scopes", o.BaseURL, authServerId)
	resp, err := o.doRequest(ctx, "POST", url, scope)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return "", nil // treat as success
	}
	var s struct {
		Id string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return "", err
	}
	return s.Id, nil
}

// DeleteScope deletes a scope
func (o *OktaAPI) DeleteScope(ctx context.Context, authServerId, scopeId string) error {
	url := fmt.Sprintf("%s/api/v1/authorizationServers/%s/scopes/%s", o.BaseURL, authServerId, scopeId)
	resp, err := o.doRequest(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
