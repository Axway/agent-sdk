package oauth

import "net/url"

type clientSecretAuthenticator struct {
	clientID     string
	clientSecret string
	scope        string
}

func (p *clientSecretAuthenticator) prepareRequest() (url.Values, error) {
	v := url.Values{
		metaGrantType: []string{grantClientCredentials},
		metaClientID:  []string{p.clientID},
	}

	if p.clientSecret != "" {
		v.Add(metaClientSecret, p.clientSecret)
	}

	if p.scope != "" {
		v.Add(metaScope, p.scope)
	}
	return v, nil
}
