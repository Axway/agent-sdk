package oauth

import (
	"encoding/base64"
	"net/url"
)

type clientSecretBasicAuthenticator struct {
	clientID     string
	clientSecret string
	scope        string
}

func (p *clientSecretBasicAuthenticator) prepareRequest() (url.Values, map[string]string, error) {
	v := url.Values{
		metaGrantType: []string{grantClientCredentials},
	}

	if p.scope != "" {
		v.Add(metaScope, p.scope)
	}

	token := base64.StdEncoding.EncodeToString([]byte(p.clientID + ":" + p.clientSecret))
	headers := map[string]string{
		"Authorization": "Basic " + token,
	}
	return v, headers, nil
}
