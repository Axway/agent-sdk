package traceability

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExecute(t *testing.T) {
	ta := &traceabilityAgentHealthChecker{}

	job := condorHealthCheckJob{
		agentHealthChecker: ta,
	}
	// hc not okay
	err := job.Execute()
	assert.Nil(t, err)
}

func TestReady(t *testing.T) {
	ta := &traceabilityAgentHealthChecker{
		protocol: "tcp",
		host:     "somehost.com:543",
		proxyURL: "",
		timeout:  time.Second * 1,
	}

	job := condorHealthCheckJob{
		agentHealthChecker: ta,
	}
	// hc not okay
	ready := job.Ready()
	assert.False(t, ready)
}

func TestJobStatus(t *testing.T) {
	os.Setenv("HTTP_CLIENT_TIMEOUT", "1s")

	testCases := []struct {
		name     string
		protocol string
		proxy    string
		errStr   string
	}{
		{
			name:     "TCP no Proxy",
			protocol: "tcp",
			proxy:    "",
			errStr:   "connection failed",
		},
		{
			name:     "TCP bad Proxy URL",
			protocol: "tcp",
			proxy:    "socks5://host:\\//test.com:1080",
			errStr:   "proxy could not be parsed",
		},
		{
			name:     "TCP bad Proxy Protocol",
			protocol: "tcp",
			proxy:    "sock://test.com:1080",
			errStr:   "could not setup proxy",
		},
		{
			name:     "TCP good Proxy",
			protocol: "tcp",
			proxy:    "socks5://test.com:1080",
			errStr:   "connection failed",
		},
		{
			name:     "HTTPS no Proxy",
			protocol: "https",
			proxy:    "",
			errStr:   "connection failed",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ta := &traceabilityAgentHealthChecker{
				protocol: test.protocol,
				host:     "somehost.com:543",
				proxyURL: test.proxy,
				timeout:  time.Second * 1,
			}

			job := condorHealthCheckJob{
				agentHealthChecker: ta,
			}

			err := job.Status()
			assert.NotNil(t, err)
			assert.Contains(t, err.Error(), test.errStr)
		})
	}
}
