package oauth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MockIDPServer - interface for mock IDP server
type MockIDPServer interface {
	GetMetadataURL() string
	GetIssuer() string
	GetTokenURL() string
	GetAuthEndpoint() string
	GetRegistrationEndpoint() string
	SetMetadataResponseCode(statusCode int)
	SetTokenResponse(accessToken string, expiry time.Duration, statusCode int)
	SetRegistrationResponseCode(statusCode int)
	SetUseRegistrationAccessToken(useRegistrationAccessToken bool)
	GetTokenRequestHeaders() http.Header
	GetTokenQueryParams() url.Values
	GetTokenRequestValues() url.Values
	GetRequestHeaders() http.Header
	GetQueryParams() url.Values
	Close()
}

type mockIDPServer struct {
	metadataResponseCode       int
	tokenResponseCode          int
	registerResponseCode       int
	useRegistrationAccessToken bool
	accessToken                string
	tokenExpiry                time.Duration
	serverMetadata             *AuthorizationServerMetadata
	server                     *httptest.Server
	tokenReqHeaders            http.Header
	tokenQueryParams           url.Values
	tokenReqValues             url.Values
	reqHeaders                 http.Header
	reqQueryParam              url.Values
}

// NewMockIDPServer - creates a new mock IDP server for tests
func NewMockIDPServer() MockIDPServer {
	m := &mockIDPServer{
		metadataResponseCode: http.StatusOK,
		tokenResponseCode:    http.StatusOK,
		registerResponseCode: http.StatusCreated,
	}

	m.server = httptest.NewServer(http.HandlerFunc(m.handleRequest))
	m.serverMetadata = &AuthorizationServerMetadata{
		Issuer:                m.GetIssuer(),
		TokenEndpoint:         m.GetTokenURL(),
		AuthorizationEndpoint: m.GetAuthEndpoint(),
		RegistrationEndpoint:  m.GetRegistrationEndpoint(),
	}
	return m
}

func (m *mockIDPServer) handleRequest(resp http.ResponseWriter, req *http.Request) {
	if strings.Contains(req.RequestURI, "/metadata") {
		defer func() {
			m.metadataResponseCode = http.StatusOK
		}()
		if m.metadataResponseCode != http.StatusOK {
			resp.WriteHeader(m.metadataResponseCode)

			return
		}
		buf, _ := json.Marshal(m.serverMetadata)
		resp.Write(buf)
	}
	if strings.Contains(req.RequestURI, "/token") {
		m.tokenReqHeaders = req.Header
		m.tokenQueryParams = req.URL.Query()
		m.tokenReqValues = nil
		reqBuf, _ := io.ReadAll(req.Body)
		if len(reqBuf) != 0 {
			fmt.Printf("%s\n", string(reqBuf))
			val, err := url.ParseQuery(string(reqBuf))
			if err == nil {
				m.tokenReqValues = val
			}
		}
		defer func() {
			m.tokenResponseCode = http.StatusOK
			m.accessToken = ""
			m.tokenExpiry = 0
		}()
		if m.tokenResponseCode != http.StatusOK {
			resp.WriteHeader(m.tokenResponseCode)
			return
		}

		now := time.Now()
		t := tokenResponse{
			AccessToken: m.accessToken,
			ExpiresIn:   now.Add(m.tokenExpiry).UnixNano() / 1e9,
		}
		buf, _ := json.Marshal(t)
		resp.Write(buf)
	}
	if strings.Contains(req.RequestURI, "/register") {
		m.reqHeaders = req.Header
		m.reqQueryParam = req.URL.Query()
		defer func() {
			m.registerResponseCode = http.StatusCreated
		}()
		if req.Method == http.MethodPost {
			if m.registerResponseCode != http.StatusCreated {
				resp.WriteHeader(m.registerResponseCode)
				return
			}
			resp.WriteHeader(http.StatusCreated)
			clientBuf, _ := io.ReadAll(req.Body)
			cl := &clientMetadata{}
			json.Unmarshal(clientBuf, cl)
			cl.ClientID = uuid.New().String()
			cl.ClientSecret = uuid.New().String()
			if m.useRegistrationAccessToken {
				cl.RegistrationAccessToken = uuid.New().String()
			}
			clientBuf, _ = json.Marshal(cl)
			resp.Write(clientBuf)
		}
		if req.Method == http.MethodDelete {
			if m.registerResponseCode != http.StatusNoContent {
				resp.WriteHeader(m.registerResponseCode)
				return
			}
			resp.WriteHeader(http.StatusNoContent)
		}
	}
}

func (m *mockIDPServer) GetMetadataURL() string {
	return m.server.URL + "/metadata"
}

func (m *mockIDPServer) GetIssuer() string {
	return m.server.URL
}

func (m *mockIDPServer) GetTokenURL() string {
	return m.server.URL + "/token"
}

func (m *mockIDPServer) GetAuthEndpoint() string {
	return m.server.URL + "/auth"
}

func (m *mockIDPServer) GetRegistrationEndpoint() string {
	return m.server.URL + "/register"
}

func (m *mockIDPServer) SetMetadataResponseCode(statusCode int) {
	m.metadataResponseCode = statusCode
}

func (m *mockIDPServer) SetTokenResponse(accessToken string, expiry time.Duration, statusCode int) {
	m.accessToken = accessToken
	m.tokenExpiry = expiry
	m.tokenResponseCode = statusCode
}

func (m *mockIDPServer) SetRegistrationResponseCode(statusCode int) {
	m.registerResponseCode = statusCode
}

func (m *mockIDPServer) SetUseRegistrationAccessToken(useRegistrationAccessToken bool) {
	m.useRegistrationAccessToken = useRegistrationAccessToken
}

func (m *mockIDPServer) GetTokenRequestHeaders() http.Header {
	return m.tokenReqHeaders
}

func (m *mockIDPServer) GetTokenQueryParams() url.Values {
	return m.tokenQueryParams
}

func (m *mockIDPServer) GetTokenRequestValues() url.Values {
	return m.tokenReqValues
}

func (m *mockIDPServer) GetRequestHeaders() http.Header {
	return m.reqHeaders
}

func (m *mockIDPServer) GetQueryParams() url.Values {
	return m.reqQueryParam
}
func (m *mockIDPServer) Close() {
	m.server.Close()
}
