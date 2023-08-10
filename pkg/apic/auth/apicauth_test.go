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
