package idp

import (
	"strings"
)

type credData struct {
	clientID                string
	clientSecret            string
	scopes                  []string
	grantTypes              []string
	tokenAuthMethod         string
	responseTypes           []string
	redirectURLs            []string
	jwksURI                 string
	publicKey               string
	certificate             string
	certificateMetadata     string
	tlsClientAuthSanDNS     string
	tlsClientAuthSanEmail   string
	tlsClientAuthSanIP      string
	tlsClientAuthSanURI     string
	registrationAccessToken string
}

// GetClientID - returns client ID
func (c *credData) GetClientID() string {
	return c.clientID
}

// GetClientSecret - returns client secret
func (c *credData) GetClientSecret() string {
	return c.clientSecret
}

// GetScopes - returns client scopes
func (c *credData) GetScopes() []string {
	return c.scopes
}

// GetGrantTypes - returns grant types
func (c *credData) GetGrantTypes() []string {
	return c.grantTypes
}

// GetTokenEndpointAuthMethod - returns token auth method
func (c *credData) GetTokenEndpointAuthMethod() string {
	return c.tokenAuthMethod
}

// GetResponseTypes - returns token response type
func (c *credData) GetResponseTypes() []string {
	return c.responseTypes
}

// GetRedirectURIs - Returns redirect urls
func (c *credData) GetRedirectURIs() []string {
	return c.redirectURLs
}

// GetJwksURI - returns JWKS uri
func (c *credData) GetJwksURI() string {
	return c.jwksURI
}

// GetPublicKey - returns the public key
func (c *credData) GetPublicKey() string {
	return c.publicKey
}

// GetCertificate - returns the certificate
func (c *credData) GetCertificate() string {
	return c.certificate
}

// GetCertificateMetadata - returns the certificate metadata property
func (c *credData) GetCertificateMetadata() string {
	return c.certificateMetadata
}

// GetTLSClientAuthSanDNS - returns the value for tls_client_auth_san_dns
func (c *credData) GetTLSClientAuthSanDNS() string {
	return c.tlsClientAuthSanDNS
}

// GetTLSClientAuthSanDNS - returns the value for tls_client_auth_san_dns
func (c *credData) GetTLSClientAuthSanEmail() string {
	return c.tlsClientAuthSanEmail
}

// GetTLSClientAuthSanIP - returns the value for tls_client_auth_san_ip
func (c *credData) GetTLSClientAuthSanIP() string {
	return c.tlsClientAuthSanIP
}

// GetTLSClientAuthSanURI - returns the value for tls_client_auth_san_uri
func (c *credData) GetTLSClientAuthSanURI() string {
	return c.tlsClientAuthSanURI
}

func formattedJWKS(jwks string) string {
	formattedJWKS := strings.ReplaceAll(jwks, "----- ", "-----\n")
	return strings.ReplaceAll(formattedJWKS, " -----", "\n-----")
}
