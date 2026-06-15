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

type Okta struct {
	BaseURL  string
	APIToken string
	Client   coreapi.Client
	logger   log.FieldLogger
}

type oktaGroupProfile struct {
	Name string `json:"name"`
}

type oktaGroupSearchResult struct {
	ID      string           `json:"id"`
	Profile oktaGroupProfile `json:"profile"`
}

type oktaPolicyListResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type oktaPolicyConditionsClients struct {
	Include []string `json:"include"`
}

type oktaPolicyConditions struct {
	Clients *oktaPolicyConditionsClients `json:"clients,omitempty"`
}

type oktaCreatePolicyRequest struct {
	Name       string               `json:"name"`
	Type       string               `json:"type"`
	Status     string               `json:"status"`
	Priority   int                  `json:"priority"`
	Conditions oktaPolicyConditions `json:"conditions"`
}

type oktaPolicyRuleConditionGroups struct {
	Include []string `json:"include"`
}

type oktaPolicyRuleConditionPeople struct {
	Groups oktaPolicyRuleConditionGroups `json:"groups"`
}

type oktaPolicyRuleConditionGrantTypes struct {
	Include []string `json:"include"`
}

type oktaPolicyRuleConditionScopes struct {
	Include []string `json:"include"`
}

type oktaPolicyRuleConditions struct {
	People     oktaPolicyRuleConditionPeople     `json:"people"`
	GrantTypes oktaPolicyRuleConditionGrantTypes `json:"grantTypes"`
	Scopes     oktaPolicyRuleConditionScopes     `json:"scopes"`
}

type oktaPolicyRuleActionToken struct {
	AccessTokenLifetimeMinutes int `json:"accessTokenLifetimeMinutes"`
}

type oktaPolicyRuleActions struct {
	Token oktaPolicyRuleActionToken `json:"token"`
}

type oktaCreatePolicyRuleRequest struct {
	Name       string                   `json:"name"`
	Type       string                   `json:"type"`
	Conditions oktaPolicyRuleConditions `json:"conditions"`
	Actions    oktaPolicyRuleActions    `json:"actions"`
}

func New(apiClient coreapi.Client, baseURL, apiToken string) *Okta {
	if apiClient == nil {
		apiClient = coreapi.NewClient(nil, "")
	}
	return &Okta{
		BaseURL:  baseURL,
		APIToken: apiToken,
		Client:   apiClient,
		logger:   log.NewFieldLogger().WithComponent("oktaClient").WithPackage("sdk.agent.authz.oauth.clients"),
	}
}

func (o *Okta) authServerPolicyEndpoint(authServerID, policyID string) string {
	return fmt.Sprintf("%s/api/v1/authorizationServers/%s/policies/%s", o.BaseURL, authServerID, policyID)
}

func (o *Okta) authServerPoliciesEndpoint(authServerID string) string {
	return fmt.Sprintf("%s/api/v1/authorizationServers/%s/policies", o.BaseURL, authServerID)
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
			"Authorization": fmt.Sprintf("%s %s", OktaAuthHeaderPrefix, o.APIToken),
			"Content-Type":  "application/json",
		},
		Body: reqBody,
	}

	return o.Client.Send(request)
}

