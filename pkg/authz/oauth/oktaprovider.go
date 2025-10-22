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
	if clientRequest.extraProperties == nil {
		clientRequest.extraProperties = make(map[string]interface{})
	}

	appType, ok := clientRequest.extraProperties[oktaApplicationType].(string)
	if !ok {
		appType = oktaAppTypeService
	}

	// Check if PKCE is required - if so, this should be a browser (public client) app
	pkceRequired, _ := clientRequest.extraProperties[oktaPKCERequired].(bool)

	for _, grantType := range clientRequest.GrantTypes {
		switch grantType {
		case GrantTypeClientCredentials:
			if len(clientRequest.ResponseTypes) == 0 {
				clientRequest.ResponseTypes = []string{AuthResponseToken}
			}
		default:
			if appType == oktaAppTypeService {
				if pkceRequired {
					appType = oktaAppTypeBrowser
					clientRequest.TokenEndpointAuthMethod = "none"
				} else {
					appType = oktaAppTypeWeb
				}
			}
		}
	}
	clientRequest.extraProperties[oktaApplicationType] = appType
}
