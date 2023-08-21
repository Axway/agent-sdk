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

const (
	SigningMethodRS256 = "RS256"
	SigningMethodRS384 = "RS384"
	SigningMethodRS512 = "RS512"

	SigningMethodES256 = "ES256"
	SigningMethodES384 = "ES384"
	SigningMethodES512 = "ES512"

	SigningMethodPS256 = "PS256"
	SigningMethodPS384 = "PS384"
	SigningMethodPS512 = "PS512"

	SigningMethodHS256 = "HS256"
	SigningMethodHS384 = "HS384"
	SigningMethodHS512 = "HS512"
)

var signingMethodMap = map[string]jwt.SigningMethod{
	SigningMethodRS256: jwt.SigningMethodRS256,
	SigningMethodRS384: jwt.SigningMethodRS384,
	SigningMethodRS512: jwt.SigningMethodRS512,
	SigningMethodES256: jwt.SigningMethodES256,
	SigningMethodES384: jwt.SigningMethodES384,
	SigningMethodES512: jwt.SigningMethodES512,
	SigningMethodPS256: jwt.SigningMethodPS256,
	SigningMethodPS384: jwt.SigningMethodPS384,
	SigningMethodPS512: jwt.SigningMethodPS512,
	SigningMethodHS256: jwt.SigningMethodHS256,
	SigningMethodHS384: jwt.SigningMethodHS384,
	SigningMethodHS512: jwt.SigningMethodHS512,
}

type keyPairAuthenticator struct {
	clientID      string
	issuer        string
	aud           string
	privateKey    *rsa.PrivateKey
	publicKey     []byte
	scope         string
	signingMethod string
}

func getSigningMethod(signingMethod string, defaultSigningMethod jwt.SigningMethod) jwt.SigningMethod {
	sm, ok := signingMethodMap[signingMethod]
	if !ok {
		return defaultSigningMethod
	}
	return sm
}

// prepareInitialToken prepares a token for an access request
func (p *keyPairAuthenticator) prepareInitialToken(kid string) (string, error) {
	now := time.Now()
	if p.issuer == "" {
		p.issuer = fmt.Sprintf("%s:%s", assertionTypeJWT, p.clientID)
	}
	token := jwt.NewWithClaims(getSigningMethod(p.signingMethod, jwt.SigningMethodRS256), jwt.StandardClaims{
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
		metaGrantType:           []string{GrantTypeClientCredentials},
		metaClientAssertionType: []string{assertionTypeJWT},
		metaClientAssertion:     []string{requestToken},
	}

	if p.scope != "" {
		v.Add(metaScope, p.scope)
	}
	return v, nil, nil
}
