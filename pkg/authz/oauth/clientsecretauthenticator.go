package oauth

import "net/url"

type clientSecretAuthenticator struct {
	clientID     string
	clientSecret string
}

func (p *clientSecretAuthenticator) prepareRequest() (url.Values, error) {
	return url.Values{
		metaGrantType:    []string{grantClientCredentials},
		metaClientID:     []string{p.clientID},
		metaClientSecret: []string{p.clientSecret},
	}, nil
}
