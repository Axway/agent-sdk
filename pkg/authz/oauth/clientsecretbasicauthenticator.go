package oauth

import (
	"encoding/base64"
	"fmt"
	"net/url"
)

const (
	basicAuthHeaderTemplate = "Basic %s"
)

type clientSecretBasicAuthenticator struct {
	clientID     string
	clientSecret string
	scope        string
}

func (p *clientSecretBasicAuthenticator) prepareRequest() (url.Values, map[string]string, error) {
	v := url.Values{
		metaGrantType: []string{GrantTypeClientCredentials},
	}

	if p.scope != "" {
		v.Add(metaScope, p.scope)
	}

	token := base64.StdEncoding.EncodeToString([]byte(p.clientID + ":" + p.clientSecret))
	headers := map[string]string{
		hdrAuthorization: fmt.Sprintf(basicAuthHeaderTemplate, token),
	}
	return v, headers, nil
}
