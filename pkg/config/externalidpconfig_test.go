package config

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func setEnvVars(t *testing.T, env map[string]string) {
	t.Helper()
	for key, val := range env {
		assert.NoError(t, os.Setenv(key, val))
	}
	t.Cleanup(func() {
		for key := range env {
			_ = os.Unsetenv(key)
		}
	})
}

func parseExternalIDP(t *testing.T) ExternalIDPConfig {
	t.Helper()
	prop := properties.NewProperties(&cobra.Command{})
	AddAgentFeaturesConfigProperties(prop)
	agentFeatures := &AgentFeaturesConfiguration{}
	assert.NoError(t, ParseExternalIDPConfig(agentFeatures, prop))
	assert.NotNil(t, agentFeatures.ExternalIDPConfig)
	return agentFeatures.ExternalIDPConfig
}

func assertIDPRoundTrip(t *testing.T, idp IDPConfig, expectedOktaGroup string, expectedOktaPolicy string) {
	t.Helper()
	buf, err := json.Marshal(idp)
	assert.NoError(t, err)
	assert.NotNil(t, buf)

	parsedIDP := &IDPConfiguration{}
	assert.NoError(t, json.Unmarshal(buf, parsedIDP))

	assert.Equal(t, idp.GetIDPName(), parsedIDP.GetIDPName())
	assert.Equal(t, idp.GetIDPType(), parsedIDP.GetIDPType())
	assert.Equal(t, idp.GetMetadataURL(), parsedIDP.GetMetadataURL())
	assert.Equal(t, len(idp.GetRequestHeaders()), len(parsedIDP.GetRequestHeaders()))
	assert.Equal(t, len(idp.GetQueryParams()), len(parsedIDP.GetQueryParams()))
	assert.Equal(t, idp.GetAuthConfig().GetType(), parsedIDP.GetAuthConfig().GetType())
	assert.Equal(t, idp.GetAuthConfig().GetAccessToken(), parsedIDP.GetAuthConfig().GetAccessToken())
	assert.Equal(t, idp.GetAuthConfig().GetClientID(), parsedIDP.GetAuthConfig().GetClientID())
	assert.Equal(t, idp.GetAuthConfig().GetClientSecret(), parsedIDP.GetAuthConfig().GetClientSecret())
	assert.Equal(t, len(idp.GetAuthConfig().GetRequestHeaders()), len(parsedIDP.GetAuthConfig().GetRequestHeaders()))
	assert.Equal(t, len(idp.GetAuthConfig().GetQueryParams()), len(parsedIDP.GetAuthConfig().GetQueryParams()))

	if expectedOktaGroup != "" {
		assert.Equal(t, expectedOktaGroup, idp.GetOktaGroup())
		assert.Equal(t, expectedOktaGroup, parsedIDP.GetOktaGroup())
	}
	if expectedOktaPolicy != "" {
		assert.Equal(t, expectedOktaPolicy, idp.GetOktaPolicy())
		assert.Equal(t, expectedOktaPolicy, parsedIDP.GetOktaPolicy())
	}
}

type externalIDPTestCase struct {
	name       string
	envNames   map[string]string
	oktaGroup  string
	oktaPolicy string
	hasError   bool
}

func runExternalIDPTestCase(t *testing.T, tc externalIDPTestCase) {
	t.Helper()
	setEnvVars(t, tc.envNames)

	idpCfgs := parseExternalIDP(t)
	err := idpCfgs.ValidateCfg()
	if tc.hasError {
		assert.Error(t, err)
		return
	}
	assert.NoError(t, err)
	for _, idp := range idpCfgs.GetIDPList() {
		assertIDPRoundTrip(t, idp, tc.oktaGroup, tc.oktaPolicy)
	}
}

