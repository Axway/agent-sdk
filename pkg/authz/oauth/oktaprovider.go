package oauth

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/authz/oauth/clients"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

const (
	oktaApplicationType = "application_type"
	oktaAppTypeService  = "service"
	oktaAppTypeWeb      = "web"
	oktaAppTypeBrowser  = "browser"
	oktaPKCERequired    = "pkce_required"
	oktaSpa             = "okta-spa"
)

type okta struct{}

type oktaPolicyConfig struct {
	CreatePolicy   *bool                  `json:"createPolicy,omitempty"`
	Enabled        *bool                  `json:"enabled,omitempty"`
	AuthServerID   string                 `json:"authServerId,omitempty"`
	PolicyTemplate map[string]interface{} `json:"policyTemplate,omitempty"`
}

func parseOktaPolicyConfig(policyJSON string) (*oktaPolicyConfig, error) {
	if strings.TrimSpace(policyJSON) == "" {
		return nil, nil
	}
	cfg := &oktaPolicyConfig{}
	if err := json.Unmarshal([]byte(policyJSON), cfg); err != nil {
		return nil, fmt.Errorf("invalid okta policy config: %w", err)
	}
	return cfg, nil
}

func (c *oktaPolicyConfig) isEnabled() bool {
	if c == nil {
		return false
	}
	if c.CreatePolicy != nil {
		return *c.CreatePolicy
	}
	if c.Enabled != nil {
		return *c.Enabled
	}
	return true
}

func oktaBaseURLFromMetadataURL(metadataURL string) (string, error) {
	if metadataURL == "" {
		return "", nil
	}
	u, err := url.Parse(metadataURL)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid metadata URL: %q", metadataURL)
	}
	return u.Scheme + "://" + u.Host, nil
}

func oktaAuthServerIDFromMetadataURL(metadataURL string) string {
	if metadataURL == "" {
		return ""
	}
	u, err := url.Parse(metadataURL)
	if err != nil {
		return ""
	}
	path := strings.Trim(u.Path, "/")
	if path == "" {
		return ""
	}
	segments := strings.Split(path, "/")
	for i := 0; i < len(segments)-1; i++ {
		if segments[i] == "oauth2" {
			return segments[i+1]
		}
	}
	return ""
}

// postProcessClientRegistration handles Okta provisioning after client registration
func (i *okta) postProcessClientRegistration(clientRes ClientMetadata, credentialObj interface{}, apiClient coreapi.Client) (map[string]string, error) {
	var metadataURL string
	var apiToken string
	var groupName string
	var policyJSON string

	if cfg, ok := credentialObj.(corecfg.IDPConfig); ok {
		metadataURL = cfg.GetMetadataURL()
		if authCfg := cfg.GetAuthConfig(); authCfg != nil {
			apiToken = authCfg.GetAccessToken()
		}
		groupName = cfg.GetOktaGroup()
		policyJSON = cfg.GetOktaPolicy()
	}

	baseURL, err := oktaBaseURLFromMetadataURL(metadataURL)
	if err != nil {
		return nil, err
	}
	if baseURL == "" || apiToken == "" {
		return nil, nil // skip if not configured
	}
	oktaClient := clients.New(apiClient, baseURL, apiToken)
	created := make(map[string]string)

	policyCfg, err := parseOktaPolicyConfig(policyJSON)
	if err != nil {
		return nil, err
	}

	if groupName != "" {
		if err := i.handleGroupAssignment(oktaClient, clientRes, groupName, created); err != nil {
			return nil, err
		}
	}
	if policyCfg != nil && policyCfg.isEnabled() {
		if err := i.handlePolicyRuleCreation(oktaClient, metadataURL, policyCfg, created); err != nil {
			return nil, err
		}
	}

	return created, nil
}

// postProcessClientUnregister handles Okta cleanup after client deprovision
func (i *okta) postProcessClientUnregister(clientID string, agentDetails map[string]string, credentialObj interface{}, apiClient coreapi.Client) error {
	var metadataURL string
	var apiToken string
	var policyJSON string
	if cfg, ok := credentialObj.(corecfg.IDPConfig); ok {
		metadataURL = cfg.GetMetadataURL()
		if authCfg := cfg.GetAuthConfig(); authCfg != nil {
			apiToken = authCfg.GetAccessToken()
		}
		policyJSON = cfg.GetOktaPolicy()
	}
	baseURL, err := oktaBaseURLFromMetadataURL(metadataURL)
	if err != nil {
		return err
	}
	if baseURL == "" || apiToken == "" {
		return nil // skip if not configured
	}
	oktaClient := clients.New(apiClient, baseURL, apiToken)

	policyCfg, err := parseOktaPolicyConfig(policyJSON)
	if err != nil {
		return err
	}

	// Remove policy/rule
	if err := i.handlePolicyRuleDeletion(oktaClient, metadataURL, policyCfg, agentDetails); err != nil {
		return err
	}

	// Unassign group
	if err := i.handleGroupUnassignment(oktaClient, clientID, agentDetails); err != nil {
		return err
	}

	return nil
}