func (o *Okta) doGetJSON(endpoint string, out interface{}) error {
	if out == nil {
		return nil
	}

	resp, err := o.doRequest(coreapi.GET, endpoint, nil)
	if err != nil {
		return err
	}

	if !isStatus(resp.Code, http.StatusOK) {
		return o.unexpectedStatusError(coreapi.GET, endpoint, resp)
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

// Two-step: list to find the policy ID, then fetch the full object so callers can update it without an extra round-trip.
func (o *Okta) FindPolicyByName(authServerID, policyName string) (map[string]interface{}, error) {
	policyName = strings.TrimSpace(policyName)
	if authServerID == "" || policyName == "" {
		return nil, nil
	}
	endpoint := o.authServerPoliciesEndpoint(authServerID)

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

// No-ops if the policy already includes the client or is assigned to ALL_CLIENTS.
func (o *Okta) AssignClientToPolicy(authServerID string, policy map[string]interface{}, clientID string) error {
	clientID = strings.TrimSpace(clientID)
	if authServerID == "" || policy == nil || clientID == "" {
		return fmt.Errorf("invalid input for policy assignment")
	}

	policyID, _ := policy["id"].(string)
	policyID = strings.TrimSpace(policyID)
	if policyID == "" {
		return fmt.Errorf("invalid input for policy assignment")
	}

	policyLogger := o.logger.
		WithField("authServerID", authServerID).
		WithField("policyID", policyID).
		WithField("clientID", clientID)

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

	if includeHasAllClients(include) {
		policyLogger.Trace("policy assignment already includes ALL_CLIENTS. Skipping client-specific policy update")
		return nil
	}

	if includeHasClient(include, clientID) {
		policyLogger.Trace("policy assignment already includes client. Skipping client-specific policy update")
		return nil
	}
	clients["include"] = append(include, clientID)
	return o.UpdatePolicy(authServerID, policyID, policy)
}

func (o *Okta) CreatePolicy(authServerID, name string, clientID string) (map[string]interface{}, error) {
	endpoint := o.authServerPoliciesEndpoint(authServerID)
	o.logger.
		WithField("authServerID", authServerID).
		WithField("policyName", name).
		WithField("endpoint", endpoint).
		Trace("creating Okta authorization server policy")
	req := oktaCreatePolicyRequest{
		Name:     name,
		Type:     "OAUTH_AUTHORIZATION_POLICY",
		Status:   "ACTIVE",
		Priority: 1, //SDB - check to see if it defaults to 1
		Conditions: oktaPolicyConditions{
			Clients: &oktaPolicyConditionsClients{Include: []string{clientID}},
		},
	}
	resp, err := o.doRequest(coreapi.POST, endpoint, req)
	if err != nil {
		return nil, err
	}
	if !isStatus(resp.Code, http.StatusCreated) {
		return nil, o.unexpectedStatusError(coreapi.POST, endpoint, resp)
	}
	var policy map[string]interface{}
	if err := json.Unmarshal(resp.Body, &policy); err != nil {
		return nil, err
	}
	return policy, nil
}

func (o *Okta) CreatePolicyRule(authServerID, policyID, name, grantType, scope string) error {
	endpoint := fmt.Sprintf("%s/api/v1/authorizationServers/%s/policies/%s/rules", o.BaseURL, authServerID, policyID)
	o.logger.
		WithField("authServerID", authServerID).
		WithField("policyID", policyID).
		WithField("ruleName", name).
		WithField("endpoint", endpoint).
		Trace("creating Okta authorization server policy rule")
	req := oktaCreatePolicyRuleRequest{
		Name: name,
		Type: "RESOURCE_ACCESS",
		Conditions: oktaPolicyRuleConditions{
			People:     oktaPolicyRuleConditionPeople{Groups: oktaPolicyRuleConditionGroups{Include: []string{"EVERYONE"}}},
			GrantTypes: oktaPolicyRuleConditionGrantTypes{Include: []string{grantType}},
			Scopes:     oktaPolicyRuleConditionScopes{Include: []string{scope}},
		},
		Actions: oktaPolicyRuleActions{
			Token: oktaPolicyRuleActionToken{AccessTokenLifetimeMinutes: 60}, // SDB - see if this defaults to 60
		},
	}
	resp, err := o.doRequest(coreapi.POST, endpoint, req)
	if err != nil {
		return err
	}
	if !isStatus(resp.Code, http.StatusCreated) {
		return o.unexpectedStatusError(coreapi.POST, endpoint, resp)
	}
	return nil
}

// The policy is never deleted, even when the include list becomes empty.
func (o *Okta) RemoveClientFromPolicy(authServerID string, policy map[string]interface{}, clientID string) error {
	clientID = strings.TrimSpace(clientID)
	policyID, _ := policy["id"].(string)
	policyID = strings.TrimSpace(policyID)
	if authServerID == "" || policyID == "" || clientID == "" {
		return fmt.Errorf("invalid input for client removal from policy")
	}
	o.logger.
		WithField("authServerID", authServerID).
		WithField("policyID", policyID).
		WithField("clientID", clientID).
		Trace("removing client from Okta authorization server policy")

	conditions := ensureMap(policy, "conditions")
	clients := ensureMap(conditions, "clients")

	includeRaw, _ := clients["include"]
	include, ok := includeRaw.([]interface{})
	if !ok || !includeHasClient(include, clientID) {
		return nil
	}

	filtered := make([]interface{}, 0, len(include))
	for _, v := range include {
		s, _ := v.(string)
		if strings.TrimSpace(s) != clientID {
			filtered = append(filtered, v)
		}
	}
	clients["include"] = filtered
	return o.UpdatePolicy(authServerID, policyID, policy)
}

// DeactivateApp deactivates an Okta application. A 404 response is treated as success.
// DeactivateApp must be called before DeleteApp.
func (o *Okta) DeactivateApp(appID string) error {
	endpoint := fmt.Sprintf("%s/api/v1/apps/%s/lifecycle/deactivate", o.BaseURL, appID)
	o.logger.WithField("appID", appID).WithField("endpoint", endpoint).Trace("deactivating Okta app")
	resp, err := o.doRequest(coreapi.POST, endpoint, nil)
	if err != nil {
		return err
	}
	if isStatus(resp.Code, http.StatusNotFound) {
		return nil
	}
	if !isStatus(resp.Code, http.StatusOK, http.StatusNoContent) {
		return o.unexpectedStatusError(coreapi.POST, endpoint, resp)
	}
	return nil
}

// DeleteApp deletes an Okta application. A 404 response is treated as success.
// DeactivateApp must be called before this method.
func (o *Okta) DeleteApp(appID string) error {
	endpoint := fmt.Sprintf("%s/api/v1/apps/%s", o.BaseURL, appID)
	o.logger.WithField("appID", appID).WithField("endpoint", endpoint).Trace("deleting Okta app")
	resp, err := o.doRequest(coreapi.DELETE, endpoint, nil)
	if err != nil {
		return err
	}
	if isStatus(resp.Code, http.StatusNotFound) {
		return nil
	}
	if !isStatus(resp.Code, http.StatusNoContent) {
		return o.unexpectedStatusError(coreapi.DELETE, endpoint, resp)
	}
	return nil
}

// FindGroupByName searches Okta for a group with the given name and returns its ID.
// Returns ("", nil) when no matching group exists.
func (o *Okta) FindGroupByName(groupName string) (string, error) {
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		return "", nil
	}
	endpoint := fmt.Sprintf("%s/api/v1/groups?q=%s", o.BaseURL, url.QueryEscape(groupName))
	o.logger.WithField("groupName", groupName).WithField("endpoint", endpoint).Trace("searching for Okta group")

	var groups []oktaGroupSearchResult
	if err := o.doGetJSON(endpoint, &groups); err != nil {
		return "", err
	}

	for _, g := range groups {
		if g.Profile.Name == groupName {
			return g.ID, nil
		}
	}
	return "", nil
}

// AssignGroupToApp adds an Okta application to the given group.
func (o *Okta) AssignGroupToApp(appID, groupID string) error {
	endpoint := fmt.Sprintf("%s/api/v1/apps/%s/groups/%s", o.BaseURL, appID, groupID)
	o.logger.WithField("appID", appID).WithField("groupID", groupID).WithField("endpoint", endpoint).Trace("assigning group to Okta app")
	resp, err := o.doRequest(coreapi.PUT, endpoint, nil)
	if err != nil {
		return err
	}
	if !isStatus(resp.Code, http.StatusOK, http.StatusCreated) {
		return o.unexpectedStatusError(coreapi.PUT, endpoint, resp)
	}
	return nil
}

// UnassignGroupFromApp removes an Okta application from the given group.
// A 404 response is treated as success (app or group already gone).
func (o *Okta) UnassignGroupFromApp(appID, groupID string) error {
	endpoint := fmt.Sprintf("%s/api/v1/apps/%s/groups/%s", o.BaseURL, appID, groupID)
	o.logger.WithField("appID", appID).WithField("groupID", groupID).WithField("endpoint", endpoint).Trace("removing app from Okta group")
	resp, err := o.doRequest(coreapi.DELETE, endpoint, nil)
	if err != nil {
		return err
	}
	if isStatus(resp.Code, http.StatusNotFound) {
		return nil
	}
	if !isStatus(resp.Code, http.StatusNoContent, http.StatusOK) {
		return o.unexpectedStatusError(coreapi.DELETE, endpoint, resp)
	}
	return nil
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
	return includeHasClient(include, "ALL_CLIENTS")
}

func includeHasClient(include []interface{}, clientID string) bool {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return false
	}
	for _, v := range include {
		s, ok := v.(string)
		if ok && strings.TrimSpace(s) == clientID {
			return true
		}
	}
	return false
}
