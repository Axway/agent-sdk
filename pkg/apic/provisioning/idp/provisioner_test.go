package idp

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/stretchr/testify/assert"
)

const (
	testCredName = "my-cred"
	testAppName  = "my-app"
	fullTemplate = config.OktaPlaceholderMPApplicationName + "-" + config.OktaPlaceholderOwningTeam + "-" + config.OktaPlaceholderCredentialName
)

func TestProvisioner(t *testing.T) {
	publicKey, err := os.ReadFile("../../../authz/oauth/testdata/publickey")
	assert.Nil(t, err)
	certificate, err := util.ReadPublicKeyBytes("../../../authz/oauth/testdata/client_cert.pem")
	assert.Nil(t, err)

	s := oauth.NewMockIDPServer()
	defer s.Close()
	idpCfg := &config.IDPConfiguration{
		Name: "test",
		Type: oauth.TypeGeneric,
		AuthConfig: &config.IDPAuthConfiguration{
			Type:                 config.Client,
			ClientID:             "test",
			ClientSecret:         "test",
			UseRegistrationToken: true,
		},
		GrantType:   oauth.GrantTypeClientCredentials,
		AuthMethod:  config.ClientSecretBasic,
		MetadataURL: s.GetMetadataURL(),
	}

	s.SetMetadataResponseCode(http.StatusOK)
	p, err := oauth.NewProvider(idpCfg, config.NewTLSConfig(), "", 30*time.Second)
	assert.Nil(t, err)
	assert.NotNil(t, p)

	idpReg := oauth.NewIdpRegistry()
	err = idpReg.RegisterProvider(context.Background(), idpCfg, config.NewTLSConfig(), "", 30*time.Second)
	assert.Nil(t, err)

	cases := map[string]struct {
		idpType                    string
		appKey                     string
		credTokenURL               string
		tokenAuthMethod            string
		publicKey                  string
		certificate                string
		registrationResponseCode   int
		unRegistrationResponseCode int
		useRegistrationAccessToken bool
		expectRegistrationErr      bool
		expectUnRegistrationErr    bool
	}{
		"provisioner for non-IdP credential": {
			credTokenURL:               "",
			registrationResponseCode:   http.StatusCreated,
			unRegistrationResponseCode: http.StatusNoContent,
		},
		"provisioner for IdP credential with client_credential": {
			credTokenURL:               s.GetTokenURL(),
			tokenAuthMethod:            config.ClientSecretPost,
			registrationResponseCode:   http.StatusCreated,
			unRegistrationResponseCode: http.StatusNoContent,
			appKey:                     "test-app-key",
			useRegistrationAccessToken: true,
		},
		"provisioner for IdP credential with private_key_jwt": {
			credTokenURL:               s.GetTokenURL(),
			tokenAuthMethod:            config.PrivateKeyJWT,
			registrationResponseCode:   http.StatusCreated,
			unRegistrationResponseCode: http.StatusNoContent,
			publicKey:                  string(publicKey),
		},
		"provisioner for IdP credential with tls_client_auth": {
			credTokenURL:               s.GetTokenURL(),
			tokenAuthMethod:            config.TLSClientAuth,
			registrationResponseCode:   http.StatusCreated,
			unRegistrationResponseCode: http.StatusNoContent,
			certificate:                string(certificate),
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			app := management.NewManagedApplication("", "")
			app.Spec.Security.EncryptionKey = tc.appKey
			cred := management.NewCredential("", "")
			cred.Spec.Data = map[string]any{
				IDPTokenURL:                       tc.credTokenURL,
				provisioning.OauthTokenAuthMethod: tc.tokenAuthMethod,
				provisioning.OauthJwks:            tc.publicKey,
				provisioning.OauthCertificate:     tc.certificate,
			}

			provisioner, err := NewProvisioner(context.Background(), idpReg, app, cred)
			if tc.credTokenURL == "" {
				assert.NotNil(t, err)
				return
			}
			assert.Nil(t, err)

			s.SetRegistrationResponseCode(tc.registrationResponseCode)
			s.SetUseRegistrationAccessToken(tc.useRegistrationAccessToken)
			err = provisioner.RegisterClient()
			if tc.expectRegistrationErr {
				assert.NotNil(t, err)
				return
			}
			assert.Nil(t, err)

			data := provisioner.GetIDPCredentialData()
			assert.NotEmpty(t, data.GetClientID())
			assert.NotEmpty(t, data.GetClientSecret())

			cr, ok := data.(*credData)
			assert.True(t, ok)

			details, err := provisioner.GetAgentDetails()
			assert.Nil(t, err)
			assert.NotNil(t, details)
			if tc.useRegistrationAccessToken {
				assert.NotEmpty(t, cr.registrationAccessToken)
				assert.NotEmpty(t, details)
			}
			cred.Data = map[string]any{
				provisioning.OauthClientID:          data.GetClientID(),
				provisioning.OauthRegistrationToken: cr.registrationAccessToken,
			}
			util.SetAgentDetails(cred, util.MapStringStringToMapStringInterface(details))

			provisioner, err = NewProvisioner(context.Background(), idpReg, app, cred)
			assert.Nil(t, err)

			s.SetRegistrationResponseCode(tc.unRegistrationResponseCode)
			err = provisioner.UnregisterClient()
			if tc.expectUnRegistrationErr {
				assert.NotNil(t, err)
				return
			}
			assert.Nil(t, err)
			assert.NotNil(t, p)
		})
	}
}

