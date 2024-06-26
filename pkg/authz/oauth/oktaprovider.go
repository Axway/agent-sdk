package oauth

const (
	oktaApplicationType  = "application_type"
	oktaAppTypeService   = "service"
	oktaAppTypeWeb       = "web"
	oktaAuthHeaderPrefix = "SSWS"
)

type okta struct {
}

func (i *okta) getAuthorizationHeaderPrefix() string {
	return oktaAuthHeaderPrefix
}

func (i *okta) preProcessClientRequest(clientRequest *clientMetadata) {
	if len(clientRequest.extraProperties) == 0 {
		clientRequest.extraProperties = make(map[string]string)
	}
	appType, ok := clientRequest.extraProperties[oktaApplicationType]
	if !ok {
		appType = oktaAppTypeService
	}

	for _, grantTypes := range clientRequest.GrantTypes {
		if grantTypes != GrantTypeClientCredentials {
			appType = oktaAppTypeWeb
		} else {
			if len(clientRequest.ResponseTypes) == 0 {
				clientRequest.ResponseTypes = []string{AuthResponseToken}
			}
		}
	}
	clientRequest.extraProperties[oktaApplicationType] = appType
}
