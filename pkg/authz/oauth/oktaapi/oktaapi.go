package oktaapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

func (o *OktaAPI) doRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "SSWS "+o.APIToken)
	req.Header.Set("Content-Type", "application/json")
	return o.Client.Do(req)
}

// unexpectedStatusError creates a useful error message when Okta returns a non-success
// HTTP status.
//
// added this because provisioning/deprovisioning flows previously could "silently"
// succeed even when Okta rejected the operation (e.g., 403/404/5xx).
func (o *OktaAPI) unexpectedStatusError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("okta api %s %s returned %d: %s", resp.Request.Method, resp.Request.URL.String(), resp.StatusCode, string(body))
}

// isStatus is a small helper to make status-code allow-lists readable
func isStatus(code int, allowed ...int) bool {
	for _, c := range allowed {
		if code == c {
			return true
		}
	}
	return false
}

// FindGroupByName returns group ID for a given group name
func (o *OktaAPI) FindGroupByName(ctx context.Context, groupName string) (string, error) {
	endpoint := fmt.Sprintf("%s/api/v1/groups?q=%s", o.BaseURL, url.QueryEscape(groupName))
	resp, err := o.doRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if !isStatus(resp.StatusCode, http.StatusOK) {
		return "", o.unexpectedStatusError(resp)
	}
	var groups []struct {
		ID      string `json:"id"`
		Profile struct {
			Name string `json:"name"`
		} `json:"profile"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		return "", err
	}
	for _, g := range groups {
		if g.Profile.Name == groupName {
			return g.ID, nil
		}
	}
	return "", nil // not found
}

// CreateGroup creates a new group and returns its ID
func (o *OktaAPI) CreateGroup(ctx context.Context, name, description string) (string, error) {
	endpoint := fmt.Sprintf("%s/api/v1/groups", o.BaseURL)
	body := map[string]interface{}{
		"profile": map[string]interface{}{
			"name":        name,
			"description": description,
		},
	}
	resp, err := o.doRequest(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return o.FindGroupByName(ctx, name)
	}
	if !isStatus(resp.StatusCode, http.StatusOK, http.StatusCreated) {
		return "", o.unexpectedStatusError(resp)
	}
	var group struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&group); err != nil {
		return "", err
	}
	return group.ID, nil
}

// AssignGroupToApp assigns a group to an app
func (o *OktaAPI) AssignGroupToApp(ctx context.Context, appID, groupID string) error {
	endpoint := fmt.Sprintf("%s/api/v1/apps/%s/groups", o.BaseURL, appID)
	body := map[string]interface{}{"id": groupID}
	resp, err := o.doRequest(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return nil // already assigned
	}
	if !isStatus(resp.StatusCode, http.StatusOK, http.StatusCreated, http.StatusNoContent) {
		return o.unexpectedStatusError(resp)
	}
	return nil
}

// UnassignGroupFromApp removes a group from an app
func (o *OktaAPI) UnassignGroupFromApp(ctx context.Context, appID, groupID string) error {
	endpoint := fmt.Sprintf("%s/api/v1/apps/%s/groups/%s", o.BaseURL, appID, groupID)
	resp, err := o.doRequest(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil // already unassigned
	}
	if !isStatus(resp.StatusCode, http.StatusOK, http.StatusNoContent) {
		return o.unexpectedStatusError(resp)
	}
	return nil
}

// CreatePolicy creates a policy and returns its ID
func (o *OktaAPI) CreatePolicy(ctx context.Context, authServerID string, policy map[string]interface{}) (string, error) {
	endpoint := fmt.Sprintf("%s/api/v1/authorizationServers/%s/policies", o.BaseURL, authServerID)
	resp, err := o.doRequest(ctx, http.MethodPost, endpoint, policy)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return "", nil // treat as success
	}
	if !isStatus(resp.StatusCode, http.StatusOK, http.StatusCreated) {
		return "", o.unexpectedStatusError(resp)
	}
	var pol struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pol); err != nil {
		return "", err
	}
	return pol.ID, nil
}

// CreateRule creates a rule under a policy and returns its ID
func (o *OktaAPI) CreateRule(ctx context.Context, authServerID, policyID string, rule map[string]interface{}) (string, error) {
	endpoint := fmt.Sprintf("%s/api/v1/authorizationServers/%s/policies/%s/rules", o.BaseURL, authServerID, policyID)
	resp, err := o.doRequest(ctx, http.MethodPost, endpoint, rule)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return "", nil // treat as success
	}
	if !isStatus(resp.StatusCode, http.StatusOK, http.StatusCreated) {
		return "", o.unexpectedStatusError(resp)
	}
	var r struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}
	return r.ID, nil
}

// DeleteRule deletes a rule
func (o *OktaAPI) DeleteRule(ctx context.Context, authServerID, policyID, ruleID string) error {
	endpoint := fmt.Sprintf("%s/api/v1/authorizationServers/%s/policies/%s/rules/%s", o.BaseURL, authServerID, policyID, ruleID)
	resp, err := o.doRequest(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil // already deleted
	}
	if !isStatus(resp.StatusCode, http.StatusOK, http.StatusNoContent) {
		return o.unexpectedStatusError(resp)
	}
	return nil
}

// DeletePolicy deletes a policy
func (o *OktaAPI) DeletePolicy(ctx context.Context, authServerID, policyID string) error {
	endpoint := fmt.Sprintf("%s/api/v1/authorizationServers/%s/policies/%s", o.BaseURL, authServerID, policyID)
	resp, err := o.doRequest(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil // already deleted
	}
	if !isStatus(resp.StatusCode, http.StatusOK, http.StatusNoContent) {
		return o.unexpectedStatusError(resp)
	}
	return nil
}

// CreateScope creates a scope and returns its ID
func (o *OktaAPI) CreateScope(ctx context.Context, authServerID string, scope map[string]interface{}) (string, error) {
	endpoint := fmt.Sprintf("%s/api/v1/authorizationServers/%s/scopes", o.BaseURL, authServerID)
	resp, err := o.doRequest(ctx, http.MethodPost, endpoint, scope)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return "", nil // treat as success
	}
	if !isStatus(resp.StatusCode, http.StatusOK, http.StatusCreated) {
		return "", o.unexpectedStatusError(resp)
	}
	var s struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return "", err
	}
	return s.ID, nil
}

// DeleteScope deletes a scope
func (o *OktaAPI) DeleteScope(ctx context.Context, authServerID, scopeID string) error {
	endpoint := fmt.Sprintf("%s/api/v1/authorizationServers/%s/scopes/%s", o.BaseURL, authServerID, scopeID)
	resp, err := o.doRequest(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil // already deleted
	}
	if !isStatus(resp.StatusCode, http.StatusOK, http.StatusNoContent) {
		return o.unexpectedStatusError(resp)
	}
	return nil
}
