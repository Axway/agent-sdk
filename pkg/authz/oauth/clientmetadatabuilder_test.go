package oauth

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestClientBuilder(t *testing.T) {
	publicKey, err := util.ReadPublicKeyBytes("testdata/publickey")
	assert.Nil(t, err)

	certificate, err := util.ReadPublicKeyBytes("testdata/client_cert.pem")
	assert.Nil(t, err)

	cases := []struct {
		name                  string
		grantTypes            []string
		tokenAuthMethod       string
		responseType          []string
		redirectURIs          []string
		scopes                []string
		logoURI               string
		publicKey             []byte
		certificate           []byte
		certificateMetadata   string
		tlsClientAuthSanDNS   string
		tlsClientAuthSanEmail string
		tlsClientAuthSanIP    string
		tlsClientAuthSanURI   string
		expectErr             bool
	}{
		{
			name:            "test_build_with_authorization_code_no_redirect",
			grantTypes:      []string{GrantTypeAuthorizationCode},
			tokenAuthMethod: config.ClientSecretBasic,
			expectErr:       true,
		},
		{
			name:            "test_build_client_secret",
			grantTypes:      []string{GrantTypeClientCredentials},
			tokenAuthMethod: config.ClientSecretPost,
			responseType:    []string{AuthResponseToken},
			redirectURIs:    []string{"http://localhost"},
			scopes:          []string{"scope1", "scope2"},
			logoURI:         "http://localhost",
		},
		{
			name:            "test_build_with_private_key_jwt_no_jwks",
			grantTypes:      []string{GrantTypeClientCredentials},
			tokenAuthMethod: config.PrivateKeyJWT,
			responseType:    []string{AuthResponseToken},
			expectErr:       true,
		},
		{
			name:            "test_build_with_private_key_jwt_invalid_jwks",
			grantTypes:      []string{GrantTypeClientCredentials},
			tokenAuthMethod: config.PrivateKeyJWT,
			responseType:    []string{AuthResponseToken},
			publicKey:       []byte("invalid-public-key"),
			expectErr:       true,
		},
		{
			name:            "test_build_with_private_key_jwt_valid_jwks",
			grantTypes:      []string{GrantTypeClientCredentials},
			tokenAuthMethod: config.PrivateKeyJWT,
			responseType:    []string{AuthResponseToken},
			publicKey:       publicKey,
		},
		{
			name:            "test_build_with_tls_client_auth_no_jwks",
			grantTypes:      []string{GrantTypeClientCredentials},
			tokenAuthMethod: config.TLSClientAuth,
			responseType:    []string{AuthResponseToken},
			expectErr:       true,
		},
		{
			name:            "test_build_with_tls_client_auth_invalid_jwks",
			grantTypes:      []string{GrantTypeClientCredentials},
			tokenAuthMethod: config.TLSClientAuth,
			responseType:    []string{AuthResponseToken},
			certificate:     []byte("invalid-client-cert"),
			expectErr:       true,
		},
		{
			name:                  "test_build_with_tls_client_auth_valid_jwks_with_subject_dn",
			grantTypes:            []string{GrantTypeClientCredentials},
			tokenAuthMethod:       config.TLSClientAuth,
			responseType:          []string{AuthResponseToken},
			certificate:           certificate,
			tlsClientAuthSanDNS:   "san-dns",
			tlsClientAuthSanEmail: "san-email",
			tlsClientAuthSanIP:    "san-ip",
			tlsClientAuthSanURI:   "san-uri",
		},
		{
			name:                  "test_build_with_tls_client_auth_valid_jwks_with_san_dns",
			grantTypes:            []string{GrantTypeClientCredentials},
			tokenAuthMethod:       config.TLSClientAuth,
			responseType:          []string{AuthResponseToken},
			certificate:           certificate,
			certificateMetadata:   TLSClientAuthSanDNS,
			tlsClientAuthSanDNS:   "san-dns",
			tlsClientAuthSanEmail: "san-email",
			tlsClientAuthSanIP:    "san-ip",
			tlsClientAuthSanURI:   "san-uri",
		},
		{
			name:                  "test_build_with_tls_client_auth_valid_jwks_with_san_email",
			grantTypes:            []string{GrantTypeClientCredentials},
			tokenAuthMethod:       config.TLSClientAuth,
			responseType:          []string{AuthResponseToken},
			certificate:           certificate,
			certificateMetadata:   TLSClientAuthSanEmail,
			tlsClientAuthSanDNS:   "san-dns",
			tlsClientAuthSanEmail: "san-email",
			tlsClientAuthSanIP:    "san-ip",
			tlsClientAuthSanURI:   "san-uri",
		},
		{
			name:                  "test_build_with_tls_client_auth_valid_jwks_with_san_ip",
			grantTypes:            []string{GrantTypeClientCredentials},
			tokenAuthMethod:       config.TLSClientAuth,
			responseType:          []string{AuthResponseToken},
			certificate:           certificate,
			certificateMetadata:   TLSClientAuthSanIP,
			tlsClientAuthSanDNS:   "san-dns",
			tlsClientAuthSanEmail: "san-email",
			tlsClientAuthSanIP:    "san-ip",
			tlsClientAuthSanURI:   "san-uri",
		},
		{
			name:                  "test_build_with_tls_client_auth_valid_jwks_with_san_uri",
			grantTypes:            []string{GrantTypeClientCredentials},
			tokenAuthMethod:       config.TLSClientAuth,
			responseType:          []string{AuthResponseToken},
			certificate:           certificate,
			certificateMetadata:   TLSClientAuthSanURI,
			tlsClientAuthSanDNS:   "san-dns",
			tlsClientAuthSanEmail: "san-email",
			tlsClientAuthSanIP:    "san-ip",
			tlsClientAuthSanURI:   "san-uri",
		},
	}
	for _, tc := range cases {
		builder := NewClientMetadataBuilder().
			SetClientName(tc.name).
			SetGrantTypes(tc.grantTypes).
			SetTokenEndpointAuthMethod(tc.tokenAuthMethod).
			SetResponseType(tc.responseType).
			SetRedirectURIs(tc.redirectURIs).
			SetScopes(tc.scopes).
			SetLogoURI(tc.logoURI)
		if tc.publicKey != nil {
			builder.SetJWKS(tc.publicKey)
		}
		if tc.certificate != nil {
			builder.SetJWKS(tc.certificate).
				SetCertificateMetadata(tc.certificateMetadata).
				SetTLSClientAuthSanDNS(tc.tlsClientAuthSanDNS).
				SetTLSClientAuthSanEmail(tc.tlsClientAuthSanEmail).
				SetTLSClientAuthSanIP(tc.tlsClientAuthSanIP).
				SetTLSClientAuthSanURI(tc.tlsClientAuthSanURI)
		}

		client, err := builder.Build()
		if tc.expectErr {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
			assert.NotNil(t, client)
			assert.Equal(t, tc.name, client.GetClientName())
			assert.Equal(t, tc.grantTypes, client.GetGrantTypes())
			assert.Equal(t, tc.tokenAuthMethod, client.GetTokenEndpointAuthMethod())
			assert.Equal(t, tc.responseType, client.GetResponseTypes())
			assert.Equal(t, tc.redirectURIs, client.GetRedirectURIs())
			assert.Equal(t, tc.scopes, client.GetScopes())
			assert.Equal(t, tc.logoURI, client.GetLogoURI())

			// assert client metadata
			if tc.publicKey != nil {
				assert.NotEmpty(t, client.GetJwks())
			}
			if tc.certificate != nil {
				assert.NotEmpty(t, client.GetJwks())
				switch tc.certificateMetadata {
				case TLSClientAuthSanDNS:
					assert.Equal(t, tc.tlsClientAuthSanDNS, client.GetTLSClientAuthSanDNS())
					assert.Equal(t, "", client.GetTLSClientAuthSanEmail())
					assert.Equal(t, "", client.GetTLSClientAuthSanIP())
					assert.Equal(t, "", client.GetTLSClientAuthSanURI())
				case TLSClientAuthSanEmail:
					assert.Equal(t, tc.tlsClientAuthSanEmail, client.GetTLSClientAuthSanEmail())
					assert.Equal(t, "", client.GetTLSClientAuthSanDNS())
					assert.Equal(t, "", client.GetTLSClientAuthSanIP())
					assert.Equal(t, "", client.GetTLSClientAuthSanURI())
				case TLSClientAuthSanIP:
					assert.Equal(t, tc.tlsClientAuthSanIP, client.GetTLSClientAuthSanIP())
					assert.Equal(t, "", client.GetTLSClientAuthSanDNS())
					assert.Equal(t, "", client.GetTLSClientAuthSanEmail())
					assert.Equal(t, "", client.GetTLSClientAuthSanURI())
				case TLSClientAuthSanURI:
					assert.Equal(t, tc.tlsClientAuthSanURI, client.GetTLSClientAuthSanURI())
					assert.Equal(t, "", client.GetTLSClientAuthSanDNS())
					assert.Equal(t, "", client.GetTLSClientAuthSanEmail())
					assert.Equal(t, "", client.GetTLSClientAuthSanIP())
				}

			}
		}
	}
}
