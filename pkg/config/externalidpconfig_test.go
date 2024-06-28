package config

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/stretchr/testify/assert"
)

func TestExternalIDPConfig(t *testing.T) {
	testCases := []struct {
		name     string
		envNames map[string]string
		hasError bool
	}{
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
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			for key, val := range test.envNames {
				os.Setenv(key, val)
			}
			defer func() {
				for key := range test.envNames {
					os.Setenv(key, "")
				}
			}()
			prop := properties.NewProperties(nil)
			AddAgentFeaturesConfigProperties(prop)
			cfg, err := ParseAgentFeaturesConfig(prop)
			assert.Nil(t, err)
			assert.NotNil(t, cfg)
			err = cfg.(*AgentFeaturesConfiguration).ValidateCfg()
			if test.hasError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				idpCfgs := cfg.GetExternalIDPConfig()
				for _, idp := range idpCfgs.GetIDPList() {
					buf, err := json.Marshal(idp)
					assert.Nil(t, err)
					assert.NotNil(t, buf)
					parsedIdP := &IDPConfiguration{}
					err = json.Unmarshal(buf, &parsedIdP)
					assert.Nil(t, err)
					assert.Equal(t, idp.GetIDPName(), parsedIdP.GetIDPName())
					assert.Equal(t, idp.GetIDPType(), parsedIdP.GetIDPType())
					assert.Equal(t, idp.GetMetadataURL(), parsedIdP.GetMetadataURL())
					assert.Equal(t, len(idp.GetRequestHeaders()), len(parsedIdP.GetRequestHeaders()))
					assert.Equal(t, len(idp.GetQueryParams()), len(parsedIdP.GetQueryParams()))
					assert.Equal(t, idp.GetAuthConfig().GetType(), parsedIdP.GetAuthConfig().GetType())
					assert.Equal(t, idp.GetAuthConfig().GetAccessToken(), parsedIdP.GetAuthConfig().GetAccessToken())
					assert.Equal(t, idp.GetAuthConfig().GetClientID(), parsedIdP.GetAuthConfig().GetClientID())
					assert.Equal(t, idp.GetAuthConfig().GetClientSecret(), parsedIdP.GetAuthConfig().GetClientSecret())
					assert.Equal(t, len(idp.GetAuthConfig().GetRequestHeaders()), len(parsedIdP.GetAuthConfig().GetRequestHeaders()))
					assert.Equal(t, len(idp.GetAuthConfig().GetQueryParams()), len(parsedIdP.GetAuthConfig().GetQueryParams()))

				}

			}
		})
	}
}
