package oauth

import (
	"fmt"
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

type okta struct {
}

func (i *okta) getAuthorizationHeaderPrefix() string {
	return oktaAuthHeaderPrefix
}

// validateExtraProperties validates Okta-specific extra properties
func (i *okta) validateExtraProperties(extraProps map[string]interface{}) error {
	// If PKCE is required, application_type (if set) must be 'browser'
	pkceRequired, ok := extraProps[oktaPKCERequired].(bool)
	if !ok || !pkceRequired {
		return nil
	}

	appType, ok := extraProps[oktaApplicationType].(string)
	if !ok || appType == "" {
		return nil // Not set is valid
	}

	if appType != oktaAppTypeBrowser {
		return fmt.Errorf("when %s is true, %s must be '%s' or unset, got '%s'",
			oktaPKCERequired, oktaApplicationType, oktaAppTypeBrowser, appType)
	}
	return nil
}

func (i *okta) preProcessClientRequest(clientRequest *clientMetadata) {
	if clientRequest.extraProperties == nil {
		clientRequest.extraProperties = make(map[string]interface{})
	}

	pkceRequired, _ := clientRequest.extraProperties[oktaPKCERequired].(bool)
	_, hasAppType := clientRequest.extraProperties[oktaApplicationType].(string)

	// Process grant types to set defaults
	appType := oktaAppTypeService
	for _, grantType := range clientRequest.GrantTypes {
		if grantType == GrantTypeClientCredentials {
			if len(clientRequest.ResponseTypes) == 0 {
				clientRequest.ResponseTypes = []string{AuthResponseToken}
			}
		} else if !hasAppType {
			// Non-client-credentials flow needs web or browser type
			if pkceRequired {
				appType = oktaAppTypeBrowser
			} else {
				appType = oktaAppTypeWeb
			}
		}
	}

	// Set application_type if not already set
	if !hasAppType {
		clientRequest.extraProperties[oktaApplicationType] = appType
	}

	if pkceRequired {
		clientRequest.TokenEndpointAuthMethod = "none"
	}
}
