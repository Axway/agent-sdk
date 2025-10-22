package oauth

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOktaPKCERequired(t *testing.T) {
	cases := []struct {
		name          string
		pkceRequired  bool
		expectedValue bool
	}{
		{
			name:          "PKCE required true",
			pkceRequired:  true,
			expectedValue: true,
		},
		{
			name:          "PKCE required false",
			pkceRequired:  false,
			expectedValue: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			props := map[string]interface{}{
				oktaPKCERequired: tc.pkceRequired,
			}
			c, err := NewClientMetadataBuilder().
				SetClientName(oktaSpa).
				SetExtraProperties(props).
				Build()
			assert.Nil(t, err)
			cm := c.(*clientMetadata)

			buf, err := json.Marshal(cm)
			assert.Nil(t, err)
			assert.NotNil(t, buf)

			var out map[string]interface{}
			err = json.Unmarshal(buf, &out)
			assert.Nil(t, err)

			// Should be a boolean, not a string
			val, ok := out[oktaPKCERequired]
			assert.True(t, ok)
			assert.IsType(t, tc.expectedValue, val)
			assert.Equal(t, tc.expectedValue, val)
		})
	}
}

func TestOktaPreProcessClientRequest(t *testing.T) {
	cases := []struct {
		name                  string
		grantTypes            []string
		responseTypes         []string
		extraProperties       map[string]interface{}
		expectedAppType       string
		expectedResponseTypes []string
		expectedAuthMethod    string
	}{
		{
			name:       "Authorization code with PKCE should use browser type",
			grantTypes: []string{GrantTypeAuthorizationCode},
			extraProperties: map[string]interface{}{
				oktaPKCERequired: true,
			},
			expectedAppType:    oktaAppTypeBrowser,
			expectedAuthMethod: "none",
		},
		{
			name:       "Authorization code without PKCE should use web type",
			grantTypes: []string{GrantTypeAuthorizationCode},
			extraProperties: map[string]interface{}{
				oktaPKCERequired: false,
			},
			expectedAppType: oktaAppTypeWeb,
		},
		{
			name:                  "Client credentials should remain service type",
			grantTypes:            []string{GrantTypeClientCredentials},
			responseTypes:         []string{},
			extraProperties:       map[string]interface{}{},
			expectedAppType:       oktaAppTypeService,
			expectedResponseTypes: []string{AuthResponseToken},
		},
		{
			name:       "Explicit browser type should be preserved",
			grantTypes: []string{GrantTypeAuthorizationCode},
			extraProperties: map[string]interface{}{
				oktaApplicationType: oktaAppTypeBrowser,
				oktaPKCERequired:    true,
			},
			expectedAppType: oktaAppTypeBrowser,
		},
		{
			name:       "Implicit flow without PKCE should use web type",
			grantTypes: []string{GrantTypeImplicit},
			extraProperties: map[string]interface{}{
				oktaPKCERequired: false,
			},
			expectedAppType: oktaAppTypeWeb,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			oktaProvider := &okta{}
			clientReq := &clientMetadata{
				GrantTypes:      tc.grantTypes,
				ResponseTypes:   tc.responseTypes,
				extraProperties: tc.extraProperties,
			}

			oktaProvider.preProcessClientRequest(clientReq)

			appType, ok := clientReq.extraProperties[oktaApplicationType].(string)
			assert.True(t, ok)
			assert.Equal(t, tc.expectedAppType, appType)

			if tc.expectedResponseTypes != nil {
				assert.Equal(t, tc.expectedResponseTypes, clientReq.ResponseTypes)
			}

			if tc.expectedAuthMethod != "" {
				assert.Equal(t, tc.expectedAuthMethod, clientReq.TokenEndpointAuthMethod)
			}
		})
	}
}
