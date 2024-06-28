package idp

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/stretchr/testify/assert"
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
		Type: "generic",
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

	cases := []struct {
		name                       string
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
		{
			name:                       "provisioner for non-IdP credential",
			credTokenURL:               "",
			registrationResponseCode:   http.StatusCreated,
			unRegistrationResponseCode: http.StatusNoContent,
		},
		{
			name:                       "provisioner for IdP credential with client_credential",
			credTokenURL:               s.GetTokenURL(),
			tokenAuthMethod:            config.ClientSecretPost,
			registrationResponseCode:   http.StatusCreated,
			unRegistrationResponseCode: http.StatusNoContent,
			appKey:                     "test-app-key",
			useRegistrationAccessToken: true,
		},
		{
			name:                       "provisioner for IdP credential with private_key_jwt",
			credTokenURL:               s.GetTokenURL(),
			tokenAuthMethod:            config.PrivateKeyJWT,
			registrationResponseCode:   http.StatusCreated,
			unRegistrationResponseCode: http.StatusNoContent,
			publicKey:                  string(publicKey),
		},
		{
			name:                       "provisioner for IdP credential with tls_client_auth",
			credTokenURL:               s.GetTokenURL(),
			tokenAuthMethod:            config.TLSClientAuth,
			registrationResponseCode:   http.StatusCreated,
			unRegistrationResponseCode: http.StatusNoContent,
			certificate:                string(certificate),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := management.NewManagedApplication("", "")
			app.Spec.Security.EncryptionKey = tc.appKey
			cred := management.NewCredential("", "")
			cred.Spec.Data = map[string]interface{}{
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
			cred.Data = map[string]interface{}{
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
