package auth

import (
	"fmt"
	"net/http"
	"testing"
)

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

func TestNetAuthenticate(t *testing.T) {
	aa := NewWithStatic("12345", "abcde")
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
