package util

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockProxyServer struct {
	responseStatus int
	proxyAuth      []string
	server         *httptest.Server
}

func (m *mockProxyServer) handleReq(resp http.ResponseWriter, req *http.Request) {
	m.proxyAuth = req.Header["Proxy-Authorization"]
	resp.WriteHeader(m.responseStatus)
}

func newMockProxyServer() *mockProxyServer {
	proxyServer := &mockProxyServer{}
	proxyServer.server = httptest.NewServer(http.HandlerFunc(proxyServer.handleReq))
	return proxyServer
}

func TestProxyDial(t *testing.T) {
	proxyURL, _ := url.Parse("http://localhost:8888")
	dialer := NewDialer(proxyURL, nil)
	conn, err := dialer.DialContext(context.Background(), "tcp", "testtarget")
	assert.Nil(t, conn)
	assert.NotNil(t, err)

	proxyServer := newMockProxyServer()
	proxyURL, _ = url.Parse(proxyServer.server.URL)
	dialer = NewDialer(proxyURL, nil)
	proxyServer.responseStatus = 200
	conn, err = dialer.DialContext(context.Background(), "tcp", "testtarget")
	assert.NotNil(t, conn)
	assert.Nil(t, err)
	assert.Nil(t, proxyServer.proxyAuth)

	proxyServer.responseStatus = 407
	conn, err = dialer.DialContext(context.Background(), "tcp", "testtarget")
	assert.Nil(t, conn)
	assert.NotNil(t, err)
	assert.Nil(t, proxyServer.proxyAuth)

	proxyServer.responseStatus = 200
	proxyAuthURL, _ := url.Parse("http://foo:bar@" + proxyURL.Host)
	dialer = NewDialer(proxyAuthURL, nil)
	conn, err = dialer.DialContext(context.Background(), "tcp", "testtarget")
	assert.NotNil(t, conn)
	assert.Nil(t, err)
	assert.NotNil(t, proxyServer.proxyAuth)
	assert.Equal(t, proxyServer.proxyAuth[0], "Basic "+base64.StdEncoding.EncodeToString([]byte("foo:bar")))
}
