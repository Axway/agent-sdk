package oauth

import (
	"net/url"
)

type clientSecretPostAuthenticator struct {
	clientID     string
	clientSecret string
	scope        string
}

func (p *clientSecretPostAuthenticator) prepareRequest() (url.Values, map[string]string, error) {
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
	return v, nil, nil
}
