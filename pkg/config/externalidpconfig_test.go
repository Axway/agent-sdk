package config

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

const (
	testIDPName     = "test"
	testMetaURL     = "test"
	testAccessTok   = "accessToken"
	testIDPTypeOkta = "okta"

	envKeyIDPName                   = "AGENTFEATURES_IDP_NAME_1"
	envKeyIDPType                   = "AGENTFEATURES_IDP_TYPE_1"
	envKeyIDPMetadataURL            = "AGENTFEATURES_IDP_METADATAURL_1"
	envKeyIDPRequestHeaders         = "AGENTFEATURES_IDP_REQUESTHEADERS_1"
	envKeyIDPQueryParams            = "AGENTFEATURES_IDP_QUERYPARAMS_1"
	envKeyIDPAuthType               = "AGENTFEATURES_IDP_AUTH_TYPE_1"
	envKeyIDPAuthAccessToken        = "AGENTFEATURES_IDP_AUTH_ACCESSTOKEN_1"
	envKeyIDPAuthClientID           = "AGENTFEATURES_IDP_AUTH_CLIENTID_1"
	envKeyIDPAuthClientSecret       = "AGENTFEATURES_IDP_AUTH_CLIENTSECRET_1"
	envKeyIDPAuthRequestHeaders     = "AGENTFEATURES_IDP_AUTH_REQUESTHEADERS_1"
	envKeyIDPAuthQueryParams        = "AGENTFEATURES_IDP_AUTH_QUERYPARAMS_1"
	envKeyIDPOktaAppNameTemplate    = "AGENTFEATURES_IDP_OKTA_APPNAME_TEMPLATE_1"
	envKeyIDPOktaPolicyNameTemplate = "AGENTFEATURES_IDP_OKTA_POLICYNAME_TEMPLATE_1"
	envKeyIDPOktaScopeSources       = "AGENTFEATURES_IDP_OKTA_SCOPE_SOURCES_1"
	envKeyIDPOktaScopeBlacklist     = "AGENTFEATURES_IDP_OKTA_SCOPE_BLACKLIST_1"
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

func assertIDPRoundTrip(t *testing.T, idp IDPConfig) {
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
}

type externalIDPTestCase struct {
	envNames map[string]string
	hasError bool
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
		assertIDPRoundTrip(t, idp)
	}
}

