package registration

// ClientBuilder - Builder for IdP client representation
type ClientBuilder interface {
	SetClientName(string) ClientBuilder
	SetScopes([]string) ClientBuilder
	SetGrantTypes([]string) ClientBuilder
	SetResponseType([]string) ClientBuilder
	SetTokenEndpointAuthMethod(tokenAuthMethod string) ClientBuilder
	SetRedirectURIs([]string) ClientBuilder
	SetLogoURI(string) ClientBuilder
	Build() Client
}

type clientBuilder struct {
	idpClient *client
}

// NewClientBuilder -  create a new instance of client builder
func NewClientBuilder() ClientBuilder {
	return &clientBuilder{
		idpClient: &client{},
	}
}

func (b *clientBuilder) SetClientName(name string) ClientBuilder {
	b.idpClient.ClientName = name
	return b
}

func (b *clientBuilder) SetScopes(scopes []string) ClientBuilder {
	b.idpClient.Scope = Scopes(scopes)
	return b
}

func (b *clientBuilder) SetGrantTypes(grantTypes []string) ClientBuilder {
	b.idpClient.GrantTypes = grantTypes
	return b
}

func (b *clientBuilder) SetResponseType(responseTypes []string) ClientBuilder {
	b.idpClient.ResponseTypes = responseTypes
	return b
}

func (b *clientBuilder) SetTokenEndpointAuthMethod(tokenAuthMethod string) ClientBuilder {
	b.idpClient.TokenEndpointAuthMethod = tokenAuthMethod
	return b
}

func (b *clientBuilder) SetRedirectURIs(redirectURIs []string) ClientBuilder {
	b.idpClient.RedirectURIs = redirectURIs
	return b
}

func (b *clientBuilder) SetLogoURI(logoURI string) ClientBuilder {
	b.idpClient.LogoURI = logoURI
	return b
}

func (b *clientBuilder) Build() Client {
	return b.idpClient
}
