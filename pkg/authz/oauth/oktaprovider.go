package oauth

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/authz/oauth/clients"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	oktaApplicationType = "application_type"
	oktaAppTypeService  = "service"
	oktaAppTypeWeb      = "web"
	oktaAppTypeBrowser  = "browser"
	oktaPKCERequired    = "pkce_required"
	oktaSpa             = "okta-spa"
)

type okta struct {
	logger log.FieldLogger
}

func oktaBaseURLFromMetadataURL(metadataURL string) (string, error) {
	if metadataURL == "" {
		return "", fmt.Errorf("metadata URL is empty")
	}
	u, err := url.Parse(metadataURL)
	if err != nil {
		return "", fmt.Errorf("invalid metadata URL %q: %w", metadataURL, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid metadata URL %q: missing scheme or host", metadataURL)
	}
	return u.Scheme + "://" + u.Host, nil
}

func oktaAuthServerIDFromMetadataURL(metadataURL string) (string, error) {
	if metadataURL == "" {
		return "", fmt.Errorf("metadata URL is empty")
	}
	u, err := url.Parse(metadataURL)
	if err != nil {
		return "", fmt.Errorf("invalid metadata URL %q: %w", metadataURL, err)
	}
	path := strings.Trim(u.Path, "/")
	if path == "" {
		return "", fmt.Errorf("unable to determine Okta authorization server from metadata URL")
	}
	segments := strings.Split(path, "/")
	for i := 0; i < len(segments)-1; i++ {
		if segments[i] == "oauth2" {
			return segments[i+1], nil
		}
	}
	return "", fmt.Errorf("unable to determine Okta authorization server from metadata URL")
}

func normalizeGrantType(grantType string) string {
	switch grantType {
	case GrantTypeClientCredentials:
		return "clientcredentials"
	case GrantTypeAuthorizationCode:
		return "authorizationcode"
	case GrantTypeImplicit:
		return "implicitgrant"
	default:
		return strings.ReplaceAll(grantType, "_", "")
	}
}

func policyNameTemplate(idp corecfg.IDPConfig) string {
	cfg, ok := idp.(interface{ GetPolicyNameTemplate() string })
	if !ok {
		return corecfg.OktaPlaceholderScope + "-" + corecfg.OktaPlaceholderOAuthFlow
	}
	if t := cfg.GetPolicyNameTemplate(); t != "" {
		return t
	}
	return corecfg.OktaPlaceholderScope + "-" + corecfg.OktaPlaceholderOAuthFlow
}

func (i *okta) postProcessClientRegistration(clientRes ClientMetadata, idp corecfg.IDPConfig, apiClient coreapi.Client) error {
	i.logger.WithField("clientID", clientRes.GetClientID()).Debug("running Okta post-registration hook")

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
		return nil
	}

	authServerID, err := oktaAuthServerIDFromMetadataURL(metadataURL)
	if err != nil {
		return err
	}

	oktaClient := clients.New(apiClient, baseURL, apiToken)

	grantType := ""
	if gt := clientRes.GetGrantTypes(); len(gt) > 0 {
		grantType = gt[0]
	}

	if err := i.handlePerScopePolicyAssign(oktaClient, clientRes, idp, authServerID, grantType); err != nil {
		return err
	}

	if err := i.handleGroupAssignment(oktaClient, clientRes.GetClientID(), idp); err != nil {
		return err
	}

	i.logger.WithField("clientID", clientRes.GetClientID()).Info("completed Okta post-registration hook")
	return nil
}

func (i *okta) postProcessClientUnregister(clientID string, idp corecfg.IDPConfig, apiClient coreapi.Client, scopes []string, grantType string) error {
	i.logger.WithField("clientID", clientID).Debug("running Okta post-unregister hook")

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
		return nil
	}

	authServerID, err := oktaAuthServerIDFromMetadataURL(metadataURL)
	if err != nil {
		return err
	}

	oktaClient := clients.New(apiClient, baseURL, apiToken)

	if err := oktaClient.DeactivateApp(clientID); err != nil {
		return err
	}
	if err := oktaClient.DeleteApp(clientID); err != nil {
		return err
	}

	if err := i.handleGroupRemoval(oktaClient, clientID, idp); err != nil {
		return err
	}

	if err := i.handlePerScopePolicyUnassign(oktaClient, clientID, scopes, idp, authServerID, grantType); err != nil {
		return err
	}

	i.logger.WithField("clientID", clientID).Info("completed Okta post-unregister hook")
	return nil
}

func (i *okta) handlePerScopePolicyAssign(
	oktaClient *clients.Okta,
	clientRes ClientMetadata,
	idp corecfg.IDPConfig,
	authServerID string,
	grantType string,
) error {
	i.logger.
		WithField("clientID", clientRes.GetClientID()).
		WithField("authServerID", authServerID).
		Trace("handling per-scope policy assignment")

	normalizedFlow := normalizeGrantType(grantType)
	template := policyNameTemplate(idp)
	scopes := clientRes.GetScopes()

	replacer := strings.NewReplacer(corecfg.OktaPlaceholderOAuthFlow, normalizedFlow)
	for _, scope := range scopes {
		if strings.TrimSpace(scope) == "" {
			continue
		}
		policyName := replacer.Replace(strings.ReplaceAll(template, corecfg.OktaPlaceholderScope, scope))
		if err := i.assignScopePolicy(oktaClient, clientRes, idp, authServerID, grantType, policyName, scope); err != nil {
			return err
		}
	}

	return nil
}