type mockProvider struct {
	cfg            config.IDPConfig
	capturedScopes []string
	capturedGrant  string
	capturedName   string
}

func (m *mockProvider) GetName() string                                { return "" }
func (m *mockProvider) GetTitle() string                               { return "" }
func (m *mockProvider) GetIssuer() string                              { return "" }
func (m *mockProvider) GetTokenEndpoint() string                       { return "" }
func (m *mockProvider) GetMTLSTokenEndpoint() string                   { return "" }
func (m *mockProvider) GetAuthorizationEndpoint() string               { return "" }
func (m *mockProvider) GetSupportedScopes() []string                   { return nil }
func (m *mockProvider) GetSupportedGrantTypes() []string               { return nil }
func (m *mockProvider) GetSupportedTokenAuthMethods() []string         { return nil }
func (m *mockProvider) GetSupportedResponseMethod() []string           { return nil }
func (m *mockProvider) Validate() error                                { return nil }
func (m *mockProvider) GetConfig() config.IDPConfig                    { return m.cfg }
func (m *mockProvider) GetMetadata() *oauth.AuthorizationServerMetadata { return nil }
func (m *mockProvider) GetIDPResourceName() string                     { return "" }

func (m *mockProvider) RegisterClient(meta oauth.ClientMetadata) (oauth.ClientMetadata, error) {
	m.capturedName = meta.GetClientName()
	result, err := oauth.NewClientMetadataBuilder().SetClientName(meta.GetClientName()).Build()
	return result, err
}

func (m *mockProvider) UnregisterClient(clientID, accessToken, registrationClientURI string, scopes []string, grantType string) error {
	m.capturedScopes = scopes
	m.capturedGrant = grantType
	return nil
}

func TestScopeGrantTypeThreading(t *testing.T) {
	cases := map[string]struct {
		scopes    []string
		grantType string
	}{
		"multiple scopes with client credentials grant are passed to provider": {
			scopes:    []string{"scope1", "scope2"},
			grantType: oauth.GrantTypeClientCredentials,
		},
		"single scope with authorization code grant is passed to provider": {
			scopes:    []string{"read:api"},
			grantType: oauth.GrantTypeAuthorizationCode,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			mock := &mockProvider{cfg: nil}
			cred := management.NewCredential("", "")
			p := &provisioner{
				app:         management.NewManagedApplication("", ""),
				credential:  cred,
				idpProvider: mock,
				credentialData: &credData{
					scopes:     tc.scopes,
					grantTypes: []string{tc.grantType},
				},
			}
			assert.NoError(t, p.UnregisterClient())
			assert.Equal(t, tc.scopes, mock.capturedScopes)
			assert.Equal(t, tc.grantType, mock.capturedGrant)
		})
	}
}

func TestAppNameTemplate(t *testing.T) {
	cases := map[string]struct {
		cfg      config.IDPConfig
		appName  string
		teamName string
		credName string
		wantName string
		wantErr  bool
	}{
		"nil config falls back to credential name": {
			cfg:      nil,
			credName: testCredName,
			wantName: testCredName,
		},
		"non-okta IDP type falls back to credential name": {
			cfg:      &config.IDPConfiguration{Type: oauth.TypeGeneric},
			credName: testCredName,
			wantName: testCredName,
		},
		"template with all fields set": {
			cfg:      &config.IDPConfiguration{Type: oauth.TypeOkta, Okta: &config.OktaIDPConfiguration{AppNameTemplate: fullTemplate}},
			appName:  testAppName,
			teamName: "my-team",
			credName: testCredName,
			wantName: "my-app-my-team-my-cred",
		},
		"empty team name collapses double dash after normalize": {
			cfg:      &config.IDPConfiguration{Type: oauth.TypeOkta, Okta: &config.OktaIDPConfiguration{AppNameTemplate: fullTemplate}},
			appName:  testAppName,
			teamName: "",
			credName: testCredName,
			wantName: "my-app-my-cred",
		},
		"name exactly 100 chars passes": {
			cfg:      &config.IDPConfiguration{Type: oauth.TypeOkta, Okta: &config.OktaIDPConfiguration{AppNameTemplate: fullTemplate}},
			appName:  strings.Repeat("a", 48),
			teamName: strings.Repeat("b", 49),
			credName: "c",
			wantName: strings.Repeat("a", 48) + "-" + strings.Repeat("b", 49) + "-c",
		},
		"name exceeds 100 chars returns error": {
			cfg:      &config.IDPConfiguration{Type: oauth.TypeOkta, Okta: &config.OktaIDPConfiguration{AppNameTemplate: fullTemplate}},
			appName:  strings.Repeat("a", 50),
			teamName: strings.Repeat("b", 49),
			credName: strings.Repeat("c", 5),
			wantErr:  true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			mock := &mockProvider{cfg: tc.cfg}
			app := management.NewManagedApplication(tc.appName, "")
			if tc.teamName != "" {
				assert.NoError(t, util.SetAgentDetailsKey(app, agentDetailTeamName, tc.teamName))
			}
			cred := management.NewCredential(tc.credName, "")
			p := &provisioner{
				app:            app,
				credential:     cred,
				idpProvider:    mock,
				credentialData: &credData{},
			}
			got, err := p.appClientName()
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.wantName, got)
		})
	}
}
