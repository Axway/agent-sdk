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
	ta := &traceabilityAgentHealthChecker{
		protocol: "https",
		host:     "somehost.com:543",
		proxyURL: "",
	}

	os.Setenv("HTTP_CLIENT_TIMEOUT", "1s")

	job := condorHealthCheckJob{
		agentHealthChecker: ta,
	}
	err := job.Status()
	assert.NotNil(t, err)
}
