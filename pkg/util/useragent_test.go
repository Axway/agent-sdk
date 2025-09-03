package util

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestFormatUserAgents(t *testing.T) {
	agentTypeName := "Test"
	agentVersion := "1.0.0"
	sdkVersion := "1.0.0"
	hostname, _ := os.Hostname()
	runtimeId := uuid.New().String()
	tests := []struct {
		name              string
		expectedUserAgent string
		envName           string
		agentName         string
		agentVersion      string
		sdkVersion        string
		isGRPC            bool
		runtimeId         string
	}{
		{
			name:              "test-1",
			envName:           "env",
			isGRPC:            true,
			agentName:         "agent",
			agentVersion:      "v1.0.0-125678e",
			sdkVersion:        "v1.1.100",
			expectedUserAgent: fmt.Sprintf("Test/1.0.0-125678e (sdkVer:1.1.100; env:env; agent:agent; reactive:true; hostname:%s; runtimeId:%s)", hostname, runtimeId),
			runtimeId:         runtimeId,
		},
		{
			name:              "test-2",
			envName:           "env",
			isGRPC:            false,
			agentName:         "agent",
			agentVersion:      agentVersion,
			sdkVersion:        sdkVersion,
			expectedUserAgent: fmt.Sprintf("Test/1.0.0 (sdkVer:1.0.0; env:env; agent:agent; reactive:false; hostname:%s; runtimeId:%s)", hostname, runtimeId),
			runtimeId:         runtimeId,
		},
		{
			name:              "test-3",
			envName:           "env",
			isGRPC:            true,
			agentName:         "agent.da.test",
			agentVersion:      agentVersion,
			sdkVersion:        sdkVersion,
			expectedUserAgent: fmt.Sprintf("Test/1.0.0 (sdkVer:1.0.0; env:env; agent:agent.da.test; reactive:true; hostname:%s; runtimeId:%s)", hostname, runtimeId),
			runtimeId:         runtimeId,
		},
		{
			name:              "test-4",
			envName:           "prod",
			isGRPC:            true,
			agentName:         "agent",
			agentVersion:      agentVersion,
			sdkVersion:        sdkVersion,
			expectedUserAgent: fmt.Sprintf("Test/1.0.0 (sdkVer:1.0.0; env:prod; agent:agent; reactive:true; hostname:%s; runtimeId:%s)", hostname, runtimeId),
			runtimeId:         runtimeId,
		},
		{
			name:              "test-5",
			envName:           "env",
			isGRPC:            true,
			agentName:         "agent",
			agentVersion:      agentVersion,
			sdkVersion:        sdkVersion,
			expectedUserAgent: fmt.Sprintf("Test/1.0.0 (sdkVer:1.0.0; env:env; agent:agent; reactive:true; hostname:%s; runtimeId:%s)", hostname, "test-runtimeId-1234567890-abcdefghijklmnopqrstuvwxyz"),
			runtimeId:         "test-runtimeId-1234567890-abcdefghijklmnopqrstuvwxyz",
		},
		{
			name:              "test-6",
			envName:           "env",
			isGRPC:            true,
			agentName:         "",
			agentVersion:      agentVersion,
			sdkVersion:        sdkVersion,
			expectedUserAgent: fmt.Sprintf("Test/1.0.0 (sdkVer:1.0.0; env:env; agent:; reactive:true; hostname:%s; runtimeId:%s)", hostname, "test-runtimeId-empty-agent"),
			runtimeId:         "test-runtimeId-empty-agent",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ua := NewUserAgent(
				agentTypeName,
				tc.agentVersion,
				tc.sdkVersion,
				tc.envName,
				tc.agentName,
				tc.isGRPC,
				tc.runtimeId,
			)
			formattedUserAgent := ua.FormatUserAgent()
			assert.Equal(t, tc.expectedUserAgent, formattedUserAgent,
				"User-Agent should match expected value. Got: %s, Expected: %s", formattedUserAgent, tc.expectedUserAgent)
		})
	}
}

