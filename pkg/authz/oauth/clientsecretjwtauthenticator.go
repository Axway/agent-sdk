package oauth

import (
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type clientSecretJwtAuthenticator struct {
	clientID      string
	clientSecret  string
	scope         string
	issuer        string
	aud           string
	signingMethod string
}

// prepareInitialToken prepares a token for an access request
func (p *clientSecretJwtAuthenticator) prepareInitialToken() (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(getSigningMethod(p.signingMethod, jwt.SigningMethodHS256), jwt.RegisteredClaims{
		Issuer:    p.issuer,
		Subject:   p.clientID,
		Audience:  []string{p.aud},
		ExpiresAt: jwt.NewNumericDate(now.Add(60 * time.Second)),
		IssuedAt:  jwt.NewNumericDate(now),
		ID:        uuid.New().String(),
	})

	requestToken, err := token.SignedString([]byte(p.clientSecret))
	if err != nil {
		return "", err
	}

	return requestToken, nil
}

func (p *clientSecretJwtAuthenticator) prepareRequest() (url.Values, map[string]string, error) {
	requestToken, err := p.prepareInitialToken()
	if err != nil {
		return nil, nil, err
	}

	v := url.Values{
		metaGrantType:           []string{GrantTypeClientCredentials},
		metaClientID:            []string{p.clientID},
		metaClientAssertionType: []string{assertionTypeJWT},
		metaClientAssertion:     []string{requestToken},
	}

	if p.scope != "" {
		v.Add(metaScope, p.scope)
	}
	return v, nil, nil
}
