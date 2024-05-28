package oauth

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"

	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	jwkcert "github.com/lestrrat-go/jwx/v2/cert"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

var grantTypeWithRedirects = map[string]bool{GrantTypeAuthorizationCode: true, GrantTypeImplicit: true}

// ClientBuilder - Builder for IdP client representation
type ClientBuilder interface {
	SetClientName(string) ClientBuilder

	SetScopes([]string) ClientBuilder
	SetGrantTypes([]string) ClientBuilder
	SetResponseType([]string) ClientBuilder
	SetTokenEndpointAuthMethod(tokenAuthMethod string) ClientBuilder

	SetRedirectURIs([]string) ClientBuilder
	SetLogoURI(string) ClientBuilder

	SetJWKSURI(string) ClientBuilder
	SetJWKS([]byte) ClientBuilder

	SetCertificateMetadata(certificateMetaddata string) ClientBuilder
	SetTLSClientAuthSanDNS(tlsClientAuthSanDNS string) ClientBuilder
	SetTLSClientAuthSanEmail(tlsClientAuthSanEmail string) ClientBuilder
	SetTLSClientAuthSanIP(tlsClientAuthSanIP string) ClientBuilder
	SetTLSClientAuthSanURI(tlsClientAuthSanURI string) ClientBuilder
	SetExtraProperties(map[string]string) ClientBuilder

	Build() (ClientMetadata, error)
}

type clientBuilder struct {
	jwks                  []byte
	jwksURI               string
	idpClientMetadata     *clientMetadata
	certificateMetadata   string
	tlsClientAuthSanDNS   string
	tlsClientAuthSanEmail string
	tlsClientAuthSanIP    string
	tlsClientAuthSanURI   string
}

// NewClientMetadataBuilder -  create a new instance of builder to construct client metadata
func NewClientMetadataBuilder() ClientBuilder {
	return &clientBuilder{
		idpClientMetadata: &clientMetadata{},
	}
}

func (b *clientBuilder) SetClientName(name string) ClientBuilder {
	b.idpClientMetadata.ClientName = name
	return b
}

func (b *clientBuilder) SetScopes(scopes []string) ClientBuilder {
	b.idpClientMetadata.Scope = Scopes(scopes)
	return b
}

func (b *clientBuilder) SetGrantTypes(grantTypes []string) ClientBuilder {
	b.idpClientMetadata.GrantTypes = grantTypes
	return b
}

func (b *clientBuilder) SetResponseType(responseTypes []string) ClientBuilder {
	b.idpClientMetadata.ResponseTypes = responseTypes
	return b
}

func (b *clientBuilder) SetTokenEndpointAuthMethod(tokenAuthMethod string) ClientBuilder {
	b.idpClientMetadata.TokenEndpointAuthMethod = tokenAuthMethod
	return b
}

func (b *clientBuilder) SetRedirectURIs(redirectURIs []string) ClientBuilder {
	b.idpClientMetadata.RedirectURIs = redirectURIs
	return b
}

func (b *clientBuilder) SetLogoURI(logoURI string) ClientBuilder {
	b.idpClientMetadata.LogoURI = logoURI
	return b
}

func (b *clientBuilder) SetJWKSURI(jwksURI string) ClientBuilder {
	b.jwksURI = jwksURI
	return b
}

func (b *clientBuilder) SetJWKS(jwks []byte) ClientBuilder {
	b.jwks = jwks
	return b
}

func (b *clientBuilder) SetCertificateMetadata(certificateMetadata string) ClientBuilder {
	b.certificateMetadata = certificateMetadata
	return b
}

func (b *clientBuilder) SetTLSClientAuthSanDNS(tlsClientAuthSanDNS string) ClientBuilder {
	b.tlsClientAuthSanDNS = tlsClientAuthSanDNS
	return b
}

func (b *clientBuilder) SetTLSClientAuthSanEmail(tlsClientAuthSanEmail string) ClientBuilder {
	b.tlsClientAuthSanEmail = tlsClientAuthSanEmail
	return b
}

func (b *clientBuilder) SetTLSClientAuthSanIP(tlsClientAuthSanIP string) ClientBuilder {
	b.tlsClientAuthSanIP = tlsClientAuthSanIP
	return b
}

func (b *clientBuilder) SetTLSClientAuthSanURI(tlsClientAuthSanURI string) ClientBuilder {
	b.tlsClientAuthSanURI = tlsClientAuthSanURI
	return b
}

func (b *clientBuilder) SetExtraProperties(extraProperties map[string]string) ClientBuilder {
	b.idpClientMetadata.extraProperties = extraProperties
	return b
}

func (b *clientBuilder) decodePublicKeyJWKS() (jwk.Key, error) {
	p, err := util.ParsePublicKey(b.jwks)
	if err != nil {
		return nil, err
	}

	key, err := jwk.FromRaw(p)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %s", err)
	}
	kid, _ := util.ComputeKIDFromDER(b.jwks)
	key.Set(jwk.KeyIDKey, kid)
	key.Set(jwk.KeyUsageKey, jwk.ForSignature)

	return key, nil
}

