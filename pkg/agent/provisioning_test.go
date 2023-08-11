package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/mock"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestNewCredentialRequestBuilder(t *testing.T) {
	idp := oauth.NewMockIDPServer()
	defer idp.Close()

	s := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	defer s.Close()
	cfg := createCentralCfg(s.URL, "test")
	InitializeWithAgentFeatures(cfg, &config.AgentFeaturesConfiguration{MarketplaceProvisioning: true})

	agent.apicClient = &mock.Client{
		CreateOrUpdateResourceMock: func(data v1.Interface) (*v1.ResourceInstance, error) {
			ri, _ := data.AsInstance()
			return ri, nil
		},
		UpdateResourceInstanceMock: func(data v1.Interface) (*v1.ResourceInstance, error) {
			ri, _ := data.AsInstance()
			return ri, nil
		},
	}

	tests := []struct {
		name         string
		expectedName string
	}{
		{
			name:         "Test Basic Auth Helper",
			expectedName: "http-basic",
		},
		{
			name:         "Test APIKey Helper",
			expectedName: "api-key",
		},
		{
			name:         "Test OAuth Helper",
			expectedName: "oauth",
		},
		{
			name:         "Test OAuth External Helper",
			expectedName: "oauth-external",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var err error
			var crd *management.CredentialRequestDefinition
			switch test.expectedName {
			case "http-basic":
				crd, err = NewBasicAuthCredentialRequestBuilder().Register()
			case "api-key":
				crd, err = NewAPIKeyCredentialRequestBuilder().Register()
			case "oauth":
				crd, err = NewOAuthCredentialRequestBuilder().Register()
			case "oauth-external":
				cfg := &config.IDPConfiguration{
					Name:        "test",
					Type:        "okta",
					MetadataURL: idp.GetMetadataURL(),
					AuthConfig: &config.IDPAuthConfiguration{
						Type:         "client",
						ClientID:     "test",
						ClientSecret: "test",
					},
					GrantType:        oauth.GrantTypeClientCredentials,
					ClientScopes:     "read,write",
					AuthMethod:       config.ClientSecretBasic,
					AuthResponseType: "token",
					ExtraProperties:  config.ExtraProperties{"key": "value"},
				}

				p, _ := oauth.NewProvider(cfg, config.NewTLSConfig(), "", 30*time.Second)
				crd, err = NewOAuthCredentialRequestBuilder(
					WithCRDOAuthSecret(),
					WithCRDForIDP(p, []string{}),
				).Register()
			default:
				crd, err = NewCredentialRequestBuilder().Register()
			}
			assert.NotNil(t, crd)
			assert.Nil(t, err)
		})
	}
}
