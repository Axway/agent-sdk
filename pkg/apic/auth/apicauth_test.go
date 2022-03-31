package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestResponseDecode(t *testing.T) {
	fixture := `
{
  "access_token": "some_value",
  "expires_in": 1800,
  "refresh_expires_in": 21600,
  "refresh_token": "some_value",
  "token_type": "bearer",
  "not-before-policy": 1510148785,
  "session_state": "f4b0fe58-a6f7-4452-9010-3945a7ecd493"
}`

	tokens := &axwayTokenResponse{}
	json.Unmarshal([]byte(fixture), tokens)
	if tokens.AccessToken != "some_value" {
		t.Error("unexpected access token value")
	}
	if tokens.ExpiresIn != 1800 {
		t.Error("unexpected expires in token")
	}
}

func TestEmptyTokenHolder(t *testing.T) {
	th := &tokenHolder{}

	if token := th.getCachedToken(); token != "" {
		t.Error("unexpected token from cache")
	}
}

func TestExpiredTokenHolder(t *testing.T) {
	th := &tokenHolder{
		tokens: &axwayTokenResponse{
			AccessToken: "some_token",
		},
		expiry: time.NewTimer(0),
	}

	time.Sleep(time.Millisecond)

	if token := th.getCachedToken(); token != "" {
		t.Error("unexpected token from cache")
	}
}

func TestGetPlatformTokensHttpError(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusForbidden)
	}))
	defer s.Close()

	ptg := platformTokenGenerator{s.URL, time.Millisecond, nil, "", "", nil}

	_, err := ptg.getPlatformTokens("some_token")
	if err == nil {
		t.Error("Expected error on token call")
	}
}

func TestGetPlatformTokensTimeout(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
	}))

	defer s.Close()
	ptg := platformTokenGenerator{s.URL, time.Second, nil, "", "", nil}

	_, err := ptg.getPlatformTokens("some_token")
	if err == nil {
		t.Error("Expected error on token call")
	}
}

func TestChannelTokenGetter(t *testing.T) {
	token := "some_token"
	tg := staticTokenGetter(token)

	ctg := tokenGetterWithChannel(tg)
	defer ctg.Close()

	for i := 0; i < 10; i++ {
		returnToken, err := ctg.GetToken()

		if err != nil {
			t.Error("Unexpected error ", err)
		}
		if token != returnToken {
			t.Errorf("Expected %s, got %s", token, returnToken)
		}
	}
}

func TestChannelTokenGetterPropagatesError(t *testing.T) {
	err := fmt.Errorf("some error")
	tg := funcTokenGetter(func() (string, error) { return "", err })
	ctg := tokenGetterWithChannel(tg)
	_, gotErr := ctg.GetToken()
	if err != gotErr {
		t.Errorf("Expected error %s, got %s", err, gotErr)
	}
}

func TestChannelTokenGetterCloses(t *testing.T) {
	tg := funcTokenGetter(func() (string, error) { return "", nil })
	ctg := tokenGetterWithChannel(tg)
	ctg.Close()

	if _, ok := <-ctg.responses; ok {
		t.Error("Expected responses channels to be closed")
	}
}

func TestGetKey(t *testing.T) {
	cases := []struct {
		description string
		kr          *keyReader
	}{
		{
			"no password",
			&keyReader{
				privKey: "testdata/private_key.pem",
			},
		},
		{
			"with empty password file",
			&keyReader{
				privKey:  "testdata/private_key.pem",
				password: "testdata/password_empty",
			},
		},
		{
			"with password",
			&keyReader{
				privKey:  "testdata/private_key_with_pwd.pem",
				password: "testdata/password",
			},
		},
	}

	for _, testCase := range cases {
		if _, err := testCase.kr.getPrivateKey(); err != nil {
			t.Errorf("testcase: %s: failed to read rsa key %s", testCase.description, err)
		}
	}
}

func TestGetPublicKey(t *testing.T) {
	cases := []struct {
		description string
		kr          *keyReader
	}{
		{
			"with public key",
			&keyReader{
				publicKey: "testdata/public_key",
			},
		},
		{
			"with private and public key",
			&keyReader{
				privKey:   "testdata/private_key.pem",
				publicKey: "testdata/public_key",
			},
		},
		{
			"with private, public key, and password",
			&keyReader{
				privKey:   "testdata/private_key_with_pwd.pem",
				password:  "testdata/password",
				publicKey: "testdata/public_key",
			},
		},
	}
	for _, testCase := range cases {
		if _, err := testCase.kr.getPublicKey(); err != nil {
			t.Errorf("testcase: %s: failed to read public key %s", testCase.description, err)
		}
	}
}

func TestComputeKIDFromDER(t *testing.T) {
	key, err := ioutil.ReadFile("testdata/public_key")
	if err != nil {
		t.Errorf("unable to read public_key")
	}
	res, err := computeKIDFromDER(key)
	if err != nil {
		t.Errorf("unable to compute kid")
	}
	expected := "1wzYoslzjo-ROTN1CUWPQYtTUqrqiaDO96fAAmb7JvA"
	if res != expected {
		t.Fail()
	}

	// der file format
	key, err = ioutil.ReadFile("testdata/public_key.der")
	if err != nil {
		t.Errorf("unable to read public_key.der")
	}
	res, err = computeKIDFromDER(key)
	if err != nil {
		t.Errorf("unable to compute kid")
	}
	expected = "iXcfstYFMANhYzgPwMWJxIQdfLQBqWjdiwyl7e4xv6Q"
	if res != expected {
		t.Fail()
	}
}

func TestNetAuthenticate(t *testing.T) {
	aa := NewWithStatic("12345", "abcde")
	if aa == nil {
		t.Errorf("unable to create ApicAuth")
	}
	if aa.tenantID != "12345" {
		t.Fail()
	}

	token, err := aa.GetToken()
	if err != nil {
		t.Errorf("error getting token")
	}
	if token != "abcde" {
		t.Fail()
	}
	err = aa.AuthenticateNet(&http.Request{
		Header: make(map[string][]string),
	})
	if err != nil {
		t.Errorf("error from AuthenticateNet")
	}
}
