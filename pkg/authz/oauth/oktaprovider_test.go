package oauth

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOktaPostProcessClientRegistration(t *testing.T) {
	oktaProvider := &okta{}
	extraProps := map[string]interface{}{
		"group":        "MyAppUsers",
		"createPolicy": true,
		"authServerId": "default",
		"policyTemplate": map[string]interface{}{
			"name":        "AutoPolicy-Test",
			"description": "Auto-created",
			"rule": map[string]interface{}{
				"name":       "AutoRule-Test",
				"conditions": map[string]interface{}{"grantTypes": map[string]interface{}{"include": []string{"authorization_code"}}},
				"actions":    map[string]interface{}{"token": map[string]interface{}{"accessTokenLifetime": 3600}},
			},
		},
		"createScopes": true,
		"scopes": []interface{}{
			map[string]interface{}{"name": "read:items", "description": "Read items"},
			map[string]interface{}{"name": "write:items", "description": "Write items"},
		},
	}
	clientRes := &clientMetadata{ClientID: "app123"}
	// Mock oktaapi.New and methods if needed
	// For now, just check no error
	err := oktaProvider.postProcessClientRegistration(clientRes, extraProps, nil)
	assert.NoError(t, err)
}

func TestOktaPostProcessClientUnregister(t *testing.T) {
	oktaProvider := &okta{}
	extraProps := map[string]interface{}{
		"authServerId": "default",
	}
	agentDetails := map[string]string{
		"oktaPolicyId": "pol-123",
		"oktaRuleId":   "r-456",
		"oktaGroupId":  "00g-789",
		"oktaScopeId":  "scp-101",
	}
	clientID := "app123"
	// Mock oktaapi.New and methods if needed
	// For now, just check no error
	err := oktaProvider.postProcessClientUnregister(clientID, agentDetails, extraProps, nil)
	assert.NoError(t, err)
}

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

func TestValidateOktaExtraProperties(t *testing.T) {
	cases := []struct {
		name        string
		extraProps  map[string]interface{}
		expectError bool
	}{
		{
			name: "Valid: PKCE with browser type",
			extraProps: map[string]interface{}{
				oktaPKCERequired:    true,
				oktaApplicationType: oktaAppTypeBrowser,
			},
			expectError: false,
		},
		{
			name: "Valid: PKCE without app type",
			extraProps: map[string]interface{}{
				oktaPKCERequired: true,
			},
			expectError: false,
		},
		{
			name: "Invalid: PKCE with service type",
			extraProps: map[string]interface{}{
				oktaPKCERequired:    true,
				oktaApplicationType: oktaAppTypeService,
			},
			expectError: true,
		},
		{
			name: "Invalid: PKCE with web type",
			extraProps: map[string]interface{}{
				oktaPKCERequired:    true,
				oktaApplicationType: oktaAppTypeWeb,
			},
			expectError: true,
		},
		{
			name: "Valid: No PKCE with any type",
			extraProps: map[string]interface{}{
				oktaApplicationType: oktaAppTypeService,
			},
			expectError: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			oktaProvider := &okta{}
			err := oktaProvider.validateExtraProperties(tc.extraProps)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
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
			name:       "Explicit browser type should be preserved with PKCE",
			grantTypes: []string{GrantTypeAuthorizationCode},
			extraProperties: map[string]interface{}{
				oktaApplicationType: oktaAppTypeBrowser,
				oktaPKCERequired:    true,
			},
			expectedAppType:    oktaAppTypeBrowser,
			expectedAuthMethod: "none",
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

			// Simulate validation step which sets defaults (as happens in NewProvider)
			_ = oktaProvider.validateExtraProperties(clientReq.extraProperties)

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