func mergeEnv(base, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func TestExternalIDPConfig(t *testing.T) {
	baseOkta := map[string]string{
		envKeyIDPName:            testIDPName,
		envKeyIDPType:            testIDPTypeOkta,
		envKeyIDPMetadataURL:     testMetaURL,
		envKeyIDPAuthType:        AccessToken,
		envKeyIDPAuthAccessToken: testAccessTok,
	}

	cases := map[string]externalIDPTestCase{
		"no external IDP config": {envNames: map[string]string{}},
		"no name in IDP config": {
			envNames: map[string]string{envKeyIDPMetadataURL: testMetaURL},
			hasError: true,
		},
		"no metadata URL in IDP config": {
			envNames: map[string]string{envKeyIDPName: testIDPName},
			hasError: true,
		},
		"no auth config in IDP config": {
			envNames: map[string]string{
				envKeyIDPName:        testIDPName,
				envKeyIDPMetadataURL: testMetaURL,
			},
			hasError: true,
		},
		"invalid IDP auth type config": {
			envNames: map[string]string{
				envKeyIDPName:        testIDPName,
				envKeyIDPMetadataURL: testMetaURL,
				envKeyIDPAuthType:    "invalid",
			},
			hasError: true,
		},
		"accessToken auth config with no token": {
			envNames: map[string]string{
				envKeyIDPName:        testIDPName,
				envKeyIDPMetadataURL: testMetaURL,
				envKeyIDPAuthType:    AccessToken,
			},
			hasError: true,
		},
		"accessToken auth config with valid token": {
			envNames: map[string]string{
				envKeyIDPName:            testIDPName,
				envKeyIDPMetadataURL:     testMetaURL,
				envKeyIDPAuthType:        AccessToken,
				envKeyIDPAuthAccessToken: testAccessTok,
			},
		},
		"okta appname template config via env var": {
			envNames: mergeEnv(baseOkta, map[string]string{envKeyIDPOktaAppNameTemplate: OktaPlaceholderMPApplicationName}),
		},
		"okta policy name template config via env var": {
			envNames: mergeEnv(baseOkta, map[string]string{envKeyIDPOktaPolicyNameTemplate: OktaPlaceholderScope}),
		},
		"okta scope sources config via env var": {
			envNames: mergeEnv(baseOkta, map[string]string{envKeyIDPOktaScopeSources: "swagger,okta"}),
		},
		"okta scope blacklist config via env var": {
			envNames: mergeEnv(baseOkta, map[string]string{envKeyIDPOktaScopeBlacklist: "openid,profile"}),
		},
		"client auth config with no clientid/secret": {
			envNames: map[string]string{
				envKeyIDPName:        testIDPName,
				envKeyIDPMetadataURL: testMetaURL,
				envKeyIDPAuthType:    Client,
			},
			hasError: true,
		},
		"client auth config with no client secret": {
			envNames: map[string]string{
				envKeyIDPName:         testIDPName,
				envKeyIDPMetadataURL:  testMetaURL,
				envKeyIDPAuthType:     Client,
				envKeyIDPAuthClientID: "client-id",
			},
			hasError: true,
		},
		"client auth config with valid client config": {
			envNames: map[string]string{
				envKeyIDPName:               testIDPName,
				envKeyIDPMetadataURL:        testMetaURL,
				envKeyIDPRequestHeaders:     `{"hdr":"value"}`,
				envKeyIDPQueryParams:        `{"param":"value"}`,
				envKeyIDPAuthType:           Client,
				envKeyIDPAuthClientID:       "client-id",
				envKeyIDPAuthClientSecret:   "client-secret",
				envKeyIDPAuthRequestHeaders: `{"authhdr":"value"}`,
				envKeyIDPAuthQueryParams:    `{"authparam":"value"}`,
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			runExternalIDPTestCase(t, tc)
		})
	}
}

func TestOktaIDPConfigGetters(t *testing.T) {
	cases := map[string]struct {
		cfg                *IDPConfiguration
		wantAppTemplate    string
		wantPolicyTemplate string
		wantScopeSources   string
		wantScopeBlacklist string
		wantGroup          string
	}{
		"nil okta config returns all defaults": {
			cfg:                &IDPConfiguration{},
			wantAppTemplate:    defaultOktaAppNameTemplate,
			wantPolicyTemplate: defaultOktaPolicyNameTemplate,
			wantScopeSources:   defaultOktaScopeSources,
			wantScopeBlacklist: defaultOktaScopeBlacklist,
			wantGroup:          "",
		},
		"configured values are returned": {
			cfg: &IDPConfiguration{Okta: &OktaIDPConfiguration{
				Group:              "Marketplace",
				AppNameTemplate:    "my-" + OktaPlaceholderCredentialName,
				PolicyNameTemplate: OktaPlaceholderScope,
				ScopeSources:       "swagger",
				ScopeBlacklist:     "openid",
			}},
			wantAppTemplate:    "my-" + OktaPlaceholderCredentialName,
			wantPolicyTemplate: OktaPlaceholderScope,
			wantScopeSources:   "swagger",
			wantScopeBlacklist: "openid",
			wantGroup:          "Marketplace",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.wantAppTemplate, tc.cfg.GetAppNameTemplate())
			assert.Equal(t, tc.wantPolicyTemplate, tc.cfg.GetPolicyNameTemplate())
			assert.Equal(t, tc.wantScopeSources, tc.cfg.GetScopeSources())
			assert.Equal(t, tc.wantScopeBlacklist, tc.cfg.GetScopeBlacklist())
			assert.Equal(t, tc.wantGroup, tc.cfg.GetOktaGroup())
		})
	}
}

func TestRemovedSymbols(t *testing.T) {
	typ := reflect.TypeFor[OktaIDPConfiguration]()
	cases := map[string]struct{}{
		"Policy": {},
	}
	for field := range cases {
		t.Run(field, func(t *testing.T) {
			_, found := typ.FieldByName(field)
			assert.False(t, found, "field %q must not exist on OktaIDPConfiguration", field)
		})
	}
}
