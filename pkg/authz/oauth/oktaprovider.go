package oauth

const (
	oktaApplicationType  = "application_type"
	oktaAppTypeService   = "service"
	oktaAppTypeWeb       = "web"
	oktaAppTypeBrowser   = "browser"
	oktaPKCERequired     = "pkce_required"
	oktaAuthHeaderPrefix = "SSWS"
	oktaSpa              = "okta-spa"
)

type okta struct {
}

func (i *okta) getAuthorizationHeaderPrefix() string {
	return oktaAuthHeaderPrefix
}

func (i *okta) preProcessClientRequest(clientRequest *clientMetadata) {
	if clientRequest.ExtraProperties == nil {
		clientRequest.ExtraProperties = make(map[string]interface{})
	}

	appType, ok := clientRequest.ExtraProperties[oktaApplicationType].(string)
	if !ok {
		appType = oktaAppTypeService
	}

	// Check if PKCE is required - if so, this should be a browser (public client) app
	pkceRequired, _ := clientRequest.ExtraProperties[oktaPKCERequired].(bool)

	for _, grantType := range clientRequest.GrantTypes {
		switch grantType {
		case GrantTypeClientCredentials:
			if len(clientRequest.ResponseTypes) == 0 {
				clientRequest.ResponseTypes = []string{AuthResponseToken}
			}
		default:
			if pkceRequired {
				appType = oktaAppTypeBrowser
				clientRequest.TokenEndpointAuthMethod = "none"
			} else if appType == oktaAppTypeService {
				appType = oktaAppTypeWeb
			}
		}
	}
	clientRequest.ExtraProperties[oktaApplicationType] = appType
}