func TestParseUserAgents(t *testing.T) {
	hostname, _ := os.Hostname()
	hostname2 := "test-test.abc.com"
	runtimeId := uuid.New().String()
	tests := []struct {
		name       string
		userAgent  string
		expectedUA *CentralUserAgent
	}{
		{
			name:      "test-1",
			userAgent: fmt.Sprintf("Test/1.0.0-7e7eb72d (sdkVer:1.0.0; env:env; agent:agent; reactive:true; hostname:%s; runtimeId:%s)", hostname, runtimeId),
			expectedUA: &CentralUserAgent{
				AgentType:           "Test",
				Version:             "1.0.0",
				CommitSHA:           "7e7eb72d",
				SDKVersion:          "1.0.0",
				Environment:         "env",
				AgentName:           "agent",
				IsGRPC:              true,
				HostName:            hostname,
				UseGRPCStatusUpdate: true,
				RuntimeId:           runtimeId,
			},
		},
		{
			name:      "test-2",
			userAgent: fmt.Sprintf("Test/1.0.0-7e7eb72d (sdkVer:1.0.0; env:env; agent:agent; reactive:true; hostname:%s)", hostname),
			expectedUA: &CentralUserAgent{
				AgentType:           "Test",
				Version:             "1.0.0",
				CommitSHA:           "7e7eb72d",
				SDKVersion:          "1.0.0",
				Environment:         "env",
				AgentName:           "agent",
				IsGRPC:              true,
				HostName:            hostname,
				UseGRPCStatusUpdate: true,
				RuntimeId:           "",
			},
		},
		{
			name:      "test-3",
			userAgent: fmt.Sprintf("Test/1.0.0 (sdkVer:1.0.0; env:env; agent:agent; reactive:true; hostname:%s; runtimeId:%s) grpc-go/1.65.0", hostname, runtimeId),
			expectedUA: &CentralUserAgent{
				AgentType:           "Test",
				Version:             "1.0.0",
				SDKVersion:          "1.0.0",
				Environment:         "env",
				AgentName:           "agent",
				IsGRPC:              true,
				HostName:            hostname,
				UseGRPCStatusUpdate: true,
				RuntimeId:           runtimeId,
			},
		},
		{
			name:      "test-4",
			userAgent: fmt.Sprintf("Test/1.0.0 (sdkVer:1.0.0; env:env; agent:agent; reactive:true; hostname:%s) grpc-go/1.65.0", hostname),
			expectedUA: &CentralUserAgent{
				AgentType:           "Test",
				Version:             "1.0.0",
				SDKVersion:          "1.0.0",
				Environment:         "env",
				AgentName:           "agent",
				IsGRPC:              true,
				HostName:            hostname,
				UseGRPCStatusUpdate: true,
				RuntimeId:           "",
			},
		},
		{
			name:      "test-5",
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
			name:      "test-6",
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
			name:       "test-7",
			userAgent:  "invalid user-agent",
			expectedUA: nil,
		},
		{
			name:      "test-8",
			userAgent: fmt.Sprintf("WSO2DiscoveryAgent/1.0.0-65a0b4c (sdkVer:1.1.110; env:wso2; agent:wso2-da; reactive:true; hostname:%s; runtimeId:%s)", hostname2, runtimeId),
			expectedUA: &CentralUserAgent{
				AgentType:           "WSO2DiscoveryAgent",
				Version:             "1.0.0",
				CommitSHA:           "65a0b4c",
				SDKVersion:          "1.1.110",
				Environment:         "wso2",
				AgentName:           "wso2-da",
				IsGRPC:              true,
				HostName:            hostname2,
				UseGRPCStatusUpdate: true,
				RuntimeId:           runtimeId,
			},
		},
		{
			name:      "test-9",
			userAgent: fmt.Sprintf("WSO2DiscoveryAgent/1.0.0-65a0b4c (sdkVer:1.1.110; env:wso2; agent:wso2-da; reactive:true; hostname:%s)", hostname2),
			expectedUA: &CentralUserAgent{
				AgentType:           "WSO2DiscoveryAgent",
				Version:             "1.0.0",
				CommitSHA:           "65a0b4c",
				SDKVersion:          "1.1.110",
				Environment:         "wso2",
				AgentName:           "wso2-da",
				IsGRPC:              true,
				HostName:            hostname2,
				UseGRPCStatusUpdate: true,
				RuntimeId:           "",
			},
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
			assert.Equal(t, tc.expectedUA.RuntimeId, ua.RuntimeId)
		})
	}
}
