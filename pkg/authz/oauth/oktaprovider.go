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
	if clientRequest.extraProperties == nil {
		clientRequest.extraProperties = make(map[string]string)
	}

	appType, ok := clientRequest.extraProperties[oktaApplicationType]
	if !ok {
		appType = oktaAppTypeService
	}

	for _, grantType := range clientRequest.GrantTypes {
		if grantType != GrantTypeClientCredentials {
			// Allow "browser" to be set by user, otherwise default to "web"
			if appType != OktaAppTypeBrowser {
				appType = oktaAppTypeWeb
			}
		} else {
			if len(clientRequest.ResponseTypes) == 0 {
				clientRequest.ResponseTypes = []string{AuthResponseToken}
			}
		}
	}

	clientRequest.extraProperties[oktaApplicationType] = appType
	convertStringBoolsToMarker(clientRequest.extraProperties)
}

// knownBoolPropsSet allows O(1) lookup and easy extension
var knownBoolPropsSet = map[string]struct{}{
	OktaPKCERequired: {},
}

func convertStringBoolsToMarker(m map[string]string) {
	for prop := range knownBoolPropsSet {
		if val, ok := m[prop]; ok && (val == StringTrue || val == StringFalse) {
			delete(m, prop)
			m[prop+SuffixBool] = val
		}
	}
}
