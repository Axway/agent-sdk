package oauth

import (
	"fmt"
	"net/url"
	"slices"

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

// postProcessClientRegistration handles Okta provisioning after client registration
func (i *okta) postProcessClientRegistration(clientRes ClientMetadata, extraProps map[string]interface{}, credentialObj interface{}, apiClient coreapi.Client) (map[string]string, error) {
	var metadataURL string
	var apiToken string
	if cfg, ok := credentialObj.(corecfg.IDPConfig); ok {
		metadataURL = cfg.GetMetadataURL()
		if authCfg := cfg.GetAuthConfig(); authCfg != nil {
			apiToken = authCfg.GetAccessToken()
		}
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

	if err := i.handleGroupAssignment(oktaClient, clientRes, extraProps, created); err != nil {
		return nil, err
	}

	if err := i.handlePolicyRuleCreation(oktaClient, extraProps, created); err != nil {
		return nil, err
	}

	return created, nil
}

// postProcessClientUnregister handles Okta cleanup after client deprovision
func (i *okta) postProcessClientUnregister(clientID string, agentDetails map[string]string, extraProps map[string]interface{}, credentialObj interface{}, apiClient coreapi.Client) error {
	var metadataURL string
	var apiToken string
	if cfg, ok := credentialObj.(corecfg.IDPConfig); ok {
		metadataURL = cfg.GetMetadataURL()
		if authCfg := cfg.GetAuthConfig(); authCfg != nil {
			apiToken = authCfg.GetAccessToken()
		}
	}
	baseURL, err := oktaBaseURLFromMetadataURL(metadataURL)
	if err != nil {
		return err
	}
	if baseURL == "" || apiToken == "" {
		return nil // skip if not configured
	}
	oktaClient := clients.New(apiClient, baseURL, apiToken)

	// Remove policy/rule
	if err := i.handlePolicyRuleDeletion(oktaClient, extraProps, agentDetails); err != nil {
		return err
	}

	// Unassign group
	if err := i.handleGroupUnassignment(oktaClient, clientID, agentDetails); err != nil {
		return err
	}

	return nil
}

// handlePolicyRuleDeletion deletes rule then policy if present in agentDetails
func (i *okta) handlePolicyRuleDeletion(oktaClient *clients.Okta, extraProps map[string]interface{}, agentDetails map[string]string) error {
	authServerID, _ := extraProps["authServerId"].(string)
	if policyID, ok := agentDetails["oktaPolicyId"]; ok && policyID != "" {
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
func (i *okta) handleGroupAssignment(oktaClient *clients.Okta, clientRes ClientMetadata, extraProps map[string]interface{}, created map[string]string) error {
	// Accept several possible keys for group name to be forgiving in config
	groupName, _ := extraProps["group"].(string)
	if groupName == "" {
		groupName, _ = extraProps["Group"].(string)
	}
	if groupName == "" {
		groupName, _ = extraProps["GroupName"].(string)
	}
	if groupName == "" {
		groupName, _ = extraProps["groupName"].(string)
	}
	if groupName == "" {
		return nil
	}

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
func (i *okta) handlePolicyRuleCreation(oktaClient *clients.Okta, extraProps map[string]interface{}, created map[string]string) error {
	createPolicy, _ := extraProps["createPolicy"].(bool)
	if createPolicy {
		authServerID, _ := extraProps["authServerId"].(string)
		policyTemplate, _ := extraProps["policyTemplate"].(map[string]interface{})
		if authServerID == "" {
			return fmt.Errorf("authServerId is required when createPolicy is true")
		}
		if len(policyTemplate) == 0 {
			return fmt.Errorf("policyTemplate is required when createPolicy is true")
		}

		policyID, err := oktaClient.CreatePolicy(authServerID, policyTemplate)
		if err != nil {
			return err
		}
		created["oktaPolicyId"] = policyID
		if rule, ok := policyTemplate["rule"].(map[string]interface{}); ok {
			ruleID, err := oktaClient.CreateRule(authServerID, policyID, rule)
			if err != nil {
				return err
			}
			created["oktaRuleId"] = ruleID
		}
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

	createPolicy, _ := extraProps["createPolicy"].(bool)
	if createPolicy {
		authServerID, _ := extraProps["authServerId"].(string)
		policyTemplate, _ := extraProps["policyTemplate"].(map[string]interface{})
		if authServerID == "" {
			return fmt.Errorf("authServerId is required when createPolicy is true")
		}
		if len(policyTemplate) == 0 {
			return fmt.Errorf("policyTemplate is required when createPolicy is true")
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
