package oauth

import (
	"fmt"
	"slices"
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
