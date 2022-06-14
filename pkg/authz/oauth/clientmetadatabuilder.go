package oauth

import (
	"encoding/json"
	"fmt"

	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/lestrrat-go/jwx/jwk"
)

var grantTypeWithRedirects = map[string]bool{grantAuthorizationCode: true, grantImplicit: true}

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
	SetExtraProperties(map[string]string) ClientBuilder

	Build() (ClientMetadata, error)
}

type clientBuilder struct {
	publicKey         []byte
	idpClientMetadata *clientMetadata
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
	b.idpClientMetadata.JwksURI = jwksURI
	return b
}

func (b *clientBuilder) SetJWKS(publicKey []byte) ClientBuilder {
	b.publicKey = publicKey
	return b
}

func (b *clientBuilder) SetExtraProperties(extraProperties map[string]string) ClientBuilder {
	b.idpClientMetadata.extraProperties = extraProperties
	return b
}

func (b *clientBuilder) decodeJWKS() ([]byte, error) {
	p, err := util.ParsePublicKey(b.publicKey)
	if err != nil {
		return nil, err
	}

	key, err := jwk.New(p)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %s", err)
	}
	kid, _ := util.ComputeKIDFromDER(b.publicKey)
	key.Set(jwk.KeyIDKey, kid)
	key.Set(jwk.KeyUsageKey, jwk.ForSignature)

	buf, err := json.MarshalIndent(key, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal jwks: %s", err)
	}
	return buf, nil
}

func (b *clientBuilder) Build() (ClientMetadata, error) {
	if b.publicKey != nil && len(b.publicKey) > 0 {
		jwksBuf, err := b.decodeJWKS()
		if err != nil {
			return nil, err
		}

		b.idpClientMetadata.Jwks = map[string]interface{}{
			"keys": []json.RawMessage{json.RawMessage(jwksBuf)},
		}
	}

	for _, grantType := range b.idpClientMetadata.GrantTypes {
		if _, ok := grantTypeWithRedirects[grantType]; ok && len(b.idpClientMetadata.RedirectURIs) == 0 {
			return nil, fmt.Errorf("invalid client metadata redirect uri should be set for %s grant type", grantType)
		}
	}
	return b.idpClientMetadata, nil
}
