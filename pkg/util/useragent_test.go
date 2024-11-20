package util

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatUserAgents(t *testing.T) {
	agentTypeName := "Test"
	agentVersion := "1.0.0"
	sdkVersion := "1.0.0"
	hostname, _ := os.Hostname()
	tests := []struct {
		name              string
		userAgent         string
		expectedUserAgent string
		envName           string
		agentName         string
		isGRPC            bool
	}{
		{
			name:              "test-1",
			envName:           "env",
			isGRPC:            true,
			agentName:         "agent",
			expectedUserAgent: fmt.Sprintf("Test/1.0.0 (sdkVer:1.0.0; env:env; agent:agent; reactive:true; hostname:%s)", hostname),
		},
		{
			name:              "test-2",
			envName:           "env",
			agentName:         "agent",
			expectedUserAgent: fmt.Sprintf("Test/1.0.0 (sdkVer:1.0.0; env:env; agent:agent; reactive:false; hostname:%s)", hostname),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ua := NewUserAgent(
				agentTypeName,
				agentVersion,
				sdkVersion,
				tc.envName,
				tc.agentName,
				tc.isGRPC)
			formattedUserAgent := ua.FormatUserAgent()
			assert.Equal(t, tc.expectedUserAgent, formattedUserAgent)
		})
	}
}

func TestParseUserAgents(t *testing.T) {
	hostname, _ := os.Hostname()
	tests := []struct {
		name       string
		userAgent  string
		expectedUA *CentralUserAgent
	}{
		{
			name:      "test-1",
			userAgent: fmt.Sprintf("Test/1.0.0 (sdkVer:1.0.0; env:env; agent:agent; reactive:true; hostname:%s)", hostname),
			expectedUA: &CentralUserAgent{
				AgentType:           "Test",
				Version:             "1.0.0",
				SDKVersion:          "1.0.0",
				Environment:         "env",
				AgentName:           "agent",
				IsGRPC:              true,
				HostName:            hostname,
				UseGRPCStatusUpdate: true,
			},
		},
		{
			name:      "test-2",
			userAgent: fmt.Sprintf("Test/1.0.0 (sdkVer:1.0.0; env:env; agent:agent; reactive:true; hostname:%s) grpc-go-client-1.1.0", hostname),
			expectedUA: &CentralUserAgent{
				AgentType:           "Test",
				Version:             "1.0.0",
				SDKVersion:          "1.0.0",
				Environment:         "env",
				AgentName:           "agent",
				IsGRPC:              true,
				HostName:            hostname,
				UseGRPCStatusUpdate: true,
			},
		},
		{
			name:      "test-3",
			userAgent: fmt.Sprintf("Test/1.0.0 (sdkVer:1.0.0; env:env; agent:agent; reactive:false; hostname:%s)", hostname),
			expectedUA: &CentralUserAgent{
				AgentType:           "Test",
				Version:             "1.0.0",
				SDKVersion:          "1.0.0",
				Environment:         "env",
				AgentName:           "agent",
				IsGRPC:              false,
				HostName:            hostname,
				UseGRPCStatusUpdate: false,
			},
		},
		{
			name:      "test-4",
			userAgent: "Test/1.0.0 SDK/1.0.0 env agent docker reactive",
			expectedUA: &CentralUserAgent{
				AgentType:           "Test",
				Version:             "1.0.0",
				SDKVersion:          "1.0.0",
				Environment:         "env",
				AgentName:           "agent",
				IsGRPC:              true,
				UseGRPCStatusUpdate: false,
			},
		},
		{
			name:      "test-5",
			userAgent: "Test/1.0.0 SDK/1.0.0 env agent binary",
			expectedUA: &CentralUserAgent{
				AgentType:           "Test",
				Version:             "1.0.0",
				SDKVersion:          "1.0.0",
				Environment:         "env",
				AgentName:           "agent",
				IsGRPC:              false,
				UseGRPCStatusUpdate: false,
			},
		},
		{
			name:       "test-5",
			userAgent:  "invalid user-agent",
			expectedUA: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ua := ParseUserAgent(tc.userAgent)
			if tc.expectedUA == nil {
				assert.Nil(t, ua)
				return
			}
			assert.NotNil(t, ua)
			assert.Equal(t, tc.expectedUA.AgentType, ua.AgentType)
			assert.Equal(t, tc.expectedUA.Version, ua.Version)
			assert.Equal(t, tc.expectedUA.SDKVersion, ua.SDKVersion)
			assert.Equal(t, tc.expectedUA.Environment, ua.Environment)
			assert.Equal(t, tc.expectedUA.AgentName, ua.AgentName)
			assert.Equal(t, tc.expectedUA.IsGRPC, ua.IsGRPC)
			assert.Equal(t, tc.expectedUA.HostName, ua.HostName)
			assert.Equal(t, tc.expectedUA.UseGRPCStatusUpdate, ua.UseGRPCStatusUpdate)
		})
	}
}
