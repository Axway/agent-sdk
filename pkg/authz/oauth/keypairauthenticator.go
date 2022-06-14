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
	aud        string
	privateKey *rsa.PrivateKey
	publicKey  []byte
}

// prepareInitialToken prepares a token for an access request
func (p *keyPairAuthenticator) prepareInitialToken(privateKey interface{}, kid, clientID, aud string) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.StandardClaims{
		Issuer:    fmt.Sprintf("%s:%s", assertionTypeJWT, clientID),
		Subject:   clientID,
		Audience:  aud,
		ExpiresAt: now.Add(60*time.Second).UnixNano() / 1e9,
		IssuedAt:  now.UnixNano() / 1e9,
		Id:        uuid.New().String(),
	})

	token.Header["kid"] = kid

	requestToken, err := token.SignedString(privateKey)
	if err != nil {
		return "", err
	}

	return requestToken, nil
}

func (p *keyPairAuthenticator) prepareRequest() (url.Values, error) {
	kid, err := util.ComputeKIDFromDER(p.publicKey)
	if err != nil {
		return nil, err
	}

	requestToken, err := p.prepareInitialToken(p.privateKey, kid, p.clientID, p.aud)
	if err != nil {
		return nil, err
	}

	return url.Values{
		metaGrantType:           []string{grantClientCredentials},
		metaClientAssertionType: []string{assertionTypeJWT},
		metaClientAssertion:     []string{requestToken},
	}, nil
}