func (b *clientBuilder) decodeCertificateJWKS() (string, jwk.Key, error) {
	pemBlock, _ := pem.Decode(b.jwks)
	if pemBlock == nil {
		return "", nil, fmt.Errorf("failed to decode certificate")
	}

	cert, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse certificate: %s", err)
	}

	subjectDN := cert.Subject.String()
	key, err := jwk.FromRaw(cert.PublicKey)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse client certificate: %s", err)
	}

	c := &jwkcert.Chain{}
	c.Add([]byte(base64.StdEncoding.EncodeToString(pemBlock.Bytes)))
	key.Set(jwk.X509CertChainKey, c)
	key.Set(jwk.KeyUsageKey, jwk.ForSignature)

	return subjectDN, key, nil
}

func (b *clientBuilder) setClientMetadataJWKS(key jwk.Key) error {
	jwksBuf, err := json.Marshal(key)
	if err != nil {
		return err
	}
	b.idpClientMetadata.Jwks = map[string]interface{}{
		"keys": []json.RawMessage{json.RawMessage(jwksBuf)},
	}

	return nil
}

func (b *clientBuilder) setPrivateKeyJWTProperties() error {
	if len(b.jwks) == 0 && len(b.jwksURI) == 0 {
		return fmt.Errorf("public key is required for private_key_jwt token authentication method")
	}
	if len(b.jwks) != 0 {
		key, err := b.decodePublicKeyJWKS()
		if err != nil {
			return err
		}
		b.setClientMetadataJWKS(key)
	}
	b.idpClientMetadata.JwksURI = b.jwksURI
	return nil
}

func (b *clientBuilder) setTLSClientAuthProperties() error {
	if len(b.jwks) == 0 && len(b.jwksURI) == 0 {
		return fmt.Errorf("client certificate is required for tls_client_auth/self_signed_tls_client_auth token authentication method")
	}
	if len(b.jwks) != 0 {
		subjectDN, jwks, err := b.decodeCertificateJWKS()
		if err != nil {
			return err
		}
		b.setClientMetadataJWKS(jwks)

		switch b.certificateMetadata {
		case TLSClientAuthSanDNS:
			if b.tlsClientAuthSanDNS == "" {
				return fmt.Errorf("no value provided for tls_client_auth_san_dns")
			}
			b.idpClientMetadata.TLSClientAuthSanDNS = b.tlsClientAuthSanDNS
		case TLSClientAuthSanEmail:
			if b.tlsClientAuthSanEmail == "" {
				return fmt.Errorf("no value provided for tls_client_auth_san_email")
			}
			b.idpClientMetadata.TLSClientAuthSanEmail = b.tlsClientAuthSanEmail
		case TLSClientAuthSanIP:
			if b.tlsClientAuthSanIP == "" {
				return fmt.Errorf("no value provided for tls_client_auth_san_ip")
			}
			b.idpClientMetadata.TLSClientAuthSanIP = b.tlsClientAuthSanIP
		case TLSClientAuthSanURI:
			if b.tlsClientAuthSanURI == "" {
				return fmt.Errorf("no value provided for tls_client_auth_san_uri")
			}
			b.idpClientMetadata.TLSClientAuthSanURI = b.tlsClientAuthSanURI
		default:
			b.idpClientMetadata.TLSClientAuthSubjectDN = subjectDN
		}
	}
	b.idpClientMetadata.JwksURI = b.jwksURI
	return nil
}

func (b *clientBuilder) Build() (ClientMetadata, error) {
	responseTypes := make(map[string]string)
	for _, grantType := range b.idpClientMetadata.GrantTypes {
		if _, ok := grantTypeWithRedirects[grantType]; ok && len(b.idpClientMetadata.RedirectURIs) == 0 {
			return nil, fmt.Errorf("invalid client metadata redirect uri should be set for %s grant type", grantType)
		}
		switch grantType {
		case GrantTypeAuthorizationCode:
			responseTypes[AuthResponseCode] = AuthResponseCode
		case GrantTypeImplicit:
			responseTypes[AuthResponseToken] = AuthResponseToken
		}
	}
	b.idpClientMetadata.ResponseTypes = make([]string, 0)
	if len(responseTypes) > 0 {
		for responseTypes := range responseTypes {
			b.idpClientMetadata.ResponseTypes = append(b.idpClientMetadata.ResponseTypes, responseTypes)
		}
	}

	switch b.idpClientMetadata.GetTokenEndpointAuthMethod() {
	case config.PrivateKeyJWT:
		err := b.setPrivateKeyJWTProperties()
		if err != nil {
			return nil, err
		}
	case config.TLSClientAuth:
		fallthrough
	case config.SelfSignedTLSClientAuth:
		err := b.setTLSClientAuthProperties()
		if err != nil {
			return nil, err
		}
	}
	return b.idpClientMetadata, nil
}
