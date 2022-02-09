package log

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Request struct {
	User      UserInfo
	UserAgent string
	Host      string
	TLS       string `json:"Tls"`
}

type UserInfo struct {
	ClientName string
	ClientCode int
	ClientPass string
}

func TestObscureComplexStrings(t *testing.T) {

	user := UserInfo{ClientName: "John Doe", ClientCode: 32156, ClientPass: "GHSGD&#&BGL˜X"}
	request := Request{
		User:      user,
		UserAgent: "Version 17.2.0.12 - Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/535.8 (KHTML, like Gecko) Beamrise/17.2.0.12 Chrome/17.0.939.0 Safari/535.8",
		Host:      "http://example.com",
		TLS:       "-----BEGIN-----ABCDEFGHIJKLMNOPQRSTUVWXYZ-----END-----",
	}

	fmt.Printf("%s", ObscureArguments([]string{"ClientPass", "Tls"}, request))

	expected := "[{\"User\":{\"ClientName\":\"John Doe\",\"ClientCode\":32156,\"ClientPass\": \"[redacted]\"},\"UserAgent\":\"Version 17.2.0.12 - Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/535.8 (KHTML, like Gecko) Beamrise/17.2.0.12 Chrome/17.0.939.0 Safari/535.8\",\"Host\":\"http://example.com\",\"Tls\": \"[redacted]\"}]"
	assert.Equal(t, expected, fmt.Sprintf("%s", ObscureArguments([]string{"ClientPass", "Tls"}, request)))
}

func TestObscureNumbers(t *testing.T) {

	user := UserInfo{ClientName: "John Doe", ClientCode: 32156, ClientPass: "GHSGD&#&BGL˜X"}
	request := Request{
		User:      user,
		UserAgent: "Version 17.2.0.12 - Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/535.8 (KHTML, like Gecko) Beamrise/17.2.0.12 Chrome/17.0.939.0 Safari/535.8",
		Host:      "http://example.com",
		TLS:       "-----BEGIN-----ABCDEFGHIJKLMNOPQRSTUVWXYZ-----END-----",
	}

	fmt.Printf("%s", ObscureArguments([]string{"ClientCode"}, request))

	expected := "[{\"User\":{\"ClientName\":\"John Doe\",\"ClientCode\": \"[redacted]\",\"ClientPass\":\"GHSGD\\u0026#\\u0026BGL˜X\"},\"UserAgent\":\"Version 17.2.0.12 - Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/535.8 (KHTML, like Gecko) Beamrise/17.2.0.12 Chrome/17.0.939.0 Safari/535.8\",\"Host\":\"http://example.com\",\"Tls\":\"-----BEGIN-----ABCDEFGHIJKLMNOPQRSTUVWXYZ-----END-----\"}]"
	assert.Equal(t, expected, fmt.Sprintf("%s", ObscureArguments([]string{"ClientCode"}, request)))
}
