package oauth

import (
	"crypto/rsa"
	"fmt"
	"net/url"
	"time"

	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

type keyPairAuthenticator struct {
	clientID   string
	issuer     string
	aud        string
	privateKey *rsa.PrivateKey
	publicKey  []byte
	scope      string
}

// prepareInitialToken prepares a token for an access request
func (p *keyPairAuthenticator) prepareInitialToken(kid string) (string, error) {
	now := time.Now()
	if p.issuer == "" {
		p.issuer = fmt.Sprintf("%s:%s", assertionTypeJWT, p.clientID)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.StandardClaims{
		Issuer:    p.issuer,
		Subject:   p.clientID,
		Audience:  p.aud,
		ExpiresAt: now.Add(60*time.Second).UnixNano() / 1e9,
		IssuedAt:  now.UnixNano() / 1e9,
		Id:        uuid.New().String(),
	})

	token.Header["kid"] = kid

	requestToken, err := token.SignedString(p.privateKey)
	if err != nil {
		return "", err
	}

	return requestToken, nil
}

func (p *keyPairAuthenticator) prepareRequest() (url.Values, map[string]string, error) {
	kid, err := util.ComputeKIDFromDER(p.publicKey)
	if err != nil {
		return nil, nil, err
	}

	requestToken, err := p.prepareInitialToken(kid)
	if err != nil {
		return nil, nil, err
	}

	v := url.Values{
		metaGrantType:           []string{grantClientCredentials},
		metaClientAssertionType: []string{assertionTypeJWT},
		metaClientAssertion:     []string{requestToken},
	}

	if p.scope != "" {
		v.Add(metaScope, p.scope)
	}
	return v, nil, nil
}
