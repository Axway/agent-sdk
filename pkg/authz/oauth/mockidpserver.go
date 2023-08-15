package oauth

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
	SetMetadataResponseCode(statusCode int)
	SetTokenResponse(accessToken string, expiry time.Duration, statusCode int)
	SetRegistrationResponseCode(statusCode int)
	Close()
}

type mockIDPServer struct {
	metadataResponseCode int
	tokenResponseCode    int
	registerResponseCode int
	accessToken          string
	tokenExpiry          time.Duration
	serverMetadata       *AuthorizationServerMetadata
	server               *httptest.Server
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

func (m *mockIDPServer) Close() {
	m.server.Close()
}
