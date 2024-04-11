package traceability

import (
	"context"
	"fmt"
	"testing"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/stretchr/testify/assert"
)

type mockTransportClient struct {
	connectErr bool
}

func (m mockTransportClient) Close() error {
	return nil
}

func (m mockTransportClient) Publish(context.Context, publisher.Batch) error {
	return nil
}

func (m mockTransportClient) String() string {
	return ""
}

func (m mockTransportClient) Connect() error {
	if m.connectErr {
		return fmt.Errorf("error")
	}
	return nil
}

func TestExecute(t *testing.T) {
	job := newTraceabilityHealthCheckJob()
	// hc not okay
	err := job.Execute()
	assert.Nil(t, err)
}

func TestReady(t *testing.T) {
	agent.InitializeForTest(&apic.ServiceClient{})
	job := newTraceabilityHealthCheckJob()

	// hc not okay
	client := &mockTransportClient{connectErr: true}
	addClient(&Client{transportClient: client})
	ready := job.Ready()
	assert.False(t, ready)

	// hc okay
	client.connectErr = false
	ready = job.Ready()
	assert.True(t, ready)
}

func TestStatus(t *testing.T) {
	job := newTraceabilityHealthCheckJob()

	// no previous errors, status ok
	err := job.Status()
	assert.Nil(t, err)

	// previous errors, status not ok
	job.prevErr = fmt.Errorf("")
	err = job.Status()
	assert.Nil(t, err)
}

func TestHealthCheck(t *testing.T) {
	agent.InitializeForTest(&apic.ServiceClient{})
	job := newTraceabilityHealthCheckJob()
	client := &mockTransportClient{connectErr: true}
	addClient(&Client{transportClient: client})

	testCases := map[string]struct {
		isReady    bool
		expRes     hc.StatusLevel
		prevErr    error
		expDetails string
	}{
		"success when read and no error": {
			isReady: true,
			expRes:  hc.OK,
		},
		"expect error when not ready": {
			expRes:     hc.FAIL,
			expDetails: "agent not connected to traceability yet",
		},
		"expect error when previous error": {
			expRes:     hc.FAIL,
			isReady:    true,
			prevErr:    fmt.Errorf("error"),
			expDetails: "connection error: name Failed. error",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			job.prevErr = tc.prevErr
			job.ready = tc.isReady

			status := job.healthcheck("name")
			assert.NotNil(t, status)
			assert.Equal(t, tc.expRes, status.Result)
			assert.Equal(t, tc.expDetails, status.Details)
		})
	}
}

// func TestJobStatus(t *testing.T) {
// 	os.Setenv("HTTP_CLIENT_TIMEOUT", "1s")

// 	testCases := []struct {
// 		name     string
// 		protocol string
// 		proxy    string
// 		errStr   string
// 	}{
// 		{
// 			name:     "TCP no Proxy",
// 			protocol: "tcp",
// 			proxy:    "",
// 			errStr:   "connection failed",
// 		},
// 		{
// 			name:     "TCP bad Proxy URL",
// 			protocol: "tcp",
// 			proxy:    "socks5://host:\\//localhost:1080",
// 			errStr:   "proxy could not be parsed",
// 		},
// 		{
// 			name:     "TCP bad Proxy Protocol",
// 			protocol: "tcp",
// 			proxy:    "sock://localhost:1080",
// 			errStr:   "could not setup proxy",
// 		},
// 		{
// 			name:     "TCP good Proxy",
// 			protocol: "tcp",
// 			proxy:    "socks5://localhost:1080",
// 			errStr:   "connection failed",
// 		},
// 		{
// 			name:     "HTTPS no Proxy",
// 			protocol: "https",
// 			proxy:    "",
// 			errStr:   "connection failed",
// 		},
// 	}

// 	for _, test := range testCases {
// 		t.Run(test.name, func(t *testing.T) {
// 			traceCfg = &Config{
// 				Proxy: ProxyConfig{
// 					URL: test.proxy,
// 				},
// 			}
// 			job := condorHealthCheckJob{
// 				protocol: test.protocol,
// 				host:     "somehost.com:543",
// 				proxyURL: test.proxy,
// 				timeout:  time.Second * 1,
// 			}

// 			err := job.Status()
// 			assert.NotNil(t, err)
// 			assert.Contains(t, err.Error(), test.errStr)
// 		})
// 	}
// }