func (i *okta) assignScopePolicy(
	oktaClient *clients.Okta,
	clientRes ClientMetadata,
	idp corecfg.IDPConfig,
	authServerID, grantType, policyName, scope string,
) error {
	if len(policyName) > 100 {
		return fmt.Errorf("Okta policy name exceeds 100-character limit: %s (%d chars)", policyName, len(policyName))
	}

	i.logger.WithField("policyName", policyName).Trace("looking up policy")

	policy, err := oktaClient.FindPolicyByName(authServerID, policyName)
	if err != nil {
		return err
	}

	if policy == nil {
		i.logger.WithField("policyName", policyName).Trace("policy not found, creating")
		createdPolicy, err := oktaClient.CreatePolicy(authServerID, policyName, clientRes.GetClientID())
		if err != nil {
			return err
		}
		policyID, _ := createdPolicy["id"].(string)
		if err := oktaClient.CreatePolicyRule(authServerID, policyID, policyName, grantType, scope); err != nil {
			return err
		}
		i.logger.WithField("policyName", policyName).Trace("created policy and rule")
		return nil
	}

	i.logger.WithField("policyName", policyName).Trace("policy found, assigning client")
	return oktaClient.AssignClientToPolicy(authServerID, policy, clientRes.GetClientID())
}

func (i *okta) handlePerScopePolicyUnassign(
	oktaClient *clients.Okta,
	clientID string,
	scopes []string,
	idp corecfg.IDPConfig,
	authServerID string,
	grantType string,
) error {
	i.logger.
		WithField("clientID", clientID).
		WithField("authServerID", authServerID).
		Trace("handling per-scope policy unassignment")

	normalizedFlow := normalizeGrantType(grantType)
	template := policyNameTemplate(idp)

	for _, scope := range scopes {
		policyName := strings.NewReplacer(corecfg.OktaPlaceholderScope, scope, corecfg.OktaPlaceholderOAuthFlow, normalizedFlow).Replace(template)
		i.logger.WithField("policyName", policyName).Trace("looking up policy for removal")

		policy, err := oktaClient.FindPolicyByName(authServerID, policyName)
		if err != nil {
			return err
		}

		if policy == nil {
			i.logger.
				WithField("scope", scope).
				WithField("policyName", policyName).
				Trace("policy not found for scope during unassign, skipping")
			continue
		}

		i.logger.
			WithField("clientID", clientID).
			WithField("policyName", policyName).
			Trace("removing client from policy")
		if err := oktaClient.RemoveClientFromPolicy(authServerID, policy, clientID); err != nil {
			return err
		}
	}

	return nil
}

func (i *okta) validateExtraProperties(extraProps map[string]interface{}) error {
	pkceRequired, _ := extraProps[oktaPKCERequired].(bool)
	appType, _ := extraProps[oktaApplicationType].(string)

	if pkceRequired && appType != "" && appType != oktaAppTypeBrowser {
		return fmt.Errorf("when %s is true, %s must be '%s' or unset, got '%s'",
			oktaPKCERequired, oktaApplicationType, oktaAppTypeBrowser, appType)
	}

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

func (i *okta) getAuthorizationHeaderPrefix() string {
	return clients.OktaAuthHeaderPrefix
}

func (i *okta) handleGroupAssignment(oktaClient *clients.Okta, clientID string, idp corecfg.IDPConfig) error {
	groupName := oktaGroupName(idp)
	if groupName == "" {
		return nil
	}
	i.logger.WithField("clientID", clientID).WithField("groupName", groupName).Trace("adding app to Okta group")
	groupID, err := oktaClient.FindGroupByName(groupName)
	if err != nil {
		return fmt.Errorf("error finding Okta group %q: %w", groupName, err)
	}
	if groupID == "" {
		return fmt.Errorf("configured Okta group %q not found", groupName)
	}
	return oktaClient.AssignGroupToApp(clientID, groupID)
}

func (i *okta) handleGroupRemoval(oktaClient *clients.Okta, clientID string, idp corecfg.IDPConfig) error {
	groupName := oktaGroupName(idp)
	if groupName == "" {
		return nil
	}
	i.logger.WithField("clientID", clientID).WithField("groupName", groupName).Trace("removing app from Okta group")
	groupID, err := oktaClient.FindGroupByName(groupName)
	if err != nil {
		return fmt.Errorf("error finding Okta group %q: %w", groupName, err)
	}
	if groupID == "" {
		return nil
	}
	return oktaClient.UnassignGroupFromApp(clientID, groupID)
}

func oktaGroupName(idp corecfg.IDPConfig) string {
	cfg, ok := idp.(interface{ GetOktaGroup() string })
	if !ok {
		return ""
	}
	return cfg.GetOktaGroup()
}

func validateOktaGroupExists(idp corecfg.IDPConfig, apiClient coreapi.Client) error {
	groupName := oktaGroupName(idp)
	if groupName == "" {
		return nil
	}
	var apiToken string
	if authCfg := idp.GetAuthConfig(); authCfg != nil {
		apiToken = authCfg.GetAccessToken()
	}
	if apiToken == "" {
		return nil
	}
	baseURL, err := oktaBaseURLFromMetadataURL(idp.GetMetadataURL())
	if err != nil {
		return err
	}
	oktaClient := clients.New(apiClient, baseURL, apiToken)
	groupID, err := oktaClient.FindGroupByName(groupName)
	if err != nil {
		return fmt.Errorf("error looking up Okta group %q: %w", groupName, err)
	}
	if groupID == "" {
		return fmt.Errorf("configured Okta group %q not found", groupName)
	}
	return nil
}
