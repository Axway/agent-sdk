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
func (i *okta) postProcessClientRegistration(clientRes ClientMetadata, extraProps map[string]interface{}, credentialObj interface{}) error {
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

	// Group assignment
	groupName, _ := extraProps["group"].(string)
	if groupName != "" {
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
		err = oktaClient.AssignGroupToApp(ctx, clientRes.GetClientID(), groupId)
		if err != nil {
			return err
		}
	}

	// Policy/rule creation
	if createPolicy, ok := extraProps["createPolicy"].(bool); ok && createPolicy {
		authServerId, _ := extraProps["authServerId"].(string)
		policyTemplate, _ := extraProps["policyTemplate"].(map[string]interface{})
		policyId, err := oktaClient.CreatePolicy(ctx, authServerId, policyTemplate)
		if err != nil {
			return err
		}
		if rule, ok := policyTemplate["rule"].(map[string]interface{}); ok {
			_, err = oktaClient.CreateRule(ctx, authServerId, policyId, rule)
			if err != nil {
				return err
			}
		}
	}

	// Scope creation
	if createScopes, ok := extraProps["createScopes"].(bool); ok && createScopes {
		authServerId, _ := extraProps["authServerId"].(string)
		scopes, _ := extraProps["scopes"].([]interface{})
		for _, s := range scopes {
			scopeMap, ok := s.(map[string]interface{})
			if !ok {
				continue
			}
			_, err := oktaClient.CreateScope(ctx, authServerId, scopeMap)
			if err != nil {
				return err
			}
		}
	}
	return nil
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
	authServerId, _ := extraProps["authServerId"].(string)
	if policyId, ok := agentDetails["oktaPolicyId"]; ok && policyId != "" {
		if ruleId, ok := agentDetails["oktaRuleId"]; ok && ruleId != "" {
			err := oktaClient.DeleteRule(ctx, authServerId, policyId, ruleId)
			if err != nil {
				return err
			}
		}
		err := oktaClient.DeletePolicy(ctx, authServerId, policyId)
		if err != nil {
			return err
		}
	}

	// Unassign group
	if groupId, ok := agentDetails["oktaGroupId"]; ok && groupId != "" {
		err := oktaClient.UnassignGroupFromApp(ctx, clientID, groupId)
		if err != nil {
			return err
		}
	}

	// Remove scopes
	if scopeId, ok := agentDetails["oktaScopeId"]; ok && scopeId != "" {
		err := oktaClient.DeleteScope(ctx, authServerId, scopeId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *okta) getAuthorizationHeaderPrefix() string {
	return oktaAuthHeaderPrefix
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
