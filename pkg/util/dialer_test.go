package util

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockTCPServer struct {
	listener net.Listener
}

func newMockTCPServer() (*mockTCPServer, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, err
	}

	server := &mockTCPServer{
		listener: l,
	}
	return server, nil
}

func (s *mockTCPServer) getAddr() string {
	if s.listener != nil {
		return s.listener.Addr().(*net.TCPAddr).String()
	}
	return ""
}

func (s *mockTCPServer) getIP() string {
	if s.listener != nil {
		return s.listener.Addr().(*net.TCPAddr).IP.String()
	}
	return ""
}

func (s *mockTCPServer) getPort() int {
	if s.listener != nil {
		return s.listener.Addr().(*net.TCPAddr).Port
	}
	return 0
}

func (s *mockTCPServer) close() {
	if s.listener != nil {
		s.listener.Close()
		s.listener = nil
	}
}

type mocHTTPServer struct {
	responseStatus  int
	proxyAuth       []string
	server          *httptest.Server
	requestReceived bool
}

func (m *mocHTTPServer) handleReq(resp http.ResponseWriter, req *http.Request) {
	m.requestReceived = true
	proxyAuth, ok := req.Header["Proxy-Authorization"]
	if ok {
		m.proxyAuth = proxyAuth
		resp.WriteHeader(m.responseStatus)
	}
	resp.WriteHeader(m.responseStatus)
}

func newMockHTTPServer() *mocHTTPServer {
	mockServer := &mocHTTPServer{}
	mockServer.server = httptest.NewServer(http.HandlerFunc(mockServer.handleReq))
	return mockServer
}

func TestProxyDial(t *testing.T) {
	proxyURL, _ := url.Parse("http://localhost:8888")
	dialer := NewDialer(proxyURL, nil)
	conn, err := dialer.DialContext(context.Background(), "tcp", "testtarget")
	assert.Nil(t, conn)
	assert.NotNil(t, err)

	proxyServer := newMockHTTPServer()
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

func TestSingleEntryDial(t *testing.T) {
	targetServer, _ := newMockTCPServer()
	defer targetServer.close()
	singleEntryServer, _ := newMockTCPServer()
	defer singleEntryServer.close()

	// No proxy, no single entry, validate connection directly to target server
	targetServerURL, _ := url.Parse(fmt.Sprintf("https://%s", targetServer.getAddr()))
	singleHostMapping := map[string]string{}
	dialer := NewDialer(nil, singleHostMapping)
	conn, err := dialer.Dial("tcp", targetServerURL.Host)
	assert.NotNil(t, conn)
	assert.Nil(t, err)

	assert.Equal(t, targetServer.getIP(), conn.RemoteAddr().(*net.TCPAddr).IP.String())
	assert.Equal(t, targetServer.getPort(), conn.RemoteAddr().(*net.TCPAddr).Port)
	assert.NotEqual(t, singleEntryServer.getPort(), conn.RemoteAddr().(*net.TCPAddr).Port)

	// No proxy, single entry configured to match target, validate connection to single entry
	singleHostMapping = map[string]string{
		targetServer.getAddr(): singleEntryServer.getAddr(),
	}
	dialer = NewDialer(nil, singleHostMapping)
	conn, err = dialer.Dial("tcp", targetServerURL.Host)
	assert.NotNil(t, conn)
	assert.Nil(t, err)

	assert.Equal(t, targetServer.getIP(), conn.RemoteAddr().(*net.TCPAddr).IP.String())
	assert.NotEqual(t, targetServer.getPort(), conn.RemoteAddr().(*net.TCPAddr).Port)
	assert.Equal(t, singleEntryServer.getPort(), conn.RemoteAddr().(*net.TCPAddr).Port)

	// Proxy configured, single entry configured to match target, validate connection to proxy
	proxyServer := newMockHTTPServer()
	proxyURL, _ := url.Parse(proxyServer.server.URL)
	dialer = NewDialer(proxyURL, singleHostMapping)
	proxyServer.responseStatus = 200
	conn, err = dialer.Dial("tcp", targetServerURL.Host)
	assert.NotNil(t, conn)
	assert.Nil(t, err)

	assert.Equal(t, targetServer.getIP(), conn.RemoteAddr().(*net.TCPAddr).IP.String())
	assert.NotEqual(t, targetServer.getPort(), conn.RemoteAddr().(*net.TCPAddr).Port)
	assert.NotEqual(t, singleEntryServer.getPort(), conn.RemoteAddr().(*net.TCPAddr).Port)
	assert.Equal(t, ParsePort(proxyURL), conn.RemoteAddr().(*net.TCPAddr).Port)
	assert.Equal(t, true, proxyServer.requestReceived)

	// Invalid proxy configured
	proxyURL, _ = url.Parse("socks5://test:test@localhost:0")
	dialer = NewDialer(proxyURL, singleHostMapping)
	conn, err = dialer.Dial("tcp", targetServerURL.Host)
	assert.Nil(t, conn)
	assert.NotNil(t, err)

	// Invalid proxy scheme
	proxyURL, _ = url.Parse("noscheme://localhost:0")
	dialer = NewDialer(proxyURL, singleHostMapping)
	conn, err = dialer.Dial("tcp", targetServerURL.Host)
	assert.Nil(t, conn)
	assert.NotNil(t, err)
}
