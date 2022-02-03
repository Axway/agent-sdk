package log

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

type Request struct {
	User      UserInfo
	UserAgent string
	Host      string
	Tls       string
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
		Tls:       "-----BEGIN-----ABCDEFGHIJKLMNOPQRSTUVWXYZ-----END-----",
	}

	fmt.Println(fmt.Sprintf("%s", ObscureArguments([]string{"ClientPass", "Tls"}, request)))

	expected := "[{\"User\":{\"ClientName\":\"John Doe\",\"ClientCode\":32156,\"ClientPass\": \"[redacted]\"},\"UserAgent\":\"Version 17.2.0.12 - Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/535.8 (KHTML, like Gecko) Beamrise/17.2.0.12 Chrome/17.0.939.0 Safari/535.8\",\"Host\":\"http://example.com\",\"Tls\": \"[redacted]\"}]"
	assert.Equal(t, expected, fmt.Sprintf("%s", ObscureArguments([]string{"ClientPass", "Tls"}, request)))
}

func TestObscureNumbers(t *testing.T) {

	user := UserInfo{ClientName: "John Doe", ClientCode: 32156, ClientPass: "GHSGD&#&BGL˜X"}
	request := Request{
		User:      user,
		UserAgent: "Version 17.2.0.12 - Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/535.8 (KHTML, like Gecko) Beamrise/17.2.0.12 Chrome/17.0.939.0 Safari/535.8",
		Host:      "http://example.com",
		Tls:       "-----BEGIN-----ABCDEFGHIJKLMNOPQRSTUVWXYZ-----END-----",
	}

	fmt.Println(fmt.Sprintf("%s", ObscureArguments([]string{"ClientCode"}, request)))

	expected := "[{\"User\":{\"ClientName\":\"John Doe\",\"ClientCode\": \"[redacted]\",\"ClientPass\":\"GHSGD\\u0026#\\u0026BGL˜X\"},\"UserAgent\":\"Version 17.2.0.12 - Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/535.8 (KHTML, like Gecko) Beamrise/17.2.0.12 Chrome/17.0.939.0 Safari/535.8\",\"Host\":\"http://example.com\",\"Tls\":\"-----BEGIN-----ABCDEFGHIJKLMNOPQRSTUVWXYZ-----END-----\"}]"
	assert.Equal(t, expected, fmt.Sprintf("%s", ObscureArguments([]string{"ClientCode"}, request)))
}
