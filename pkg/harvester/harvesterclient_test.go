package harvester

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

type mockHTTPServer struct {
	harvesterResponse interface{}
	responseStatus    int

	server *httptest.Server
}

func newMockHTTPServer() *mockHTTPServer {
	mockServer := &mockHTTPServer{}
	mockServer.server = httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if mockServer.responseStatus == 0 {
			mockServer.responseStatus = 200
		}

		resp.WriteHeader(mockServer.responseStatus)
		if mockServer.responseStatus >= 400 {
			resp.WriteHeader(mockServer.responseStatus)
			return
		}
		body, _ := json.Marshal(mockServer.harvesterResponse)
		resp.Write(body)
	}))
	return mockServer
}

func TestReceiveSyncEvents(t *testing.T) {
	s := newMockHTTPServer()
	defer s.server.Close()
	mockServerURL, _ := url.Parse(s.server.URL)
	port, _ := strconv.Atoi(mockServerURL.Port())
	cfg := &Config{
		Protocol: mockServerURL.Scheme,
		Host:     mockServerURL.Hostname(),
		Port:     uint32(port),
		TenantID: "12345",
		TokenGetter: func() (string, error) {
			return "", nil
		},
		PageSize: 2,
	}
	client := NewClient(cfg)

	eventCh := make(chan *proto.Event, 1)
	stopCh := make(chan bool)
	events := make([]*proto.Event, 0)
	go func() {
		for {
			select {
			case <-stopCh:
				return
			case event := <-eventCh:
				events = append(events, event)
			}
		}
	}()
	s.responseStatus = 200
	s.harvesterResponse = []resourceEntryExternalEvent{}
	_, err := client.ReceiveSyncEvents(context.Background(), "/test", 1, eventCh)
	assert.Nil(t, err)
	stopCh <- true
	assert.Equal(t, 0, len(events))
}
