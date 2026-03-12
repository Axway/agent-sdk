package clients

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const OktaAuthHeaderPrefix = "SSWS"

var logger = log.NewFieldLogger().WithComponent("oktaClient").WithPackage("sdk.agent.authz.oauth.clients")

// Okta wraps Okta Management API calls.
type Okta struct {
	BaseURL  string
	APIToken string
	Client   coreapi.Client
}

type oktaGroupSearchResult struct {
	ID      string           `json:"id"`
	Profile oktaGroupProfile `json:"profile"`
}

type oktaGroupProfile struct {
	Name string `json:"name"`
}

type oktaPolicyListResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func New(apiClient coreapi.Client, baseURL, apiToken string) *Okta {
	if apiClient == nil {
		apiClient = coreapi.NewClient(nil, "")
	}
	return &Okta{
		BaseURL:  baseURL,
		APIToken: apiToken,
		Client:   apiClient,
	}
}

func (o *Okta) doRequest(method, endpoint string, body interface{}) (*coreapi.Response, error) {
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

	return o.Client.Send(request)
}

func (o *Okta) doGetJSON(endpoint string, out interface{}) error {
	resp, err := o.doRequest(coreapi.GET, endpoint, nil)
	if err != nil {
		return err
	}
	if !isStatus(resp.Code, http.StatusOK) {
		return o.unexpectedStatusError(coreapi.GET, endpoint, resp)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(resp.Body, out)
}

func (o *Okta) unexpectedStatusError(method, endpoint string, resp *coreapi.Response) error {
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

func (o *Okta) authServerPolicyEndpoint(authServerID, policyID string) string {
	return fmt.Sprintf("%s/api/v1/authorizationServers/%s/policies/%s", o.BaseURL, authServerID, policyID)
}

func (o *Okta) FindGroupByName(groupName string) (string, error) {
	endpoint := fmt.Sprintf("%s/api/v1/groups?q=%s", o.BaseURL, url.QueryEscape(groupName))
	resp, err := o.doRequest(coreapi.GET, endpoint, nil)
	if err != nil {
		return "", err
	}
	if !isStatus(resp.Code, http.StatusOK) {
		return "", o.unexpectedStatusError(coreapi.GET, endpoint, resp)
	}

	var groups []oktaGroupSearchResult
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

func (o *Okta) AssignGroupToApp(appID, groupID string) error {
	endpoint := fmt.Sprintf("%s/api/v1/apps/%s/groups/%s", o.BaseURL, appID, groupID)
	resp, err := o.doRequest(coreapi.PUT, endpoint, nil)
	if err != nil {
		return err
	}
	if resp.Code == http.StatusConflict {
		return nil
	}
	if !isStatus(resp.Code, http.StatusOK, http.StatusCreated, http.StatusNoContent) {
		return o.unexpectedStatusError(coreapi.PUT, endpoint, resp)
	}
	return nil
}

func (o *Okta) UnassignGroupFromApp(appID, groupID string) error {
	endpoint := fmt.Sprintf("%s/api/v1/apps/%s/groups/%s", o.BaseURL, appID, groupID)
	resp, err := o.doRequest(coreapi.DELETE, endpoint, nil)
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

// FindPolicyByName returns the policy object for the given policy name on the authorization server.
// Returns nil if not found.
//
// Note: This does a list call to locate the policy ID and then retrieves the policy by ID
// so callers can update it without needing an additional fetch.
func (o *Okta) FindPolicyByName(authServerID, policyName string) (map[string]interface{}, error) {
	policyName = strings.TrimSpace(policyName)
	if authServerID == "" || policyName == "" {
		return nil, nil
	}
	endpoint := fmt.Sprintf("%s/api/v1/authorizationServers/%s/policies", o.BaseURL, authServerID)
	
	var policies []oktaPolicyListResult
	if err := o.doGetJSON(endpoint, &policies); err != nil {
		return nil, err
	}

	policyID := ""
	for _, p := range policies {
		if p.Name == policyName {
			policyID = p.ID
			break
		}
	}
	if policyID == "" {
		return nil, nil
	}

	policyEndpoint := o.authServerPolicyEndpoint(authServerID, policyID)
	var policy map[string]interface{}
	if err := o.doGetJSON(policyEndpoint, &policy); err != nil {
		return nil, err
	}
	return policy, nil
}

// UpdatePolicy updates an existing authorization server policy.
func (o *Okta) UpdatePolicy(authServerID, policyID string, policy map[string]interface{}) error {
	if authServerID == "" || policyID == "" {
		return nil
	}
	endpoint := o.authServerPolicyEndpoint(authServerID, policyID)
	resp, err := o.doRequest(coreapi.PUT, endpoint, policy)
	if err != nil {
		return err
	}
	if !isStatus(resp.Code, http.StatusOK) {
		return o.unexpectedStatusError(coreapi.PUT, endpoint, resp)
	}
	return nil
}

// AssignClientToPolicy updates the policy-level "Assigned to clients" list to include the given client.
// If the policy is already assigned to ALL_CLIENTS or already includes the client, it no-ops.
//
// The policy map is modified in-place and then persisted via UpdatePolicy.
func (o *Okta) AssignClientToPolicy(authServerID string, policy map[string]interface{}, clientID string) error {
	clientID = strings.TrimSpace(clientID)
	if authServerID == "" || policy == nil || clientID == "" {
		return nil
	}
	policyID, _ := policy["id"].(string)
	policyID = strings.TrimSpace(policyID)
	if policyID == "" {
		return nil
	}

	conditions := ensureMap(policy, "conditions")
	clients := ensureMap(conditions, "clients")

	includeRaw, hasInclude := clients["include"]
	if !hasInclude || includeRaw == nil {
		clients["include"] = []interface{}{clientID}
		return o.UpdatePolicy(authServerID, policyID, policy)
	}

	include, ok := includeRaw.([]interface{})
	if !ok {
		// Unexpected type; avoid breaking existing policy structure.
		return nil
	}

	// check if policy has all clients configured
	if includeHasAllClients(include) {
		logger.
			WithField("authServerID", authServerID).
			WithField("policyID", policyID).
			WithField("clientID", clientID).
			Trace("policy assignment already includes ALL_CLIENTS. Skipping client-specific policy update")
		return nil
	}

	// check if client is already included in policy assignment
	if includeHasClient(include, clientID) {
		logger.
			WithField("authServerID", authServerID).
			WithField("policyID", policyID).
			WithField("clientID", clientID).
			Trace("policy assignment already includes client. Skipping client-specific policy update")
		return nil
	}
	clients["include"] = append(include, clientID)
	return o.UpdatePolicy(authServerID, policyID, policy)
}

func ensureMap(parent map[string]interface{}, key string) map[string]interface{} {
	child, ok := parent[key].(map[string]interface{})
	if ok && child != nil {
		return child
	}
	child = make(map[string]interface{})
	parent[key] = child
	return child
}

func includeHasAllClients(include []interface{}) bool {
	for _, v := range include {
		s, ok := v.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s == "ALL_CLIENTS" {
			return true
		}
	}
	return false
}

func includeHasClient(include []interface{}, clientID string) bool {
	for _, v := range include {
		s, ok := v.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s == clientID {
			return true
		}
	}
	return false
}
