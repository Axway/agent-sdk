package oauth

import (
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

func validateOktaConfiguredResources(idp corecfg.IDPConfig, apiClient coreapi.Client) error {
	groupName, policyName := oktaValidationNames(idp)
	if groupName == "" && policyName == "" {
		return nil
	}

	apiToken := oktaManagementAPIToken(idp)
	if apiToken == "" {
		return fmt.Errorf("okta group/policy validation requires a management API access token. Set %q in the IDP auth configuration", "auth.accessToken")
	}

	metadataURL := idp.GetMetadataURL()
	oktaClient, err := oktaClientFromMetadataURL(apiClient, metadataURL, apiToken)
	if err != nil {
		return err
	}

	authServerID := oktaAuthServerIDFromMetadataURL(metadataURL)

	if err := validateOktaGroupExists(oktaClient, groupName); err != nil {
		return err
	}
	if err := validateOktaPolicyExists(oktaClient, authServerID, policyName); err != nil {
		return err
	}
	return nil
}

func oktaValidationNames(idp corecfg.IDPConfig) (groupName, policyName string) {
	return strings.TrimSpace(idp.GetOktaGroup()), strings.TrimSpace(idp.GetOktaPolicy())
}

func oktaManagementAPIToken(idp corecfg.IDPConfig) string {
	authCfg := idp.GetAuthConfig()
	if authCfg == nil {
		return ""
	}
	return strings.TrimSpace(authCfg.GetAccessToken())
}

func oktaClientFromMetadataURL(apiClient coreapi.Client, metadataURL, apiToken string) (*clients.Okta, error) {
	baseURL, err := oktaBaseURLFromMetadataURL(metadataURL)
	if err != nil {
		return nil, err
	}
	if baseURL == "" {
		return nil, fmt.Errorf("invalid okta metadata URL: %q", metadataURL)
	}
	return clients.New(apiClient, baseURL, apiToken), nil
}

func validateOktaGroupExists(oktaClient *clients.Okta, groupName string) error {
	if strings.TrimSpace(groupName) == "" {
		return nil
	}
	groupID, err := oktaClient.FindGroupByName(groupName)
	if err != nil {
		return err
	}
	if groupID == "" {
		return fmt.Errorf("configured okta group %q was not found", groupName)
	}
	return nil
}

func validateOktaPolicyExists(oktaClient *clients.Okta, authServerID, policyName string) error {
	if strings.TrimSpace(policyName) == "" {
		return nil
	}
	if authServerID == "" {
		return fmt.Errorf("authServerId could not be inferred from the metadata URL. Ensure the metadata URL follows the Okta custom authorization server pattern (/oauth2/{authServerId}/...)")
	}
	policy, err := oktaClient.FindPolicyByName(authServerID, policyName)
	if err != nil {
		return err
	}
	if policy == nil {
		return fmt.Errorf("configured okta policy %q was not found on authorization server %q", policyName, authServerID)
	}
	return nil
}

// postProcessClientRegistration handles Okta provisioning after client registration
func (i *okta) postProcessClientRegistration(clientRes ClientMetadata, idp corecfg.IDPConfig, apiClient coreapi.Client) (map[string]string, error) {
	metadataURL := idp.GetMetadataURL()
	var apiToken string
	if authCfg := idp.GetAuthConfig(); authCfg != nil {
		apiToken = authCfg.GetAccessToken()
	}
	groupName := idp.GetOktaGroup()
	policyName := idp.GetOktaPolicy()

	baseURL, err := oktaBaseURLFromMetadataURL(metadataURL)
	if err != nil {
		return nil, err
	}
	if baseURL == "" || apiToken == "" {
		return nil, nil // skip if not configured
	}
	oktaClient := clients.New(apiClient, baseURL, apiToken)
	updated := make(map[string]string)

	if err := i.handleGroupAssignment(oktaClient, clientRes, groupName, updated); err != nil {
		return nil, err
	}

	authServerID := oktaAuthServerIDFromMetadataURL(metadataURL)
	if err := i.handlePolicyAssignment(oktaClient, clientRes.GetClientID(), authServerID, policyName, updated); err != nil {
		return nil, err
	}

	return updated, nil
}

// postProcessClientUnregister removes only the app→group assignment (no deletes).
func (i *okta) postProcessClientUnregister(clientID string, agentDetails map[string]string, idp corecfg.IDPConfig, apiClient coreapi.Client) error {
	metadataURL := idp.GetMetadataURL()
	var apiToken string
	if authCfg := idp.GetAuthConfig(); authCfg != nil {
		apiToken = authCfg.GetAccessToken()
	}

	baseURL, err := oktaBaseURLFromMetadataURL(metadataURL)
	if err != nil {
		return err
	}
	if baseURL == "" || apiToken == "" {
		return nil // skip if not configured
	}

	oktaClient := clients.New(apiClient, baseURL, apiToken)
	return i.handleGroupUnassignment(oktaClient, clientID, agentDetails)
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

// handleGroupAssignment looks up the named group and assigns it to the app.
// Returns an error if the group is not found.  Even though we fail fast at the start, we check again to guard against the possibility that the group was deleted between validation and assignment.
func (i *okta) handleGroupAssignment(oktaClient *clients.Okta, clientRes ClientMetadata, groupName string, updated map[string]string) error {
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		return nil
	}
	groupId, err := oktaClient.FindGroupByName(groupName)
	if err != nil {
		return err
	}
	if groupId == "" {
		return fmt.Errorf("configured okta group %q was not found", groupName)
	}
	if err := oktaClient.AssignGroupToApp(clientRes.GetClientID(), groupId); err != nil {
		return err
	}
	updated["oktaGroupId"] = groupId
	return nil
}

// handlePolicyAssignment looks up the named policy on the authorization server and assigns the client to it.
// Returns an error if the policy is not found. Even though we fail fast at the start, we check again to guard against the possibility that the policy was deleted between validation and assignment.
func (i *okta) handlePolicyAssignment(oktaClient *clients.Okta, clientID, authServerID string, policyName string, updated map[string]string) error {
	policyName = strings.TrimSpace(policyName)
	if policyName == "" {
		return nil
	}
	if authServerID == "" {
		return fmt.Errorf("authServerId could not be inferred from the metadata URL; ensure the metadata URL follows the Okta custom authorization server pattern (/oauth2/{authServerId}/)")
	}
	policy, err := oktaClient.FindPolicyByName(authServerID, policyName)
	if err != nil {
		return err
	}
	if policy == nil {
		return fmt.Errorf("configured okta policy %q was not found on authorization server %q", policyName, authServerID)
	}
	policyID, _ := policy["id"].(string)
	policyID = strings.TrimSpace(policyID)
	if policyID == "" {
		return fmt.Errorf("okta policy %q has no id field", policyName)
	}
	// Update policy-level client assignment (no rules).
	if err := oktaClient.AssignClientToPolicy(authServerID, policy, clientID); err != nil {
		return err
	}
	updated["oktaPolicyId"] = policyID
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