// handlePolicyRuleDeletion deletes rule then policy if present in agentDetails
func (i *okta) handlePolicyRuleDeletion(oktaClient *clients.Okta, metadataURL string, policyCfg *oktaPolicyConfig, agentDetails map[string]string) error {
	authServerID := ""
	if policyCfg != nil {
		authServerID = policyCfg.AuthServerID
	}
	if authServerID == "" {
		authServerID = oktaAuthServerIDFromMetadataURL(metadataURL)
	}
	if policyID, ok := agentDetails["oktaPolicyId"]; ok && policyID != "" {
		if authServerID == "" {
			return fmt.Errorf("authServerId is required to delete okta policy/rule (could not infer from metadataURL)")
		}
		if ruleID, ok := agentDetails["oktaRuleId"]; ok && ruleID != "" {
			if err := oktaClient.DeleteRule(authServerID, policyID, ruleID); err != nil {
				return err
			}
		}
		if err := oktaClient.DeletePolicy(authServerID, policyID); err != nil {
			return err
		}
	}
	return nil
}

// handleGroupUnassignment unassigns a group from the app if group id present
func (i *okta) handleGroupUnassignment(oktaClient *clients.Okta, clientID string, agentDetails map[string]string) error {
	if groupId, ok := agentDetails["oktaGroupId"]; ok && groupId != "" {
		if err := oktaClient.UnassignGroupFromApp(clientID, groupId); err != nil {
			return err
		}
	}
	return nil
}
func (i *okta) getAuthorizationHeaderPrefix() string {
	return clients.OktaAuthHeaderPrefix
}

// handleGroupAssignment provisions/assigns a group and records created id into created map
func (i *okta) handleGroupAssignment(oktaClient *clients.Okta, clientRes ClientMetadata, groupName string, created map[string]string) error {
	groupId, err := oktaClient.FindGroupByName(groupName)
	if err != nil {
		return err
	}
	if groupId == "" {
		groupId, err = oktaClient.CreateGroup(groupName, "Auto-created group")
		if err != nil {
			return err
		}
	}
	if err := oktaClient.AssignGroupToApp(clientRes.GetClientID(), groupId); err != nil {
		return err
	}
	created["oktaGroupId"] = groupId
	return nil
}

// handlePolicyRuleCreation creates policy and rule if requested and records ids
func (i *okta) handlePolicyRuleCreation(oktaClient *clients.Okta, metadataURL string, policyCfg *oktaPolicyConfig, created map[string]string) error {
	if policyCfg == nil || !policyCfg.isEnabled() {
		return nil
	}
	authServerID := policyCfg.AuthServerID
	if authServerID == "" {
		authServerID = oktaAuthServerIDFromMetadataURL(metadataURL)
	}
	if authServerID == "" {
		return fmt.Errorf("authServerId is required for okta policy creation (could not infer from metadataURL)")
	}
	if len(policyCfg.PolicyTemplate) == 0 {
		return fmt.Errorf("policyTemplate is required for okta policy creation")
	}

	policyID, err := oktaClient.CreatePolicy(authServerID, policyCfg.PolicyTemplate)
	if err != nil {
		return err
	}
	created["oktaPolicyId"] = policyID
	if rule, ok := policyCfg.PolicyTemplate["rule"].(map[string]interface{}); ok {
		ruleID, err := oktaClient.CreateRule(authServerID, policyID, rule)
		if err != nil {
			return err
		}
		created["oktaRuleId"] = ruleID
	}
	return nil
}

// validateExtraProperties validates Okta-specific extra properties and sets defaults
func (i *okta) validateExtraProperties(extraProps map[string]interface{}) error {
	pkceRequired, _ := extraProps[oktaPKCERequired].(bool)
	appType, _ := extraProps[oktaApplicationType].(string)

	// Validate that if PKCE is required and application_type is explicitly set, it must be 'browser'
	if pkceRequired && appType != "" && appType != oktaAppTypeBrowser {
		return fmt.Errorf("when %s is true, %s must be '%s' or unset, got '%s'",
			oktaPKCERequired, oktaApplicationType, oktaAppTypeBrowser, appType)
	}

	// Set default application_type only if not explicitly set by user
	// Check appType == "" to preserve valid explicit user choices (e.g., "service")
	// Default to 'web', override to 'browser' if PKCE is required
	// Note: May be further overridden to 'service' in preProcessClientRequest for client credentials flows
	if appType == "" {
		extraProps[oktaApplicationType] = oktaAppTypeWeb
		if pkceRequired {
			extraProps[oktaApplicationType] = oktaAppTypeBrowser
		}
	}

	return nil
}

func (i *okta) preProcessClientRequest(clientRequest *clientMetadata) {
	if clientRequest.extraProperties == nil {
		clientRequest.extraProperties = make(map[string]interface{})
	}

	pkceRequired, _ := clientRequest.extraProperties[oktaPKCERequired].(bool)

	// Override application_type to 'service' for client credentials flows
	// (validateExtraProperties sets default to 'web' or 'browser')
	if slices.Contains(clientRequest.GrantTypes, GrantTypeClientCredentials) {
		clientRequest.extraProperties[oktaApplicationType] = oktaAppTypeService
		if len(clientRequest.ResponseTypes) == 0 {
			clientRequest.ResponseTypes = []string{AuthResponseToken}
		}
	}

	if pkceRequired {
		clientRequest.TokenEndpointAuthMethod = "none"
	}
}