func TestExternalIDPConfig(t *testing.T) {
	testCases := []externalIDPTestCase{
		{
			name:     "no external IDP config",
			envNames: map[string]string{},
			hasError: false,
		},
		{
			name: "no name in IDP config",
			envNames: map[string]string{
				"AGENTFEATURES_IDP_METADATAURL_1": "test",
			},
			hasError: true,
		},
		{
			name: "no metadata URL in IDP config",
			envNames: map[string]string{
				"AGENTFEATURES_IDP_NAME_1": "test",
			},
			hasError: true,
		},
		{
			name: "no auth config in IDP config",
			envNames: map[string]string{
				"AGENTFEATURES_IDP_NAME_1":        "test",
				"AGENTFEATURES_IDP_METADATAURL_1": "test",
			},
			hasError: true,
		},
		{
			name: "invalid IDP auth type config in IDP config",
			envNames: map[string]string{
				"AGENTFEATURES_IDP_NAME_1":        "test",
				"AGENTFEATURES_IDP_METADATAURL_1": "test",
				"AGENTFEATURES_IDP_AUTH_TYPE_1":   "invalid",
			},
			hasError: true,
		},
		{
			name: "accessToken auth config with no token in IDP config",
			envNames: map[string]string{
				"AGENTFEATURES_IDP_NAME_1":        "test",
				"AGENTFEATURES_IDP_METADATAURL_1": "test",
				"AGENTFEATURES_IDP_AUTH_TYPE_1":   "accessToken",
			},
			hasError: true,
		},
		{
			name: "accessToken auth config with valid token in IDP config",
			envNames: map[string]string{
				"AGENTFEATURES_IDP_NAME_1":             "test",
				"AGENTFEATURES_IDP_METADATAURL_1":      "test",
				"AGENTFEATURES_IDP_AUTH_TYPE_1":        "accessToken",
				"AGENTFEATURES_IDP_AUTH_ACCESSTOKEN_1": "accessToken",
			},
			hasError: false,
		},
		{
			name: "okta group config via env var",
			envNames: map[string]string{
				"AGENTFEATURES_IDP_NAME_1":             "test",
				"AGENTFEATURES_IDP_TYPE_1":             "okta",
				"AGENTFEATURES_IDP_METADATAURL_1":      "test",
				"AGENTFEATURES_IDP_AUTH_TYPE_1":        "accessToken",
				"AGENTFEATURES_IDP_AUTH_ACCESSTOKEN_1": "accessToken",
				"AGENTFEATURES_IDP_OKTA_GROUP_1":       "MyAppUsers",
			},
			oktaGroup: "MyAppUsers",
			hasError:  false,
		},
		{
			name: "okta policy config via env var",
			envNames: map[string]string{
				"AGENTFEATURES_IDP_NAME_1":             "test",
				"AGENTFEATURES_IDP_TYPE_1":             "okta",
				"AGENTFEATURES_IDP_METADATAURL_1":      "test",
				"AGENTFEATURES_IDP_AUTH_TYPE_1":        "accessToken",
				"AGENTFEATURES_IDP_AUTH_ACCESSTOKEN_1": "accessToken",
				"AGENTFEATURES_IDP_OKTA_POLICY_1":      `{"policyTemplate":{"name":"AutoPolicy"}}`,
			},
			oktaPolicy: `{"policyTemplate":{"name":"AutoPolicy"}}`,
			hasError:   false,
		},
		{
			name: "client auth config with no clientid/secret in IDP config",
			envNames: map[string]string{
				"AGENTFEATURES_IDP_NAME_1":        "test",
				"AGENTFEATURES_IDP_METADATAURL_1": "test",
				"AGENTFEATURES_IDP_AUTH_TYPE_1":   "client",
			},
			hasError: true,
		},
		{
			name: "client auth config with no client secret in IDP config",
			envNames: map[string]string{
				"AGENTFEATURES_IDP_NAME_1":          "test",
				"AGENTFEATURES_IDP_METADATAURL_1":   "test",
				"AGENTFEATURES_IDP_AUTH_TYPE_1":     "client",
				"AGENTFEATURES_IDP_AUTH_CLIENTID_1": "client-id",
			},
			hasError: true,
		},
		{
			name: "client auth config with valid client config in IDP config",
			envNames: map[string]string{
				"AGENTFEATURES_IDP_NAME_1":                "test",
				"AGENTFEATURES_IDP_METADATAURL_1":         "test",
				"AGENTFEATURES_IDP_REQUESTHEADERS_1":      "{\"hdr\":\"value\"}",
				"AGENTFEATURES_IDP_QUERYPARAMS_1":         "{\"param\":\"value\"}",
				"AGENTFEATURES_IDP_AUTH_TYPE_1":           "client",
				"AGENTFEATURES_IDP_AUTH_CLIENTID_1":       "client-id",
				"AGENTFEATURES_IDP_AUTH_CLIENTSECRET_1":   "client-secret",
				"AGENTFEATURES_IDP_AUTH_REQUESTHEADERS_1": "{\"authhdr\":\"value\"}",
				"AGENTFEATURES_IDP_AUTH_QUERYPARAMS_1":    "{\"authparam\":\"value\"}",
			},
			hasError: false,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			runExternalIDPTestCase(t, tc)
		})
	}
}
