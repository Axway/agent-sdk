package agent

import (
	"errors"
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
	InitializeWithAgentFeatures(cfg, &config.AgentFeaturesConfiguration{}, nil)

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

func TestNewAccessRequestBuilder(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	defer s.Close()
	cfg := createCentralCfg(s.URL, "test")
	InitializeWithAgentFeatures(cfg, &config.AgentFeaturesConfiguration{})

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

	tests := map[string]struct {
		name    string
		apdName string
	}{
		"Test Basic Auth Helper": {
			name: "http-basic",
		},
		"Test APIKey Helper": {
			name: "api-key",
		},
		"Custom ARD": {
			name: "custom-ard",
		},
		"Validate APD added": {
			name:    "custom-ard",
			apdName: "apd",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			agent.applicationProfileDefinition = test.apdName
			var err error
			var ard *management.AccessRequestDefinition
			switch test.name {
			case "http-basic":
				ard, err = NewBasicAuthAccessRequestBuilder().SetApplicationProfileDefinition(test.apdName).Register()
			case "api-key":
				ard, err = NewAPIKeyAccessRequestBuilder().SetApplicationProfileDefinition(test.apdName).Register()
			default:
				ard, err = NewAccessRequestBuilder().SetApplicationProfileDefinition(test.apdName).SetName(test.name).Register()
			}
			assert.Equal(t, test.name, ard.Name)
			assert.Equal(t, test.apdName, ard.Applicationprofile.Name)
			assert.NotNil(t, ard)
			assert.Nil(t, err)
		})
	}
}

func TestNewApplicationProfileDefinitionBuilder(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	defer s.Close()
	cfg := createCentralCfg(s.URL, "test")
	InitializeWithAgentFeatures(cfg, &config.AgentFeaturesConfiguration{})
	var sentAPD *v1.ResourceInstance
	var createOrUpdateCalled bool

	agent.apicClient = &mock.Client{
		CreateOrUpdateResourceMock: func(data v1.Interface) (*v1.ResourceInstance, error) {
			sentAPD, _ = data.AsInstance()
			createOrUpdateCalled = true
			return sentAPD, nil
		},
	}

	tests := map[string]struct {
		title  string
		name   string
		exists bool
	}{
		"success when app profile does not exist": {
			title: "Test Application Profile 1",
			name:  "app-profile-1",
		},
		"success when app profile exists": {
			title:  "Test Application Profile 2",
			name:   "app-profile-2",
			exists: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			createOrUpdateCalled = false
			sentAPD = nil
			if tc.exists {
				apd := management.NewApplicationProfileDefinition(tc.name, agent.cfg.GetEnvironmentName())
				ri, _ := apd.AsInstance()
				agent.cacheManager.AddApplicationProfileDefinition(ri)
			}
			apd, err := NewApplicationProfileBuilder().
				SetName(tc.name).
				SetTitle(tc.title).
				Register()
			assert.NotNil(t, apd)
			assert.Nil(t, err)
			assert.Equal(t, tc.name, sentAPD.Name)
			assert.Equal(t, tc.title, sentAPD.Title)
			assert.True(t, createOrUpdateCalled)
		})
	}
}

func TestCleanApplicationProfileDefinition(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	defer s.Close()
	cfg := createCentralCfg(s.URL, "test")
	InitializeWithAgentFeatures(cfg, &config.AgentFeaturesConfiguration{})
	var deleteAPD *v1.ResourceInstance
	var deleteCalled, returnErr bool

	agent.apicClient = &mock.Client{
		DeleteResourceInstanceMock: func(data v1.Interface) error {
			deleteCalled = true
			if returnErr {
				return errors.New("error")
			}
			deleteAPD, _ = data.AsInstance()
			return nil
		},
	}

	tests := map[string]struct {
		name         string
		exists       bool
		returnErr    bool
		deleteCalled bool
		expectErr    bool
	}{
		"success removing when app profile does not exist": {
			name: "delete-app-profile-1",
		},
		"success removing when app profile exists": {
			name:         "delete-app-profile-2",
			exists:       true,
			deleteCalled: true,
		},
		"expect error when call to remove app profile fails": {
			name:         "delete-app-profile-3",
			exists:       true,
			deleteCalled: true,
			returnErr:    true,
			expectErr:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			returnErr = tc.returnErr
			deleteCalled = false
			deleteAPD = nil
			if tc.exists {
				apd := management.NewApplicationProfileDefinition(tc.name, agent.cfg.GetEnvironmentName())
				ri, _ := apd.AsInstance()
				agent.cacheManager.AddApplicationProfileDefinition(ri)
			}
			err := CleanApplicationProfileDefinition(tc.name)
			if tc.expectErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, tc.deleteCalled, deleteCalled)
			if !tc.deleteCalled || tc.expectErr {
				return
			}
			assert.Equal(t, tc.name, deleteAPD.Name)
		})
	}
}
