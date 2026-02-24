package oauth

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/Axway/agent-sdk/pkg/authz/oauth/oktaapi"
)

const (
	oktaApplicationType  = "application_type"
	oktaAppTypeService   = "service"
	oktaAppTypeWeb       = "web"
	oktaAppTypeBrowser   = "browser"
	oktaPKCERequired     = "pkce_required"
	oktaAuthHeaderPrefix = "SSWS"
	oktaSpa              = "okta-spa"
)

type okta struct{}

// postProcessClientRegistration handles Okta provisioning after client registration
func (i *okta) postProcessClientRegistration(clientRes ClientMetadata, extraProps map[string]interface{}, credentialObj interface{}) (map[string]string, error) {
	ctx := context.Background()
	baseURL := os.Getenv("OKTA_BASE_URL")
	apiToken := os.Getenv("OKTA_API_TOKEN")
	if v, ok := extraProps["oktaBaseURL"].(string); ok && v != "" {
		baseURL = v
	}
	if v, ok := extraProps["oktaApiToken"].(string); ok && v != "" {
		apiToken = v
	}
	if baseURL == "" || apiToken == "" {
		return nil, nil // skip if not configured
	}
	oktaClient := oktaapi.New(baseURL, apiToken)
	created := make(map[string]string)

	if err := i.handleGroupAssignment(ctx, oktaClient, clientRes, extraProps, created); err != nil {
		return nil, err
	}

	if err := i.handlePolicyRuleCreation(ctx, oktaClient, extraProps, created); err != nil {
		return nil, err
	}

	if err := i.handleScopeCreation(ctx, oktaClient, extraProps, created); err != nil {
		return nil, err
	}

	return created, nil
}

// postProcessClientUnregister handles Okta cleanup after client deprovision
func (i *okta) postProcessClientUnregister(clientID string, agentDetails map[string]string, extraProps map[string]interface{}, credentialObj interface{}) error {
	ctx := context.Background()
	baseURL := os.Getenv("OKTA_BASE_URL")
	apiToken := os.Getenv("OKTA_API_TOKEN")
	if v, ok := extraProps["oktaBaseURL"].(string); ok && v != "" {
		baseURL = v
	}
	if v, ok := extraProps["oktaApiToken"].(string); ok && v != "" {
		apiToken = v
	}
	if baseURL == "" || apiToken == "" {
		return nil // skip if not configured
	}
	oktaClient := oktaapi.New(baseURL, apiToken)

	// Remove policy/rule
	if err := i.handlePolicyRuleDeletion(ctx, oktaClient, extraProps, agentDetails); err != nil {
		return err
	}

	// Unassign group
	if err := i.handleGroupUnassignment(ctx, oktaClient, clientID, agentDetails); err != nil {
		return err
	}

	// Remove scopes
	if err := i.handleScopeDeletion(ctx, oktaClient, extraProps, agentDetails); err != nil {
		return err
	}

	return nil
}

// handlePolicyRuleDeletion deletes rule then policy if present in agentDetails
func (i *okta) handlePolicyRuleDeletion(ctx context.Context, oktaClient *oktaapi.OktaAPI, extraProps map[string]interface{}, agentDetails map[string]string) error {
	authServerId, _ := extraProps["authServerId"].(string)
	if policyId, ok := agentDetails["oktaPolicyId"]; ok && policyId != "" {
		if ruleId, ok := agentDetails["oktaRuleId"]; ok && ruleId != "" {
			if err := oktaClient.DeleteRule(ctx, authServerId, policyId, ruleId); err != nil {
				return err
			}
		}
		if err := oktaClient.DeletePolicy(ctx, authServerId, policyId); err != nil {
			return err
		}
	}
	return nil
}

// handleGroupUnassignment unassigns a group from the app if group id present
func (i *okta) handleGroupUnassignment(ctx context.Context, oktaClient *oktaapi.OktaAPI, clientID string, agentDetails map[string]string) error {
	if groupId, ok := agentDetails["oktaGroupId"]; ok && groupId != "" {
		if err := oktaClient.UnassignGroupFromApp(ctx, clientID, groupId); err != nil {
			return err
		}
	}
	return nil
}

// handleScopeDeletion deletes the scope if scope id present in agentDetails
func (i *okta) handleScopeDeletion(ctx context.Context, oktaClient *oktaapi.OktaAPI, extraProps map[string]interface{}, agentDetails map[string]string) error {
	authServerId, _ := extraProps["authServerId"].(string)
	if scopeId, ok := agentDetails["oktaScopeId"]; ok && scopeId != "" {
		if err := oktaClient.DeleteScope(ctx, authServerId, scopeId); err != nil {
			return err
		}
	}
	return nil
}

func (i *okta) getAuthorizationHeaderPrefix() string {
	return oktaAuthHeaderPrefix
}

// handleGroupAssignment provisions/assigns a group and records created id into created map
func (i *okta) handleGroupAssignment(ctx context.Context, oktaClient *oktaapi.OktaAPI, clientRes ClientMetadata, extraProps map[string]interface{}, created map[string]string) error {
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

	groupId, err := oktaClient.FindGroupByName(ctx, groupName)
	if err != nil {
		return err
	}
	if groupId == "" {
		groupId, err = oktaClient.CreateGroup(ctx, groupName, "Auto-created group")
		if err != nil {
			return err
		}
	}
	if err := oktaClient.AssignGroupToApp(ctx, clientRes.GetClientID(), groupId); err != nil {
		return err
	}
	created["oktaGroupId"] = groupId
	return nil
}

// handlePolicyRuleCreation creates policy and rule if requested and records ids
func (i *okta) handlePolicyRuleCreation(ctx context.Context, oktaClient *oktaapi.OktaAPI, extraProps map[string]interface{}, created map[string]string) error {
	// Default to creating policy/rule for Okta unless explicitly disabled
	createPolicy, ok := extraProps["createPolicy"].(bool)
	if !ok {
		createPolicy = true
	}
	if createPolicy {
		authServerId, _ := extraProps["authServerId"].(string)
		policyTemplate, _ := extraProps["policyTemplate"].(map[string]interface{})
		policyId, err := oktaClient.CreatePolicy(ctx, authServerId, policyTemplate)
		if err != nil {
			return err
		}
		created["oktaPolicyId"] = policyId
		if rule, ok := policyTemplate["rule"].(map[string]interface{}); ok {
			ruleId, err := oktaClient.CreateRule(ctx, authServerId, policyId, rule)
			if err != nil {
				return err
			}
			created["oktaRuleId"] = ruleId
		}
	}
	return nil
}

// handleScopeCreation creates scopes and records the first created id
func (i *okta) handleScopeCreation(ctx context.Context, oktaClient *oktaapi.OktaAPI, extraProps map[string]interface{}, created map[string]string) error {
	if createScopes, ok := extraProps["createScopes"].(bool); ok && createScopes {
		authServerId, _ := extraProps["authServerId"].(string)
		scopes, _ := extraProps["scopes"].([]interface{})
		for _, s := range scopes {
			scopeMap, ok := s.(map[string]interface{})
			if !ok {
				continue
			}
			scopeId, err := oktaClient.CreateScope(ctx, authServerId, scopeMap)
			if err != nil {
				return err
			}
			if _, exists := created["oktaScopeId"]; !exists {
				created["oktaScopeId"] = scopeId
			}
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
