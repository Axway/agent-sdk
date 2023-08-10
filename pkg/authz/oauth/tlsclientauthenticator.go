package oauth

import "net/url"

type tlsClientAuthenticator struct {
	clientID string
	scope    string
}

func (p *tlsClientAuthenticator) prepareRequest() (url.Values, map[string]string, error) {
	v := url.Values{
		metaGrantType: []string{grantClientCredentials},
		metaClientID:  []string{p.clientID},
	}

	if p.scope != "" {
		v.Add(metaScope, p.scope)
	}
	return v, nil, nil
}
